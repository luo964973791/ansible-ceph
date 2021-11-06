### ansible-playbook 在Centos8部署ceph6.0稳定版本.

```javascript
# 在部署之前全部更改hosts.
[root@node1 ~]# cat /etc/hosts
127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4
::1         localhost localhost.localdomain localhost6 localhost6.localdomain6
172.27.0.6 node1
172.27.0.7 node2
172.27.0.8 node3

#安装依赖.
yum -y install vim-enhanced lrzsz tree bash-completion net-tools wget bzip2 lsof zip unzip gcc make gcc-c++ glibc glibc-devel pcre pcre-devel openssl openssl-devel systemd-devel zlib-devel chrony jq && yum clean all && yum makecache
yum install python36 -y && ln -s /usr/bin/pip3.6 /usr/bin/pip
pip install netaddr pyyaml
```

### 修改复制文件

```javascript
cd ceph-ansible/group_vars/
# grep -Ev '^#|^$' all.yml
---
dummy:
ceph_origin: repository
ceph_repository: community
ceph_mirror: http://mirrors.aliyun.com/ceph
ceph_stable_key: http://mirrors.aliyun.com/ceph/keys/release.asc
ceph_stable_release: pacific
ceph_stable_repo: "{{ ceph_mirror }}/rpm-{{ ceph_stable_release }}"
cephx: true
monitor_interface: eth0          #根据自己的网卡名更改.
public_network: 172.27.0.0/24    #注意这个地方和下面的public_network在生产环境要设置不一样,这里是测试就设置的一样.
cluster_network: "{{ public_network }}"
radosgw_interface: eth0          #根据自己的网卡名更改.
dashboard_enabled: False
```

### 挂载点

```javascript
vi /root/ansible-ceph/group_vars/osds.yml
devices:
  - '/dev/vdb'            #根据自己的网ceph数据磁盘进行更改.
```

### 安装ceph

```javascript
cd /root/ansible-ceph && ansible-playbook -i hosts site.yml
#安装ceph会报错.
vi /usr/sbin/ceph-volume-systemd
#! /usr/bin/env python3   #第一行更改为这个，所有服务器都需要更改



vi /usr/sbin/ceph-volume
#! /usr/bin/env python3  #第一行更改为这个，所有服务器都需要更改



#再进行安装.
ansible-playbook -i hosts site.yml
ceph config set mon auth_allow_insecure_global_id_reclaim false  #HEALTH_WARN解决方法.

```

### 检查状态.

```javascript
#检查
ceph health detail
ceph -s
  cluster:
    id:     4ff55516-ade8-4801-b7bc-ad689bd75efd
    health: HEALTH_OK
 
  services:
    mon: 1 daemons, quorum mon
    mgr: mon(active)
    osd: 2 osds: 2 up, 2 in
 
  data:
    pools:   0 pools, 0 pgs
    objects: 0 objects, 0B
    usage:   2.00GiB used, 37.8GiB / 39.8GiB avail
    pgs:
```

### k8s挂载cephFS

```javascript
mkdir /mnt/cephfs
#挂载
ceph-fuse -m 172.27.0.6:6789,172.27.0.7:6789,172.27.0.8:6789 /mnt/ceph
    
#vi /etc/fstab
none /mnt/ceph fuse.ceph ceph.id=admin,ceph.conf=/etc/ceph/ceph.conf,nonempty,_netdev,defaults 0 0



# 创建命名空间
kubectl create ns cephfs
ceph auth get-key client.admin

#拿到上面的值,创建权限.    
kubectl create secret generic ceph-secret-admin --from-literal=key="AQDyWw9dOUm/FhAA4JCA9PXkPo6+OXpOj9N2ZQ==" -n cephfs

```

### 创建provisioner

```javascript
git clone https://github.com/kubernetes-incubator/external-storage.git
cd external-storage/ceph/cephfs/deploy
NAMESPACE=cephfs
sed -r -i "s/namespace: [^ ]+/namespace: $NAMESPACE/g" ./rbac/*.yaml
sed -r -i "N;s/(name: PROVISIONER_SECRET_NAMESPACE.*\n[[:space:]]*)value:.*/\1value: $NAMESPACE/" ./rbac/deployment.yaml
kubectl -n $NAMESPACE apply -f ./rbac
```

### 创建class

```javascript
[root@ ~]# cat cephfs-class.yaml
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: cephfs
provisioner: ceph.com/cephfs
parameters:
    monitors: 172.27.0.6:6789,172.27.0.7:6789,172.27.0.8:6789
    adminId: admin
    adminSecretName: ceph-secret-admin
    adminSecretNamespace: cephfs
    claimRoot: /volumes/kubernetes
```

### pvc

```javascript
[root@ ~]# cat cephfs-pvc.yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: cephfs-pvc
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 8Gi
  storageClassName: cephfs
```

### 创建pod.yaml

```javascript
[root@ ~]# cat nginx.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 2
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        imagePullPolicy: IfNotPresent
        ports:
        - name: http
          containerPort: 80
        - name: https
          containerPort: 443
        volumeMounts:
        - name: cephfs-pvc
          mountPath: /usr/share/nginx/html
      volumes:
      - name: cephfs-pvc
        persistentVolumeClaim:
          claimName: cephfs-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
spec:
  type: NodePort
  ports:
  - name: nginx
    port: 80
    protocol: TCP
    targetPort: 80
    nodePort: 32668
  selector:
    app: nginx
```


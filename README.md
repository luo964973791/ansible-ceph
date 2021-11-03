### ansible-playbook 在Centos8部署ceph

```javascript
# 在部署之前全部更改hosts,比如
hostnamectl set-hostname node1
hostnamectl set-hostname node2
hostnamectl set-hostname node3

yum install epel-release -y
yum install python3-pip git -y
pip3 install --upgrade pip
pip3 install --upgrade setuptools
cd /root && git clone https://github.com/luo964973791/ansible-ceph.git
cd ansible-ceph
pip3 install -r requirements.txt
```

### 准备hosts文件

```javascript
vi /root/ansible-ceph/hosts
[mons]
172.27.0.6
172.27.0.7
172.27.0.8

[mgrs]
172.27.0.6
172.27.0.7
172.27.0.8

[osds]
172.27.0.6
172.27.0.7
172.27.0.8

[rgws]
172.27.0.6
172.27.0.7
172.27.0.8

[mdss]
172.27.0.6
172.27.0.7
172.27.0.8

[grafana-server]
172.27.0.6
172.27.0.7
172.27.0.8

[clients]
172.27.0.6
172.27.0.7
172.27.0.8
```

### 修改复制文件

```javascript
cd /root/ansible-ceph
cp group_vars/clients.yml.sample group_vars/clients.yml
cp group_vars/mons.yml.sample group_vars/mons.yml
cp group_vars/mgrs.yml.sample group_vars/mgrs.yml
cp group_vars/rgws.yml.sample group_vars/rgws.yml
cp group_vars/all.yml.sample group_vars/all.yml
cp group_vars/osds.yml.sample group_vars/osds.yml
cp site.yml.sample site.yml

vi group_vars/all.yml
# cd ceph-ansible/group_vars/
# mv all.yml.sample all.yml
# grep -Ev '^#|^$' all.yml
---
dummy:
ceph_repository_type: repository
ceph_origin: repository
ceph_repository: community
ceph_mirror: https://mirrors.tuna.tsinghua.edu.cn/ceph/
ceph_stable_key: https://mirrors.tuna.tsinghua.edu.cn/ceph/keys/release.asc
ceph_stable_release: octopus
ceph_stable_repo: https://mirrors.tuna.tsinghua.edu.cn/ceph/rpm-15.2.7/el7/
monitor_interface: eth0
public_network: 172.27.0.0/24
cluster_network: 172.16.0.0/16
osd_objectstore: bluestore
radosgw_civetweb_port: 8080
radosgw_interface: eth0
dashboard_enabled: True
dashboard_admin_user: admin
dashboard_admin_password: admin
grafana_admin_user: admin
grafana_admin_password: admin
```

### 挂载点

```javascript
vi /root/ansible-ceph/group_vars/osds.yml
devices:
  - '/dev/vdb'
osd_scenario: collocated
```

### 修改配置

```javascript
cd /root/ansible-ceph && cp site.yml.sample site.yml
vi site.yml
- hosts:
  - mons
  - agents
  - osds
  - mdss
  - rgws
  - nfss
  - restapis
  - rbdmirrors
  - clients
  - mgrs
  - iscsigws
  - iscsi-gws
```

### 安装ceph

```javascript
cd /root/ansible-ceph && ansible-playbook -i hosts site.yml
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


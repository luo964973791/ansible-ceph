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
yum install python36 -y && pip3 install --upgrade pip && pip3 install --upgrade setuptools
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
cd /root/ansible-ceph && pip3 install -r requirements.txt && ansible-playbook -i hosts site.yml
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
git clone https://github.com/ceph/ceph-csi.git
cd ceph-csi && git checkout v3.3.1
cd ./deploy/cephfs/kubernetes

vi csi-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "3887d53c-2433-46b7-b43f-7054437ac829",  #ceph -s查看clusterID
        "monitors": [
          "172.27.0.6:6789",
          "172.27.0.7:6789",
          "172.27.0.8:6789"
        ]
      }
    ]
metadata:
  name: ceph-csi-config
  
  
  

csi-provisioner-rbac.yaml  csi-nodeplugin-rbac.yaml      #里面的命名空间改为ceph-csi  
sed -i 's/namespace: default/namespace: ceph-csi/g' csi-provisioner-rbac.yaml
sed -i 's/namespace: default/namespace: ceph-csi/g' csi-nodeplugin-rbac.yaml
NAMESPACE=ceph-csi
sed -r -i "s/namespace: [^ ]+/namespace: $NAMESPACE/g" ./*.yaml
sed -r -i "N;s/(name: PROVISIONER_SECRET_NAMESPACE.*\n[[:space:]]*)value:.*/\1value: $NAMESPACE/" ./*.yaml




#创建ceph的命名空间 ceph的想东西都部署在此命名空间中
kubectl create ns ceph-csi
kubectl apply -f csi-config-map.yaml -n ceph-csi
kubectl create -f csi-provisioner-rbac.yaml -n ceph-csi
kubectl create -f csi-nodeplugin-rbac.yaml -n ceph-csi
kubectl create -f csi-cephfsplugin-provisioner.yaml -n ceph-csi
kubectl create -f csi-cephfsplugin.yaml -n ceph-csi

#创建密钥.
cd ceph-csi/examples/cephfs
vim secret.yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: csi-cephfs-secret
  namespace: ceph
stringData:
  # 通过ceph auth get client.admin查看
  # Required for statically provisioned volumes
  userID: admin              
  userKey: AQBgCIpg8OC9LRAAcl8XOfU9/71WiZNLGgnjgA==
  
  # Required for dynamically provisioned volumes
  adminID: admin
  adminKey: AQBgCIpg8OC9LRAAcl8XOfU9/71WiZNLGgnjgA==
    
kubectl apply -f secret.yaml

vim storageclass.yaml

---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-cephfs-sc
provisioner: cephfs.csi.ceph.com
parameters:
  # (required) String representing a Ceph cluster to provision storage from.
  # Should be unique across all Ceph clusters in use for provisioning,
  # cannot be greater than 36 bytes in length, and should remain immutable for
  # the lifetime of the StorageClass in use.
  # Ensure to create an entry in the configmap named ceph-csi-config, based on
  # csi-config-map-sample.yaml, to accompany the string chosen to
  # represent the Ceph cluster in clusterID below
  clusterID: 71057bd8-9a18-42ea-95e0-7596901370fe  #此处就是填写上面的clusterID
  
  # (required) CephFS filesystem name into which the volume shall be created
  # eg: fsName: myfs
  fsName: cephfs
  
  # (optional) Ceph pool into which volume data shall be stored
  # pool: <cephfs-data-pool>
  
  # (optional) Comma separated string of Ceph-fuse mount options.
  # For eg:
  # fuseMountOptions: debug
  
  # (optional) Comma separated string of Cephfs kernel mount options.
  # Check man mount.ceph for mount options. For eg:
  # kernelMountOptions: readdir_max_bytes=1048576,norbytes
  
  # The secrets have to contain user and/or Ceph admin credentials.
  # 注意，这里的命名空间都改为ceph
  csi.storage.k8s.io/provisioner-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/provisioner-secret-namespace: ceph-csi 
  csi.storage.k8s.io/controller-expand-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/controller-expand-secret-namespace: ceph-csi
  csi.storage.k8s.io/node-stage-secret-name: csi-cephfs-secret
  csi.storage.k8s.io/node-stage-secret-namespace: ceph-csi
  
  # (optional) The driver can use either ceph-fuse (fuse) or
  # ceph kernelclient (kernel).
  # If omitted, default volume mounter will be used - this is
  # determined by probing for ceph-fuse and mount.ceph
  # mounter: kernel
  
  # (optional) Prefix to use for naming subvolumes.
  # If omitted, defaults to "csi-vol-".
  # volumeNamePrefix: "foo-bar-"
  
reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
  - discard
    
kubectl apply -f storageclass.yaml
kubectl get sc
NAME                    PROVISIONER           RECLAIMPOLICY   VOLUMEBINDINGMODE   ALLOWVOLUMEEXPANSION   AGE
csi-cephfs-sc           cephfs.csi.ceph.com   Delete          Immediate           true                   4s


kubectl apply -f pvc.yaml
kubectl get pvc
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS    AGE
csi-cephfs-pvc                 Bound    pvc-5c5fb10a-c8db-48da-b71c-b1cefc9ebb6e   1Gi        RWX            csi-cephfs-sc   18s



mkdir /mnt/cephfs
#挂载
ceph-fuse -m 172.27.0.6:6789,172.27.0.7:6789,172.27.0.8:6789 /mnt/ceph
    
#vi /etc/fstab
none /mnt/ceph fuse.ceph ceph.id=admin,ceph.conf=/etc/ceph/ceph.conf,nonempty,_netdev,defaults 0 0

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
          claimName: csi-cephfs-sc
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


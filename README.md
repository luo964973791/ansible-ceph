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
    
    

yum install ceph-fuse -y
mkdir /mnt/ceph
#挂载
ceph-fuse -m 172.27.0.6:6789,172.27.0.7:6789,172.27.0.8:6789 /mnt/ceph
    
#vi /etc/fstab
none /mnt/ceph fuse.ceph ceph.id=admin,ceph.conf=/etc/ceph/ceph.conf,nonempty,_netdev,defaults 0 0
```

### k8s挂载cephFS

```javascript
cd ansible-ceph/ceph-csi
cd ./deploy/cephfs/kubernetes

vi csi-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "3887d53c-2433-46b7-b43f-7054437ac829",
        "monitors": [
          "172.27.0.6:6789",
          "172.27.0.7:6789",
          "172.27.0.8:6789"
        ]
      }
    ]
metadata:
  name: ceph-csi-config
  
#创建cephfs的命名空间 ceph的想东西都部署在此命名空间中
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
  namespace: ceph-csi
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

```

### 挂载rbd
```javascript
cd ansible-ceph/ceph-csi
cd ./deploy/rbd/kubernetes

vi csi-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "3887d53c-2433-46b7-b43f-7054437ac829",
        "monitors": [
          "172.27.0.6:6789",
          "172.27.0.7:6789",
          "172.27.0.8:6789"
        ]
      }
    ]
metadata:
  name: ceph-csi-config
  
  
kubectl apply -f csi-config-map.yaml -n rbd-provisioner
kubectl apply -f csi-provisioner-rbac.yaml -n rbd-provisioner
kubectl apply -f csi-nodeplugin-rbac.yaml -n rbd-provisioner
kubectl apply -f csi-rbdplugin-provisioner.yaml -n rbd-provisioner
kubectl apply -f csi-rbdplugin.yaml -n rbd-provisioner  
  

ceph osd pool create k8s 32 32
rbd pool init k8s
ceph auth get-or-create client.k8s mon 'profile rbd' osd 'profile rbd pool=k8s' mgr 'profile rbd pool=k8s'

cd ceph-csi/examples/rbd
vim secret.yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: csi-rbd-secret
  namespace: rbd-provisioner
stringData:
  # Key values correspond to a user name and its key, as defined in the
  # ceph cluster. User ID should have required access to the 'pool'
  # specified in the storage class
  userID: admin
  userKey: AQBgCIpg8OC9LRAAcl8XOfU9/71WiZNLGgnjgA==
  
  # Encryption passphrase
  encryptionPassphrase: test_passphrase
kubectl apply -f secret.yaml


vim storageclass.yaml
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: csi-rbd-sc
provisioner: rbd.csi.ceph.com
# If topology based provisioning is desired, delayed provisioning of
# PV is required and is enabled using the following attribute
# For further information read TODO<doc>
# volumeBindingMode: WaitForFirstConsumer
parameters:
   # (required) String representing a Ceph cluster to provision storage from.
   # Should be unique across all Ceph clusters in use for provisioning,
   # cannot be greater than 36 bytes in length, and should remain immutable for
   # the lifetime of the StorageClass in use.
   # Ensure to create an entry in the configmap named ceph-csi-config, based on
   # csi-config-map-sample.yaml, to accompany the string chosen to
   # represent the Ceph cluster in clusterID below
   clusterID: 71057bd8-9a18-42ea-95e0-7596901370fe  #此处就是填写上面的clusterID
  
   # (optional) If you want to use erasure coded pool with RBD, you need to
   # create two pools. one erasure coded and one replicated.
   # You need to specify the replicated pool here in the `pool` parameter, it is
   # used for the metadata of the images.
   # The erasure coded pool must be set as the `dataPool` parameter below.
   # dataPool: <ec-data-pool>
  
   # (required) Ceph pool into which the RBD image shall be created
   # eg: pool: rbdpool
   pool: k8s  #填写上面的存储池
  
   # Set thickProvision to true if you want RBD images to be fully allocated on
   # creation (thin provisioning is the default).
   thickProvision: "false"
   # (required) RBD image features, CSI creates image with image-format 2
   # CSI RBD currently supports `layering`, `journaling`, `exclusive-lock`
   # features. If `journaling` is enabled, must enable `exclusive-lock` too.
   # imageFeatures: layering,journaling,exclusive-lock
   imageFeatures: layering
  
   # (optional) mapOptions is a comma-separated list of map options.
   # For krbd options refer
   # https://docs.ceph.com/docs/master/man/8/rbd/#kernel-rbd-krbd-options
   # For nbd options refer
   # https://docs.ceph.com/docs/master/man/8/rbd-nbd/#options
   # mapOptions: lock_on_read,queue_depth=1024
  
   # (optional) unmapOptions is a comma-separated list of unmap options.
   # For krbd options refer
   # https://docs.ceph.com/docs/master/man/8/rbd/#kernel-rbd-krbd-options
   # For nbd options refer
   # https://docs.ceph.com/docs/master/man/8/rbd-nbd/#options
   # unmapOptions: force
  
   # The secrets have to contain Ceph credentials with required access
   # to the 'pool'.
   csi.storage.k8s.io/provisioner-secret-name: csi-rbd-secret
   csi.storage.k8s.io/provisioner-secret-namespace: rbd-provisioner
   csi.storage.k8s.io/controller-expand-secret-name: csi-rbd-secret
   csi.storage.k8s.io/controller-expand-secret-namespace: rbd-provisioner
   csi.storage.k8s.io/node-stage-secret-name: csi-rbd-secret
   csi.storage.k8s.io/node-stage-secret-namespace: rbd-provisioner
  
   # (optional) Specify the filesystem type of the volume. If not specified,
   # csi-provisioner will set default as `ext4`.
   csi.storage.k8s.io/fstype: ext4
  
   # (optional) uncomment the following to use rbd-nbd as mounter
   # on supported nodes
   # mounter: rbd-nbd
  
   # (optional) Prefix to use for naming RBD images.
   # If omitted, defaults to "csi-vol-".
   # volumeNamePrefix: "foo-bar-"
  
   # (optional) Instruct the plugin it has to encrypt the volume
   # By default it is disabled. Valid values are "true" or "false".
   # A string is expected here, i.e. "true", not true.
   # encrypted: "true"
  
   # (optional) Use external key management system for encryption passphrases by
   # specifying a unique ID matching KMS ConfigMap. The ID is only used for
   # correlation to configmap entry.
   # encryptionKMSID: <kms-config-id>
  
   # Add topology constrained pools configuration, if topology based pools
   # are setup, and topology constrained provisioning is required.
   # For further information read TODO<doc>
   # topologyConstrainedPools: |
   #   [{"poolName":"pool0",
   #     "dataPool":"ec-pool0" # optional, erasure-coded pool for data
   #     "domainSegments":[
   #       {"domainLabel":"region","value":"east"},
   #       {"domainLabel":"zone","value":"zone1"}]},
   #    {"poolName":"pool1",
   #     "dataPool":"ec-pool1" # optional, erasure-coded pool for data
   #     "domainSegments":[
   #       {"domainLabel":"region","value":"east"},
   #       {"domainLabel":"zone","value":"zone2"}]},
   #    {"poolName":"pool2",
   #     "dataPool":"ec-pool2" # optional, erasure-coded pool for data
   #     "domainSegments":[
   #       {"domainLabel":"region","value":"west"},
   #       {"domainLabel":"zone","value":"zone1"}]}
   #   ]
  
reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
   - discard



kubectl apply -f storageclass.yaml
kubectl describe sts nginx #查看挂载是否成功.
```

### 创建pod.yaml

```javascript
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: web
spec:
  selector:
    matchLabels:
      app: nginx
  serviceName: "nginx"
  replicas: 3
  template:
    metadata:
      labels:
        app: nginx
    spec:
      terminationGracePeriodSeconds: 10
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
          name: web
        volumeMounts:
        - name: www
          mountPath: /usr/share/nginx/html
  volumeClaimTemplates:
  - metadata:
      name: www
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "csi-rbd-sc"
      persistentVolumeReclaimPolicy: Recycle
      resources:
        requests:
          storage: 1Gi
```


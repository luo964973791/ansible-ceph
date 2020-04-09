### ansible-playbook 在Centos7.6 Centos7.7部署ceph

```javascript
# 在部署之前全部更改hosts,比如
hostnamectl set-hostname node1
hostnamectl set-hostname node2
hostnamectl set-hostname node3

yum install epel-release -y
yum install python-pip git -y
pip install ansible==2.4.2
pip install --upgrade pip
pip install --upgrade setuptools
cd /root && git clone https://github.com/luo964973791/ansible-ceph.git
```

### 准备hosts文件

```javascript
vi /root/ansible-ceph/hosts
[mons]
172.16.68.11
172.16.68.10
172.16.68.9

[osds]
172.16.68.11
172.16.68.10
172.16.68.9

[mgrs]
172.16.68.11
172.16.68.10
172.16.68.9

[mdss]
172.16.68.11
172.16.68.10
172.16.68.9

[clients]
172.16.68.11
172.16.68.10
172.16.68.9
172.16.68.8
172.16.68.7
172.16.68.6
```

### 修改复制文件

```javascript
cd /root/ansible-ceph
cp group_vars/all.yml.sample group_vars/all.yml
cp group_vars/osds.yml.sample group_vars/osds.yml
cp site.yml.sample site.yml

vi group_vars/all.yml
ceph_origin: repository
ceph_repository: community
ceph_mirror: http://mirrors.aliyun.com/ceph
ceph_stable_key: http://mirrors.aliyun.com/ceph/keys/release.asc
ceph_stable_release: luminous
ceph_stable_repo: "{{ ceph_mirror }}/rpm-{{ ceph_stable_release }}"
fsid: 54d55c64-d458-4208-9592-36ce881cbcb7 ##通过uuidgen生成
generate_fsid: false
public_network: "10.0.0.0/24"
cluster_network: "10.0.0.0/24"
monitor_interface: eth0
journal_size: 1024
radosgw_interface: eth0
```

### 挂载点

```javascript
vi /root/ansible-ceph/group_vars/osds.yml
devices:
  - '/dev/vdb'
osd_scenario: collocated
osd_objectstore: bluestore
```

### 修改配置

```javascript
cd /root/ansible-ceph && cp site.yml.sample site.yml
vi site.yml
- hosts:
  - mons
#  - agents
  - osds
  - mdss
#  - rgws
#  - nfss
#  - restapis
#  - rbdmirrors
  - clients
  - mgrs
#  - iscsigws
#  - iscsi-gws
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

### k8s使用ceph必须更改镜像.

```javascript
# 更改 vi /etc/kubernetes/manifests/kube-controller-manager.yaml
image: k8s.gcr.io/kube-controller-manager:v1.18.0
to
image: gcr.io/google_containers/hyperkube:v1.18.0
in
```

### k8s挂载cephFS

```javascript
mkdir /mnt/cephfs
#挂载
ceph-fuse -m 172.27.0.6:6789,172.27.0.7:6789,172.27.0.8:6789 /mnt/ceph
    
#vi /etc/fstab
none /mnt/ceph fuse.ceph ceph.id=admin,ceph.conf=/etc/ceph/ceph.conf,nonempty,_netdev,defaults 0 0
```

### 生成加密key

```javascript
grep key /etc/ceph/ceph.client.admin.keyring |awk '{printf "%s", $NF}'|base64
```

### 创建ceph的secret

```javascript
[root@ ~]# cat ceph-secret.yaml 
apiVersion: v1
kind: Secret
metadata:
  name: ceph-secret
data:
  key: QVFDL2ZYMWVqRUd3S3hBQTlmSFBuekZSQ3dMYjROdHFtRWI5U0E9PQ==
```

### 创建存储类

```javascript
[root@ ~]# cat cephfs-pvc.yaml 
apiVersion: v1
kind: PersistentVolume
metadata:
  name: cephfs-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteMany
  cephfs:
    monitors:
      - 172.27.0.6:6789
      - 172.27.0.7:6789
      - 172.27.0.8:6789
    user: admin
    secretRef:
      name: ceph-secret
    readOnly: false
  persistentVolumeReclaimPolicy: Recycle
```

### 创建PersistentVolumeClaim

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
      storage: 10Gi
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
    nodePort: 32666
  selector:
    app: nginx
```


### 一、rbd块存储创建ceph pool

```javascript
ceph osd pool create kubernetes
rbd pool init kubernetes
ceph auth get-or-create client.kubernetes mon 'profile rbd' osd 'profile rbd pool=kubernetes' mgr 'profile rbd pool=kubernetes'
ceph auth get client.kubernetes
ceph mon dump
kubectl create ns ceph-csi
```

### 二、rbd块存储创建csi-config-map

```javascript
#"clusterID":  ceph -s查看
cat <<EOF | sudo tee csi-config-map.yaml
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "6f4dc742-c57f-400a-8470-33a616c88b56",
        "monitors": ["172.27.0.3:6789","172.27.0.4:6789","172.27.0.5:6789"]
      }
    ]
metadata:
  name: ceph-csi-config
  namespace: ceph-csi
EOF

kubectl apply -f csi-config-map.yaml
```

### 三、rbd块存储创建csi-kms-config-map

```javascript
cat <<EOF | sudo tee csi-kms-config-map.yaml 
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    {}
metadata:
  name: ceph-csi-encryption-kms-config
  namespace: ceph-csi
EOF

kubectl apply -f csi-kms-config-map.yaml
```

### 四、rbd块存储创建ceph pool

```javascript
#fsid 使用ceph -s查看
cat <<EOF | sudo tee ceph-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  ceph.conf: |
    [global]
    auth_cluster_required = cephx
    auth_service_required = cephx
    auth_client_required = cephx
  # keyring is a required key and its value should be empty
  keyring: |
metadata:
  name: ceph-config
  namespace: ceph-csi
EOF

kubectl apply -f ceph-config-map.yaml
```

### 五、rbd块存储创建ceph-rbd-secret.yaml

```javascript
#userKey使用 ceph auth list | grep -A 4 client.kubernetes
cat <<EOF | sudo tee csi-rbd-secret.yaml 
---
apiVersion: v1
kind: Secret
metadata:
  name: csi-rbd-secret
  namespace: ceph-csi
stringData:
  userID: kubernetes
  userKey: AQBSocdjboeeLhAABFqGOe+I2v3jgtiPwyFbMQ==
EOF

kubectl apply -f csi-rbd-secret.yaml
```

### 六、rbd块存储下载ceph-csi插件

```javascript
mkdir ceph-csi
cd ceph-csi
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-provisioner-rbac.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-nodeplugin-rbac.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-rbdplugin-provisioner.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-rbdplugin.yaml
sed -i "s/namespace: default/namespace: ceph-csi/g" $(grep -rl "namespace: default" ./)
kubectl apply -f . -n ceph-csi
```

### 七、rbd块存储创建ceph pool

```javascript
#"clusterID":  ceph -s查看
cat <<EOF | sudo tee csi-rbd-sc.yaml 
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: csi-rbd-sc
   namespace: ceph-csi
   annotations:
     storageclass.beta.kubernetes.io/is-default-class: "true"
     storageclass.kubesphere.io/supported-access-modes: '["ReadWriteOnce","ReadOnlyMany","ReadWriteMany"]'
provisioner: rbd.csi.ceph.com
parameters:
   clusterID: 6f4dc742-c57f-400a-8470-33a616c88b56
   pool: kubernetes
   imageFeatures: layering
   csi.storage.k8s.io/provisioner-secret-name: csi-rbd-secret
   csi.storage.k8s.io/provisioner-secret-namespace: ceph-csi
   csi.storage.k8s.io/controller-expand-secret-name: csi-rbd-secret
   csi.storage.k8s.io/controller-expand-secret-namespace: ceph-csi
   csi.storage.k8s.io/node-stage-secret-name: csi-rbd-secret
   csi.storage.k8s.io/node-stage-secret-namespace: ceph-csi
reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
   - discard
EOF

kubectl apply -f csi-rbd-sc.yaml
```

### 八、rbd块存储测试

```javascript
cat <<EOF | sudo tee pvc.yaml 
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: rbd-pvc
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: csi-rbd-sc
EOF

kubectl apply -f pvc.yaml
```













### 一、cephfs文件存储创建csi-config-map

```javascript
#"clusterID":  ceph -s查看
cat <<EOF | sudo tee csi-config-map.yaml
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    [
      {
        "clusterID": "88816657-ed2d-487e-9c51-1aa6e97f5aee",
        "monitors": ["172.27.0.3:6789","172.27.0.4:6789","172.27.0.5:6789"]
      }
    ]
metadata:
  name: ceph-csi-config
  namespace: ceph-csi
EOF

kubectl apply -f csi-config-map.yaml
```

### 二、cephfs文件存储创建csi-kms-config-map

```javascript
cat <<EOF | sudo tee csi-kms-config-map.yaml 
---
apiVersion: v1
kind: ConfigMap
data:
  config.json: |-
    {}
metadata:
  name: ceph-csi-encryption-kms-config
  namespace: ceph-csi
EOF

kubectl apply -f csi-kms-config-map.yaml
```

### 三、文件存储创建ceph pool

```javascript
#fsid 使用ceph -s查看
cat <<EOF | sudo tee ceph-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  ceph.conf: |
    [global]
    auth_cluster_required = cephx
    auth_service_required = cephx
    auth_client_required = cephx
  # keyring is a required key and its value should be empty
  keyring: |
metadata:
  name: ceph-config
  namespace: ceph-csi
EOF

kubectl apply -f ceph-config-map.yaml
```

### 四、cephfs文件存储创建secret认证

```javascript
cat <<EOF | sudo tee secret.yaml 
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
  userKey: AQB+gX9iXsFoOxAAP/L8vUstJ6f63vlrrap7aw==
  
  # Required for dynamically provisioned volumes
  adminID: admin
  adminKey: AQB+gX9iXsFoOxAAP/L8vUstJ6f63vlrrap7aw==
EOF

kubectl apply -f secret.yaml
```

### 五、cephfs文件存储部署

```javascript
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/cephfs/kubernetes/csi-cephfsplugin.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/cephfs/kubernetes/csi-cephfsplugin-provisioner.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/cephfs/kubernetes/csi-nodeplugin-rbac.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/cephfs/kubernetes/csi-provisioner-rbac.yaml
sed -i "s/namespace: default/namespace: ceph-csi/g" $(grep -rl "namespace: default" ./)
kubectl apply -f .
```

### 六、cephfs文件存储创建storageclass

```javascript
cat <<EOF | sudo tee storageclass.yaml 
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
  clusterID: 38bab61c-0e52-461f-a9ee-1c713dfc75a3   #ceph -s查看ID

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
  - debug
EOF

kubectl -n ceph-csi apply -f storageclass.yaml
```

### 七、cephfs创建pvc测试.

```javascript
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/examples/cephfs/pvc.yaml
kubectl apply -f pvc.yaml
```


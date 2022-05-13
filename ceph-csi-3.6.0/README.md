### 一、创建ceph pool

```javascript
ceph osd pool create kubernetes
rbd pool init kubernetes
ceph auth get-or-create client.kubernetes mon 'profile rbd' osd 'profile rbd pool=kubernetes' mgr 'profile rbd pool=kubernetes'

ceph auth list | grep -A 4 client.kubernetes
ceph mon dump   #查看集群信息
kubectl create ns ceph
```

### 一、创建csi-config-map

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
  namespace: ceph
EOF

kubectl apply -f csi-config-map.yaml
```

### 三、创建csi-kms-config-map

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
  namespace: ceph
EOF

kubectl apply -f csi-kms-config-map.yaml
```

### 一、创建ceph pool

```javascript
#fsid 使用ceph -s查看
cat <<EOF | sudo tee ceph-config-map.yaml
---
apiVersion: v1
kind: ConfigMap
data:
  ceph.conf: |
    [global]
    fsid = 88816657-ed2d-487e-9c51-1aa6e97f5aee
    public_network = 172.27.0.0/24
    cluster_network = 172.27.0.0/24
    mon_initial_members = node1,node2,node3
    mon_host = 172.27.0.3,172.27.0.4,172.27.0.5
    auth_cluster_required = cephx
    auth_service_required = cephx
    auth_client_required = cephx
    mon_allow_pool_delete = true
  # keyring is a required key and its value should be empty
  keyring: |
metadata:
  name: ceph-config
  namespace: ceph
EOF

kubectl apply -f ceph-config-map.yaml
```

### 五、创建ceph-rbd-secret.yaml

```javascript
#userKey使用 ceph auth list | grep -A 4 client.kubernetes
cat <<EOF | sudo tee csi-rbd-secret.yaml 
---
apiVersion: v1
kind: Secret
metadata:
  name: csi-rbd-secret
  namespace: ceph
stringData:
  userID: kubernetes
  userKey: AQB7AH5izD6SKRAA54v0BY6HIwviMq8Gmpnz1A==
EOF

kubectl apply -f csi-rbd-secret.yaml
```

### 六、下载ceph-csi插件

```javascript
mkdir ceph-csi
cd ceph-csi
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-provisioner-rbac.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-nodeplugin-rbac.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-rbdplugin-provisioner.yaml
wget https://raw.githubusercontent.com/ceph/ceph-csi/master/deploy/rbd/kubernetes/csi-rbdplugin.yaml
sed -i 's/namespace: default/namespace: ceph/g' ./*.yaml

kubectl apply -f .
```

### 七、创建ceph pool

```javascript
#"clusterID":  ceph -s查看
cat <<EOF | sudo tee csi-rbd-sc.yaml 
---
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
   name: csi-rbd-sc
   namespace: ceph
   annotations:
     storageclass.beta.kubernetes.io/is-default-class: "true"
     storageclass.kubesphere.io/supported-access-modes: '["ReadWriteOnce","ReadOnlyMany","ReadWriteMany"]'
provisioner: rbd.csi.ceph.com
parameters:
   clusterID: 88816657-ed2d-487e-9c51-1aa6e97f5aee
   pool: kubernetes
   imageFeatures: layering
   csi.storage.k8s.io/provisioner-secret-name: csi-rbd-secret
   csi.storage.k8s.io/provisioner-secret-namespace: ceph
   csi.storage.k8s.io/controller-expand-secret-name: csi-rbd-secret
   csi.storage.k8s.io/controller-expand-secret-namespace: ceph
   csi.storage.k8s.io/node-stage-secret-name: csi-rbd-secret
   csi.storage.k8s.io/node-stage-secret-namespace: ceph
reclaimPolicy: Delete
allowVolumeExpansion: true
mountOptions:
   - discard
EOF

kubectl apply -f csi-rbd-sc.yaml
```

### 八、测试

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


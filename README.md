### ansible-playbook部署ceph

```javascript
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
radosgw_interface: eth0
cephx: true
ceph_conf_overrides:
    global:
      rbd_default_features: 7
      auth cluster required: cephx
      auth service required: cephx
      auth client required: cephx
      osd journal size: 2048
      osd pool default size: 3
      osd pool default min size: 1
      mon_pg_warn_max_per_osd: 1024
      osd pool default pg num: 128
      osd pool default pgp num: 128
      max open files: 131072
      osd_deep_scrub_randomize_ratio: 0.01

    mgr:
      mgr modules: dashboard

    mon:
      mon_allow_pool_delete: true

    client:
      rbd_cache: true
      rbd_cache_size: 335544320
      rbd_cache_max_dirty: 134217728
      rbd_cache_max_dirty_age: 10

    osd:
      osd mkfs type: xfs
    # osd mount options xfs: "rw,noexec,nodev,noatime,nodiratime,nobarrier"
      ms_bind_port_max: 7100
      osd_client_message_size_cap: 2147483648
      osd_crush_update_on_start: true
      osd_deep_scrub_stride: 131072
      osd_disk_threads: 4
      osd_map_cache_bl_size: 128
      osd_max_object_name_len: 256
      osd_max_object_namespace_len: 64
      osd_max_write_size: 1024
      osd_op_threads: 8

      osd_recovery_op_priority: 1
      osd_recovery_max_active: 1
      osd_recovery_max_single_start: 1
      osd_recovery_max_chunk: 1048576
      osd_recovery_threads: 1
      osd_max_backfills: 4
      osd_scrub_begin_hour: 23
      osd_scrub_end_hour: 7
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
ceph osd pool create rbd 128
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

### 添加dashboard

```javascript
#ceph mgr module enable dashboard
#ceph mgr dump
{
    "epoch": 10,
    "active_gid": 4136,
    "active_name": "mon",
    "active_addr": "10.0.0.6:6800/18798",
    "available": true,
    "standbys": [],
    "modules": [
        "dashboard",
        "status"
    ],
    "available_modules": [
        "balancer",
        "dashboard",
        "influx",
        "localpool",
        "prometheus",
        "restful",
        "selftest",
        "status",
        "zabbix"
    ],
    "services": {
        "dashboard": "http://mon:7000/"
    }
}
```

### 清空集群
```javascript
cd /root/ansible-ceph && cp infrastructure-playbooks/purge-cluster.yml purge-cluster.yml # 必须copy到项目根目录下
ansible-playbook -i hosts purge-cluster.yml
```



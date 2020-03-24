### ansible-playbook部署ceph

```javascript
yum install python-pip -y
cd  /root && git clone https://github.com/ceph/ceph-ansible.git
cd /root/ceph-ansible && git checkout stable-3.2
cd /root/ceph-ansible && pip install -r requirements.txt
```

### 准备hosts文件

```javascript
vi /root/ceph_ansible/hosts
[mons]
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

[mgrs]
172.27.0.6
172.27.0.7
172.27.0.8
```

### 修改复制文件

```javascript
cd /root/ceph-ansible/group_vars && cp all.yml.sample all.yml
vi all.yaml
ceph_origin: repository
ceph_repository: community
ceph_stable_release: luminous
public_network: "10.0.0.0/24"
cluster_network: "10.0.0.0/24"
monitor_interface: eth0
radosgw_interface: eth0
```

### 挂载点

```javascript
cd /root/ceph-ansible/group_vars && cp osds.yml.sample osds.yml
vi /root/ceph-ansible/group_vars/osds.yml
devices:
  - '/dev/vdb'
osd_scenario: collocated
```

### 修改配置

```javascript
cd /root/ceph-ansible && cp site.yml.sample site.yml
vi site.yml
- hosts:
  - mons
  #- agents
  - osds
  #- mdss
  #- rgws
  #- nfss
  #- restapis
  #- rbdmirrors
  #- clients
  - mgrs
  #- iscsigws
  #- iscsi-gws
```

### 安装ceph

```javascript
cd /root/ceph-ansible && ansible-playbook -i hosts site.yml
```

### 检查状态.

```javascript
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


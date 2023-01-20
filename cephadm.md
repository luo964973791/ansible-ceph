```javascript
yum install python3-pip -y
yum install docker-ce -y && systemctl start docker && systemctl enable docker
curl --silent --remote-name --location https://github.com/ceph/ceph/raw/octopus/src/cephadm/cephadm  -o cephadm
chmod +x cephadm
./cephadm install ceph-common ceph    #安装客户端
./cephadm add-repo --release octopus  #配置yum源
#所有服务器统一修改hosts
[root@localhost ~]# cat /etc/hosts  
127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4
::1         localhost localhost.localdomain localhost6 localhost6.localdomain6
172.27.0.3 node1
172.27.0.4 node2
172.27.0.5 node3

./cephadm bootstrap --mon-ip 172.27.0.3
sudo ./cephadm shell   #以下命令均需在./cephadm shell里执行.
ceph cephadm get-pub-key > ~/ceph.pub
ssh-copy-id -f -i ~/ceph.pub root@node2
ssh-copy-id -f -i ~/ceph.pub root@node3
ceph cephadm get-ssh-config > ssh_config
ceph config-key get mgr/cephadm/ssh_identity_key > ~/cephadm_private_key
chmod 0600 ~/cephadm_private_key
ceph orch host add node2
ceph orch host add node3
ceph orch host ls
ceph orch device ls
ceph orch apply mon node1,node2,node3
ceph orch daemon add osd node1:/dev/sdb
ceph orch daemon add osd node2:/dev/sdb
ceph orch daemon add osd node3:/dev/sdb
ceph orch apply mds myfs --placement="3 node1 node2 node3"
ceph orch apply rgw sichuan us-chengdou-1 --placement="3 node1 node2 node3"
ceph orch apply mgr node1,node2,node3
```

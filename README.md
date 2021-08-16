## 1. 集群原有的证书列表
```
pki
├── admin.crt
├── admin.key
├── apiserver.crt
├── apiserver-etcd-client.crt
├── apiserver-etcd-client.key
├── apiserver.key
├── apiserver-kubelet-client.crt
├── apiserver-kubelet-client.key
├── ca.crt
├── ca.key
├── etcd
│   ├── ca.crt
│   ├── ca.key
│   ├── healthcheck-client.crt
│   ├── healthcheck-client.key
│   ├── peer.crt
│   ├── peer.key
│   ├── server.crt
│   └── server.key
├── front-proxy-ca.crt
├── front-proxy-ca.key
├── front-proxy-client.crt
├── front-proxy-client.key
├── kube-controller-manager.crt
├── kube-controller-manager.key
├── kubelet-client-2021-01-31-17-02-54.pem
├── kubelet-client-2021-02-22-11-21-08.pem
├── kubelet-client-2021-03-16-06-22-32.pem
├── kubelet-client-2021-04-09-23-14-13.pem
├── kubelet-client-2021-05-02-07-19-35.pem
├── kubelet-client-2021-05-23-12-25-19.pem
├── kubelet-client-2021-06-14-20-36-08.pem
├── kubelet-client-2021-07-11-11-29-54.pem
├── kubelet-client-2021-08-07-00-51-05.pem
├── kubelet-client-current.pem -> /etc/kubernetes/pki/kubelet-client-2021-08-07-00-51-05.pem
├── kubelet.crt
├── kubelet.key
├── kube-scheduler.crt
├── kube-scheduler.key
├── sa.key
└── sa.pub
```

## 2. 不需要更新的证书
```
pki
├── ca.crt
├── ca.key
├── etcd
│   ├── ca.crt
│   ├── ca.key
├── sa.key
└── sa.pub
```
如果etcd没有证书问题 那么不需要更新etcd的证书
```
├── etcd
│   ├── ca.crt
│   ├── ca.key
│   ├── healthcheck-client.crt
│   ├── healthcheck-client.key
│   ├── peer.crt
│   ├── peer.key
│   ├── server.crt
│   └── server.key
```


## 3. 获取集群信息
```
#kubectl get cluster
NAME         AGE
k8s-120-v3   196d

#kubectl get cluster k8s-120-v3 -o yaml

#记录spce中的config如下
clusterConfig:
apiserverDomains:
- apiserver.cluster0517.0517.antstack.com
clusterDomain: antstack.com
extensionApiserverDomains: null
serviceSubnet: 172.16.0.0/16
site: k8s-120-v3
```


## 4. 生成新的证书 [不能修改ca!!!]  [不能修改sa!!!]
```
登陆ops机器并执行以下命令:
#在ops创建文件夹
mkdir update_cert
cd update_cert

#为了获取原有的ca证书 将一台master上的pki复制到ops机器上
scp -r root@master_ip:/etc/kubernetes/pki  update_cert/old_pki
#创建update_cert/root文件夹
root
├── ca.crt                  【来源 update_cert/old_pki/ca.crt】
├── ca.key                  【来源 update_cert/old_pki/ca.key】
├── etcd                  
│   ├── ca.crt              【来源 update_cert/old_pki/etcd/ca.crt】
│   └── ca.key              【来源 update_cert/old_pki/etcd/ca.key】
├── front-proxy-ca.crt      【来源 update_cert/old_pki/front-proxy-ca.crt】
└── front-proxy-ca.key      【来源 update_cert/old_pki/front-proxy-ca.key】

## 执行二进制命令如下

## 得到结果如下

## 复制原有的sa证书

## 将新生成证书复制到对应3台master上

```
     
## 5. 替换证书
```
##备份
cp -r /etc/kubernetes/  ~/kubernetes_bak
mv /etc/kubernetes/kubeconfig ~/kubernetes_config_bak
mv /etc/kubernetes/pki ~/kubernetes_pki_bak

## 将新生成的证书复制到目的文件夹
cp -r update_cert/kubeconfig/ /etc/kubernetes/
cp -r update_cert/pki/ /etc/kubernetes/
```


## 6. 重启组件
```
# 停止四大件
mv /etc/kubernetes/manifests ~

# 修改kubelet 配置文件
vi /etc/systemd/system/kubelet.service
Environment="KUBELET_KUBECONFIG_CONFIGS=--kubeconfig=/etc/kubernetes/kubeconfig/kubelet.kubeconfig"
修改为
Environment="KUBELET_KUBECONFIG_CONFIGS=--kubeconfig=/etc/kubernetes/kubeconfig/admin.kubeconfig"

# 重启kubelet
systemctl daemon-reload
systemctl restart kubelet 

# 重启四大件
mv ~/manifests/ /etc/kubernetes/
```

## 7. 更新captain数据库
在captain页面上获取集群的id=0000000000000058
```
http://captain.antstack-plus.net:82/clusters/0000000000000058/overview
```
获取cluster_cert_id
```sql
select id from cluster_cert where cluster_id=0000000000000058 and type='admin'
```

执行sql
```sql
在master节点上执行
cat /etc/kubernetes/kubeconfig/admin.kubeconfig
update cluster_cert set kube_config = '{kube_config}' where id = 0000000000000085

在master节点上执行
cat /etc/kubernetes/pki/admin.crt | sed ":tag;N;s/\n/\\\n/;b tag" 可得到一行的cert
update cluster_cert set cert = '{cert}' where id = 0000000000000085

在master节点上执行
cat /etc/kubernetes/pki/admin.key | sed ":tag;N;s/\n/\\\n/;b tag" cakey
update cluster_cert set cakey = '{cakey}' where id = 0000000000000085
```

验证： 检查captain的页面是否会出现证书错误

## 常见错误
#### kubelet
```
iled to ensure lease exists, will retry in 7s, error: leases.coordination.k8s.io "11.166.85.79" is forbidden: User "system:node" cannot get resource "leases" in API group "coordination.k8s.io" in the namespace "kube-node-lease"
```
```
vi /etc/systemd/system/kubelet.service
Environment="KUBELET_KUBECONFIG_CONFIGS=--kubeconfig=/etc/kubernetes/kubeconfig/kubelet.kubeconfig"
修改为
Environment="KUBELET_KUBECONFIG_CONFIGS=--kubeconfig=/etc/kubernetes/kubeconfig/admin.kubeconfig"
```

#### apiserver
```
2021-08-16T21:44:30.214210267+08:00 stderr F Flag --insecure-port has been deprecated, This flag has no effect now and will be removed in v1.24.
2021-08-16T21:44:51.429686065+08:00 stderr F Error: context deadline exceeded
```
查看etcd发现没有etcd的进程了, etcd的参数写错了 多了一个-

#### node
（5）如果有应用节点在刷新证书后不健康，
 报错为
failed to run Kubelet: cannot create certificate signing request: Unauthorized
 
解决方案:
将admin.kubeconfig拷贝到不健康节点上替换掉kubelet.kubeconfig
然后重启sigma-slave

如果还是不健康，需要把 bootstrap.kubeconfig 删掉（备份下）再重启sigma-slave。


## 测试
1 node上的kubelet 不需要修改
部署一个ds并删除
```
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: netshoot-daemon
  namespace: kube-system
spec:
  selector:
    matchLabels:
      app: netshoot-daemon
  template:
    metadata:
      labels:
        app: netshoot-daemon
    spec:
      containers:
      - name: netshoot
        image: acs-reg.alipay.com/acloud/netshoot:latest
        resources:
          limits:
            cpu: "0.5"
            memory: "200Mi"
          requests:
        command: ["sleep"]
        args: ["infinity"]
```
2 node上的kubeproxy 不需要修改
创建一个service并删除 查看相关节点的iptables
```
[root@h07b13164.sqa.eu95 /root]
#kubectl get service
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   172.16.0.1      <none>        443/TCP   197d
ngin2        ClusterIP   172.16.211.67   <none>        80/TCP    5m23s
nginx        ClusterIP   None            <none>        80/TCP    188d
nginx-mp     ClusterIP   None            <none>        80/TCP    97d

[root@h07b13164.sqa.eu95 /root]
#kubectl delete service ngin2
service "ngin2" deleted

[root@h07b13164.sqa.eu95 /root]
#kubectl get service
NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   172.16.0.1   <none>        443/TCP   197d
nginx        ClusterIP   None         <none>        80/TCP    188d
nginx-mp     ClusterIP   None         <none>        80/TCP    97d

[root@h07b13164.sqa.eu95 /root]




[root@h07d07163.sqa.eu95 /root]
#iptables-save |grep ngin2
-A KUBE-SEP-S2YNWRVS2W6ZVZWY -s 11.166.118.205/32 -m comment --comment "default/ngin2:web" -j KUBE-MARK-MASQ
-A KUBE-SEP-S2YNWRVS2W6ZVZWY -p tcp -m comment --comment "default/ngin2:web" -m tcp -j DNAT --to-destination 11.166.118.205:80
-A KUBE-SEP-WWP6GAEW36WX55AM -s 11.166.118.59/32 -m comment --comment "default/ngin2:web" -j KUBE-MARK-MASQ
-A KUBE-SEP-WWP6GAEW36WX55AM -p tcp -m comment --comment "default/ngin2:web" -m tcp -j DNAT --to-destination 11.166.118.59:80
-A KUBE-SERVICES -d 172.16.211.67/32 -p tcp -m comment --comment "default/ngin2:web cluster IP" -m tcp --dport 80 -j KUBE-SVC-ZYP37SLGEAD3MW3B
-A KUBE-SVC-ZYP37SLGEAD3MW3B -m comment --comment "default/ngin2:web" -m statistic --mode random --probability 0.50000000000 -j KUBE-SEP-S2YNWRVS2W6ZVZWY
-A KUBE-SVC-ZYP37SLGEAD3MW3B -m comment --comment "default/ngin2:web" -j KUBE-SEP-WWP6GAEW36WX55AM

[root@h07d07163.sqa.eu95 /root]
#iptables-save |grep ngin2

[root@h07d07163.sqa.eu95 /root]
#
```

3 sa secert
查看所有的secret都只和ca证书有关 不需要更新


```
	ComponentCertRootPKI               = "root-pki"
	ComponentCertAdminPKI              = "admin-pki"
	ComponentCertMasterPKI             = "master-pki"
	ComponentCertExtensionApiserverPKI = "extension-apiserver-pki"

	KubeconfigScheduler         = "scheduler.kubeconfig"
	KubeconfigControllerManager = "controller-manager.kubeconfig"
	KubeconfigAdmin             = "admin.kubeconfig"
	KubeconfigMultiTenancyAdmin = "multi-tenancy-admin.kubeconfig"

kubectl get secret -n kube-system daemon-set-controller-token-vx24r -o yaml
```
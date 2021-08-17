# K8s Cert With Captain

# 背景
使用Captain搭建完集群后所有生成证书文件如下
```
#tree pki
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

## CA 证书
上图可见 我们使用了3套CA证书来管理和签发其他证书
- ca
- front-proxy-ca
- etcd/ca


## etcd证书
- etcd 的根证书
```
/etcd/ca.crt /etcd/ca.key
```
- etcd 对外提供服务的服务器证书及私钥
```
/etcd/server.crt  /etcd/server.key
```
- etcd 节点之间相互进行认证的 peer 证书、私钥以及验证 peer 的 
```
/etcd/peer.crt  /etcd/peer.key
```
- etcd 验证访问其服务的客户端的 CA
```
/etcd/healthcheck-client.crt  /etcd/healthcheck-client.key
```

## apiserver证书
- 用来签发k8s中其他证书的CA证书及私钥
```
ca.crt  ca.key
```
- 访问etcd的客户端证书及私钥，这个证书是由etcd的CA证书签发，因此也需要在apiserver中配置etcd的CA证书
```
apiserver-etcd-client.key   apiserver-etcd-client.crt   
```
- apiServer的对外提供服务的服务端证书及私钥
```
apiserver.crt   apiserver.key 
```
- apiserver 访问 kubelet 所需的客户端证书及私钥
```
apiserver-kubelet-client.crt  apiserver-kubelet-client.key
```
- 配置聚合层（apiserver扩展）的CA和客户端证书及私钥
  说明：要使聚合层在您的环境中正常工作以支持代理服务器和扩展 apiserver 之间的相互 TLS 身份验证， 需要满足一些设置要求。Kubernetes 和 kube-apiserver 具有多个 CA， 因此请确保代理是由聚合层 CA 签名的，而不是由主 CA 签名的。扩展apiserver为了能够和apiserver通讯，所以需要在apiserver中配置，假如你不需要这个功能可以不配置该证书
```
front-proxy-ca.crt  front-proxy-client.crt  front-proxy-ca.key      front-proxy-client.key 
```

- 验证 service account token 的公钥
```
sa.pub sa.key
```

yaml中证书相关的配置
```
    - --client-ca-file=/etc/kubernetes/pki/ca.crt
    - --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
    - --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
    - --kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt
    - --kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key
    - --enable-bootstrap-token-auth=true
    - --feature-gates=RotateKubeletServerCertificate=true
   
    - --etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt
    - --etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt
    - --etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key
   
    - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
    - --proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt
    - --proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key
 
    - --service-account-key-file=/etc/kubernetes/pki/sa.pub
    - --service-account-signing-key-file=/etc/kubernetes/pki/sa.key
    - --service-account-issuer=https://kubernetes.default.svc.cluster.local
    - --log-dir=/logs
    - --logtostderr=false
```

## kube-controller-mananger
```
kube-controller-manager.crt kube-controller-manager.key
```

yaml中证书相关的配置
```
    - --kubeconfig=/etc/kubernetes/kubeconfig/kube-controller-manager.kubeconfig
    - --root-ca-file=/etc/kubernetes/pki/ca.crt
    - --cluster-signing-key-file=/etc/kubernetes/pki/ca.key
    - --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
    - --use-service-account-credentials=true
    - --service-account-private-key-file=/etc/kubernetes/pki/sa.key
    - --feature-gates=RotateKubeletServerCertificate=true
```

## kube-scheduler
```
kube-scheduler.crt kube-scheduler.key
```

yaml中证书相关的配置
```
    --kubeconfig=/etc/kubernetes/kubeconfig/kube-scheduler.kubeconfig
```

## kube-proxy

## kubelet
```
kubelet.crt kubelet.key
```

## Admin
```
admin.crt admin.key
```

# Captain 生成证书

## 1. 生成CA
- ca
- front-proxy-ca
- etcd/ca

a 生成私钥key  
b 根据key生成ca证书crt

subject
```
CommonName:   cfg.CommonName,
Organization: cfg.Organization,
```
## 2. 生成SA
同上

## 3. 生成MasterCert
MasterCert有3类证书
#### a. CertWithIpActions
- CreateAPIServerCertAndKeyFiles  
subject
```
apiserverDomains    apiServer的域名
dnsDomain           dns服务器的域名  kube-dns or core-dns
serviceSubnet       service子网
```
参数的值如下
```
apiserverDomains:
 - apiserver.cluster0517.0517.antstack.com
clusterDomain: antstack.com
extensionApiserverDomains: null
serviceSubnet: 172.16.0.0/16
```
证书中的subject,其中10.15.66.30为master节点ip，172.16.0.1为service子网ip
```
X509v3 Subject Alternative Name:
   DNS:kubernetes, 
   DNS:kubernetes.default, 
   DNS:kubernetes.default.svc, 
   DNS:kubernetes.default.svc.szblocal.com, 
   DNS:apiserver.szbprod.szblocal.com, 
   IP Address:172.16.0.1, 
   IP Address:10.15.66.30
```

- CreateEtcdServerCertAndKeyFiles
- CreateEtcdPeerCertAndKeyFiles 


#### b. CertActions
- CreateEtcdHealthcheckClientCertAndKeyFiles
- CreateAPIServerEtcdClientCertAndKeyFiles
- CreateAPIServerKubeletClientCertAndKeyFiles
- CreateFrontProxyClientCertAndKeyFiles
  
#### c. ClientCerts
生成admin.key和admin.cert































# 常用命令
openssl 
```
openssl x509 -in apiserver.crt -noout -text
openssl x509 -in signed.crt -noout -dates
```

```
##环境检查
cat > check.sh <<-EOF
for x in \$(ls /etc/kubernetes/pki/*.crt);do echo -n \$x; openssl x509 -text -in \$x | grep 'Not After'; done
for x in \$(ls /etc/kubernetes/kubeconfig/*);do echo -n \$x; cat \$x | grep client-certificate-data | awk -F":" '{print \$2}' | base64 -i --decode | openssl x509 -text -noout | grep "Not After"; done
EOF
```

etcd
```
etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key --endpoints=https://11.166.85.73:2379,https://11.166.85.75:2379,https://11.166.85.79:2379 endpoint health -w table
```
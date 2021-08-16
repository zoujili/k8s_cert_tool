module cert_tool

go 1.15

require (
	github.com/sirupsen/logrus v1.8.1
	github.com/urfave/cli v1.22.5
	k8s.io/client-go v0.22.0
	k8s.io/cluster-bootstrap v0.21.1 // indirect
	k8s.io/component-base v0.22.0 // indirect
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kubernetes v1.14.7
)

replace k8s.io/kubernetes v1.14.3 => k8s.io/kubernetes v1.14.7

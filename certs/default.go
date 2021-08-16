package certs

import (
	"cert_tool/config"
	"path"
)

var (
	RootPathName = "kubernetes"

	NodeCertName = "pki"

	NodeKubeconfigName = "kubeconfig"

	NodePodsName = "manifests"

	DefaultNodeCertPath = path.Join("kubernetes", "pki")

	DefaultNodeKubeconfigPath = path.Join("kubernetes", "kubeconfig")

	DefaultNodePodsPath = path.Join("kubernetes", "manifests")

	DefaultCaAndKeyPath = path.Join("root")

	DefaultExternalClients = map[string]config.UserInfo{
		"kubelet": {
			Username: "system:node",
		},
		"kube-proxy": {
			Username: "system:kube-proxy",
		},
		"admin": {
			Username: "admin",
			Groups:   []string{"system:masters"},
		},
	}

	DefaultMasterClients = map[string]config.UserInfo{
		"kube-controller-manager": {
			Username: "system:kube-controller-manager",
		},
		"kube-scheduler": {
			Username: "system:kube-scheduler",
		},
	}
)

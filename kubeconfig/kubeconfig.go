package kubeconfig

import (
	"bytes"
	"cert_tool/certs"
	"cert_tool/config"
	"cert_tool/utils"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"text/template"
)

const (
	kubeconfigTemplate = `
kind: Config
apiVersion: v1
users:
- name: {{ .username }}
  user:
    client-certificate-data: {{ .cert }}
    client-key-data: {{ .key }}
clusters:
- cluster:
    certificate-authority-data: {{ .ca }}
    server: https://{{ .master }}:6443
  name: {{ .cluster }}
contexts:
- context:
    cluster: {{ .cluster }}
    user: {{ .username }}
  name: default
current-context: default
preferences: {}
`
)

func InitKubeConfigs(config *config.Config) error {
	if err := InitMasterKubeConfigs(config); nil != err {
		return err
	}

	if err := InitExternalKubeConfigs(config); nil != err {
		return err
	}

	if err := CopyExternalKubeConfigsToMasters(config); nil != err {
		return err
	}

	return nil
}


func GenerateKubeconfig(cluster string, master string, ca, key, cert []byte, username string) (string, error) {
	ctx := map[string]string{
		"ca":       base64.StdEncoding.EncodeToString(ca),
		"key":      base64.StdEncoding.EncodeToString(key),
		"cert":     base64.StdEncoding.EncodeToString(cert),
		"username": username,
		"master":   master,
		"cluster":  cluster,
	}

	if t, err := template.New("test").Parse(kubeconfigTemplate); nil != err {
		return "", err
	} else {
		writer := bytes.NewBuffer([]byte{})
		if err := t.Execute(writer, ctx); nil != err {
			return "", err
		}

		return writer.String(), nil
	}
}

func WriteKubeConfig(baseDir, name, content string) error {
	if err := os.MkdirAll(baseDir, os.FileMode(0755)); err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(baseDir, name), []byte(content), os.FileMode(0600))
}

func InitMasterKubeConfigs(config *config.Config) error {
	for _, ip := range config.MasterIPs {
		pkiPath := path.Join(ip, certs.DefaultNodeCertPath)
		kubeconfigPath := path.Join( ip, certs.DefaultNodeKubeconfigPath)

		for role, user := range certs.DefaultMasterClients {
			if err := certs.GenerateClientCerts(config, pkiPath, role, user, false); nil != err {
				return err
			}
			if err := NewKubeConfig(pkiPath, role, kubeconfigPath, config); nil != err {
				return err
			}
		}
	}

	return nil
}

func InitExternalKubeConfigs(config *config.Config) error {
	for role, user := range certs.DefaultExternalClients {
		if err := NewCertPairAndKubeConfig(config, role, &user); nil != err {
			return err
		}
	}

	return nil
}

func NewKubeConfig(pkiBaseDir string, role string, kubeconfigDir string, config *config.Config) error {
	ca, err := certs.LoadDefaultCertificateAuthority(config)
	if err != nil {
		return err
	}

	if nil != err {
		return err
	}

	cert, key, err := certs.LoadCertificateAndKey(pkiBaseDir, role)

	if nil != err {
		return err
	}

	kbConf, err := GenerateKubeconfig("default", config.ApiServerDomain, utils.EncodeCertPEM(ca), utils.EncodePrivateKeyPEM(key), utils.EncodeCertPEM(cert), role)

	if nil != err {
		return err
	}

	if err := WriteKubeConfig(kubeconfigDir, fmt.Sprintf("%s.kubeconfig", role), kbConf); nil != err {
		return err
	}

	return nil
}

func NewCertPairAndKubeConfig(config *config.Config, targetBasePath string, user *config.UserInfo) error {
	target := path.Join(targetBasePath)
	if err := certs.GenerateClientCerts(config, target, targetBasePath, *user, true); nil != err {
		return err
	}

	if err := NewKubeConfig(target, targetBasePath, target, config); nil != err {
		return err
	}

	return nil
}

func CopyExternalKubeConfigsToMasters(config *config.Config) error {
	for _, ip := range config.MasterIPs {
		kubeconfigPath := path.Join(ip, certs.DefaultNodeKubeconfigPath)

		for role := range certs.DefaultExternalClients {
			pkiPath := path.Join(role)
			if err := NewKubeConfig(pkiPath, role, kubeconfigPath, config); nil != err {
				return err
			}
		}
	}

	return nil
}


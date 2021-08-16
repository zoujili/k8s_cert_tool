package main_test

import (
	"cert_tool/certs"
	"cert_tool/config"
	"cert_tool/kubeconfig"
	"fmt"
	"testing"
)

func TestCert(t *testing.T) {
	conf := &config.Config{
		ApiServerDomain: "apiserver.cluster0517.0517.antstack.com",
		DnsDomain:       "antstack.com",
		ServiceSubnet:   "172.16.0.0/16",
		MasterIPs:       []string{"11.166.85.73","11.166.85.75","11.166.85.79"},
	}

	err:= certs.InitCerts(conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = kubeconfig.InitKubeConfigs(conf)
	if err != nil {
		fmt.Println(err)
		return
	}
}

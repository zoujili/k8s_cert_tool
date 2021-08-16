package main

import (
	"cert_tool/certs"
	"cert_tool/config"
	"cert_tool/kubeconfig"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var (
	FlagApiServerDomain = ""
	FlagDnsDomain       = ""
	FlagServiceSubnet   = ""
	FlagMasterIPs       = cli.StringSlice{}
)

func init() {
	App.Name = "cert tool"
	App.Usage = "renew all cert and kubeconfig"
	App.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "apiserverdomain",
			Usage:       "apiserver的域名 形如:apiserver.szbprod.szblocal.com",
			Value:       "",
			Destination: &FlagApiServerDomain,
		},
		cli.StringFlag{
			Name:        "dnsdomain",
			Usage:       "外部或内部dns的域名 形如：antstack.com",
			Value:       "",
			Destination: &FlagDnsDomain,
		},
		cli.StringFlag{
			Name:        "servicesubnet",
			Usage:       "service子网的cidr 形如：172.16.0.0/16",
			Value:       "172.16.0.0/16",
			Destination: &FlagServiceSubnet,
		},
		cli.StringSliceFlag{
			Name:  "masterips",
			Usage: "Images Source File",
			Value: &FlagMasterIPs,
		},
	}

	App.Commands = []cli.Command{
		GenCertCommand,
	}
}

var GenCertCommand = cli.Command{
	Name:  "gen",
	Usage: "Renew all cert and kubeConfig",
	Flags: []cli.Flag{},
	Action: func(cliContext *cli.Context) error {
		var err error
		if FlagApiServerDomain == "" {
			err = errors.New("flag apiServerDomain empty")
			logrus.Error(err)
			return err
		}
		if FlagDnsDomain == "" {
			err = errors.New("flag dnsDomain empty")
			logrus.Error(err)
			return err
		}
		if FlagServiceSubnet == "" {
			err = errors.New("flag serviceSubnet empty")
			logrus.Error(err)
			return err
		}
		if len(FlagMasterIPs) != 3 {
			err = errors.New("flag master ip not equal 3")
			logrus.Error(err)
			return err
		}

		conf := &config.Config{
			ApiServerDomain: FlagApiServerDomain,
			DnsDomain:       FlagDnsDomain,
			ServiceSubnet:   FlagServiceSubnet,
			MasterIPs:       FlagMasterIPs,
		}

		err = certs.InitCerts(conf)
		if err != nil {
			return err
		}

		err = kubeconfig.InitKubeConfigs(conf)
		if err != nil {
			return err
		}
		return nil
	},
}

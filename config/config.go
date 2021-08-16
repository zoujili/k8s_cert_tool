package config


type Config struct {
	ApiServerDomain string    //apiserver的域名 形如:apiserver.szbprod.szblocal.com
	DnsDomain string     //外部或内部dns的域名 形如：antstack.com
	ServiceSubnet string  //service子网的cidr 形如：172.16.0.0/16
	MasterIPs []string    //master的ip


	ExtensionApiServerDomains []string
	ExpireDays  int
}

type UserInfo struct {
	Username string
	Groups   []string
}
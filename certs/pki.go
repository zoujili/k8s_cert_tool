package certs

import (
	"cert_tool/config"
	"cert_tool/utils"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"k8s.io/kubernetes/pkg/registry/core/service/ipallocator"
	"math"
	"math/big"
	"net"
	"time"

	"k8s.io/client-go/util/cert"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"


)

const (
	defaultClusterDomain = "cluster.local"
)

var (
	// ten years
	NeverExpireDuration = time.Hour * 24 * 365 * 10

	componentCertDefaultExpireDuration = time.Hour * 24 * 365 * 10
)

type CACertKeyPair struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

type ClusterCAGroup struct {
	Apiserver  *CACertKeyPair
	Etcd       *CACertKeyPair
	FrontProxy *CACertKeyPair

	ServiceAccountPublicKey  *rsa.PublicKey
	ServiceAccountPrivateKey *rsa.PrivateKey
}

type CertConfig struct {
	CommonName     string
	Organization   []string
	AltNames       cert.AltNames
	Usages         []x509.ExtKeyUsage
	ExpireDuration time.Duration
}

func NewCACertKeyPair(isMultiTenancy bool) (*CACertKeyPair, error) {
	organization := []string{"Alibaba.Inc", "Alipay.Inc"}
	if isMultiTenancy {
		organization = []string{}
	}
	if cert, key, err := pkiutil.NewCertificateAuthority(&cert.Config{
		CommonName:   "Kubernetes",
		Organization: organization,
	}); nil != err {
		return nil, err
	} else {
		return &CACertKeyPair{
			Cert: cert,
			Key:  key,
		}, nil
	}
}

func NewClusterCACertGroup(isMultiTenancy bool) (*ClusterCAGroup, error) {
	g := &ClusterCAGroup{}

	if apiserver, err := NewCACertKeyPair(isMultiTenancy); nil != err {
		return nil, err
	} else {
		g.Apiserver = apiserver
	}

	if etcd, err := NewCACertKeyPair(isMultiTenancy); nil != err {
		return nil, err
	} else {
		g.Etcd = etcd
	}

	if frontProxy, err := NewCACertKeyPair(isMultiTenancy); nil != err {
		return nil, err
	} else {
		g.FrontProxy = frontProxy
	}

	if privateKey, err := NewPrivateKey(); nil != err {
		return nil, err
	} else {
		g.ServiceAccountPrivateKey = privateKey
		g.ServiceAccountPublicKey = &privateKey.PublicKey
	}

	return g, nil
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg CertConfig, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	expireDuration := cfg.ExpireDuration
	if expireDuration < 0 {
		expireDuration = NeverExpireDuration
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(expireDuration).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

func NewCertAndKey(cfg CertConfig, caCert *x509.Certificate, caKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key [%v]", err)
	}

	cert, err := NewSignedCert(cfg, key, caCert, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate [%v]", err)
	}

	return cert, key, nil
}

func GetComponentCertExpireDuration(config *config.Config) time.Duration {
	//if config.ExpireDays > 0 {
	//	return time.Hour * 24 * time.Duration(config.ExpireDays)
	//} else {
	//	return componentCertDefaultExpireDuration
	//}
	return NeverExpireDuration
}

func GetUserCertExpireDuration(config *config.Config) time.Duration {
	return NeverExpireDuration
}

// NewAPIServerCertAndKey generate certificate for apiserver, signed by the given CA.
func NewAPIServerCertAndKey(ca *CACertKeyPair, config *config.Config, advertiseAddress string) (*x509.Certificate, *rsa.PrivateKey, error) {
	apiserverDomain := config.ApiServerDomain
	clusterDomain := config.DnsDomain
	if len(clusterDomain) <= 0 {
		clusterDomain = defaultClusterDomain
	}

	// create AltNames with defaults DNSNames/IPs
	altNames := &cert.AltNames{
		DNSNames: []string{
			"kubernetes",
			"kubernetes.default",
			"kubernetes.default.svc",
			fmt.Sprintf("kubernetes.default.svc.%s", clusterDomain),
		},
		IPs: []net.IP{},
	}

	apiserverIP := net.ParseIP(apiserverDomain)
	if apiserverIP == nil {
		altNames.DNSNames = append(altNames.DNSNames, apiserverDomain)
	} else {
		altNames.IPs = append(altNames.IPs, apiserverIP)
	}


	if len(config.ServiceSubnet) > 0 {
		// internal IP address for the API server
		_, svcSubnet, err := net.ParseCIDR(config.ServiceSubnet)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing CIDR %q: %v", config.ServiceSubnet, err)
		}

		internalAPIServerVirtualIP, err := ipallocator.GetIndexedIP(svcSubnet, 1)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to get first IP address from the given CIDR (%s): %v", svcSubnet.String(), err)
		}

		altNames.IPs = append(altNames.IPs, internalAPIServerVirtualIP)
	}

	// advertise address
	if len(advertiseAddress) > 0 {
		advertiseAddress := net.ParseIP(advertiseAddress)
		if advertiseAddress == nil {
			return nil, nil, fmt.Errorf("error parsing AdvertiseAddress %v: is not a valid textual representation of an IP address", advertiseAddress)
		}

		altNames.IPs = append(altNames.IPs, advertiseAddress)
	}

	certCfg := CertConfig{
		CommonName:     kubeadmconstants.APIServerCertCommonName,
		AltNames:       *altNames,
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}

	apiCert, apiKey, err := NewCertAndKey(certCfg, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating apiserver key and certificate: %v", err)
	}

	return apiCert, apiKey, nil
}

// NewAPIServerKubeletClientCertAndKey generate certificate for the apiservers to connect to the kubelets securely, signed by the given CA.
func NewAPIServerKubeletClientCertAndKey(ca *CACertKeyPair, config *config.Config) (*x509.Certificate, *rsa.PrivateKey, error) {

	conf := CertConfig{
		CommonName:     kubeadmconstants.APIServerKubeletClientCertCommonName,
		Organization:   []string{kubeadmconstants.SystemPrivilegedGroup},
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	apiClientCert, apiClientKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating API server kubelet client key and certificate: %v", err)
	}

	return apiClientCert, apiClientKey, nil
}

// NewEtcdServerCertAndKey generate certificate for etcd, signed by the given CA.
func NewEtcdServerCertAndKey(ca *CACertKeyPair, config *config.Config, etcdDomain string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// create AltNames with defaults DNSNames/IPs
	altNames := &cert.AltNames{
		DNSNames: []string{"localhost", etcdDomain},
		IPs:      []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	// advertise address
	advertiseAddress := net.ParseIP(etcdDomain)
	if advertiseAddress != nil {
		altNames.IPs = append(altNames.IPs, advertiseAddress)
	}

	conf := CertConfig{
		CommonName:     "kube-etcd",
		AltNames:       *altNames,
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	etcdServerCert, etcdServerKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating etcd key and certificate: %v", err)
	}

	return etcdServerCert, etcdServerKey, nil
}

// NewEtcdPeerCertAndKey generate certificate for etcd peering, signed by the given CA.
func NewEtcdPeerCertAndKey(ca *CACertKeyPair, config *config.Config, etcdDomain string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// create AltNames with defaults DNSNames/IPs
	altNames := &cert.AltNames{
		DNSNames: []string{"localhost", etcdDomain},
		IPs:      []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	// advertise address
	advertiseAddress := net.ParseIP(etcdDomain)
	if advertiseAddress != nil {
		altNames.IPs = append(altNames.IPs, advertiseAddress)
	}

	conf := CertConfig{
		CommonName:     "kube-etcd-peer",
		AltNames:       *altNames,
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	etcdPeerCert, etcdPeerKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating etcd peer key and certificate: %v", err)
	}

	return etcdPeerCert, etcdPeerKey, nil
}

// NewEtcdHealthcheckClientCertAndKey generate certificate for liveness probes to healthcheck etcd, signed by the given CA.
func NewEtcdHealthcheckClientCertAndKey(ca *CACertKeyPair, config *config.Config) (*x509.Certificate, *rsa.PrivateKey, error) {

	conf := CertConfig{
		CommonName:     kubeadmconstants.EtcdHealthcheckClientCertCommonName,
		Organization:   []string{kubeadmconstants.SystemPrivilegedGroup},
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	etcdHealcheckClientCert, etcdHealcheckClientKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating etcd healthcheck client key and certificate: %v", err)
	}

	return etcdHealcheckClientCert, etcdHealcheckClientKey, nil
}

// NewAPIServerEtcdClientCertAndKey generate certificate for the apiservers to connect to etcd securely, signed by the given CA.
func NewAPIServerEtcdClientCertAndKey(ca *CACertKeyPair, config *config.Config) (*x509.Certificate, *rsa.PrivateKey, error) {

	conf := CertConfig{
		CommonName:     kubeadmconstants.APIServerEtcdClientCertCommonName,
		Organization:   []string{kubeadmconstants.SystemPrivilegedGroup},
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	apiClientCert, apiClientKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating API server etcd client key and certificate: %v", err)
	}

	return apiClientCert, apiClientKey, nil
}

// NewServiceAccountSigningKey generate public/private key pairs for signing service account tokens.
func NewServiceAccountSigningKey() (*rsa.PrivateKey, error) {

	// The key does NOT exist, let's generate it now
	saSigningKey, err := NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failure while creating service account token signing key: %v", err)
	}

	return saSigningKey, nil
}

// NewFrontProxyClientCertAndKey generate certificate for proxy server client, signed by the given front proxy CA.
func NewFrontProxyClientCertAndKey(ca *CACertKeyPair, config *config.Config) (*x509.Certificate, *rsa.PrivateKey, error) {

	conf := CertConfig{
		CommonName:     kubeadmconstants.FrontProxyClientCertCommonName,
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}
	frontProxyClientCert, frontProxyClientKey, err := NewCertAndKey(conf, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating front-proxy client key and certificate: %v", err)
	}

	return frontProxyClientCert, frontProxyClientKey, nil
}

// NewExtendApiserverServerCertAndKey generate certificate for extend apiserver server HTTPS, signed by the given front proxy CA.
func NewExtensionApiserverServerCertAndKey(ca *CACertKeyPair,config *config.Config) (*x509.Certificate, *rsa.PrivateKey, error) {
	altNames := &cert.AltNames{
		DNSNames: []string{
			"extensions-apiserver.kube-system.svc",
		},
		IPs: []net.IP{},
	}

	altNames.DNSNames = append(altNames.DNSNames, config.ExtensionApiServerDomains...)

	certCfg := CertConfig{
		CommonName:     "kube-extension-apiserver",
		AltNames:       *altNames,
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}

	serveCert, serveKey, err := NewCertAndKey(certCfg, ca.Cert, ca.Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while creating extension apiserver HTTPS serve key and certificate: %v", err)
	}

	return serveCert, serveKey, nil
}
func ParseCertPEM(data []byte) (*x509.Certificate, error) {
	certs, err := cert.ParseCertsPEM(data)
	if nil != err {
		return nil, err
	}

	return certs[0], nil
}

func ParsePrivateKey(data []byte) (*rsa.PrivateKey, error) {
	// Parse the private key from a file
	privKey, err := utils.ParsePrivateKeyPEM(data)
	if err != nil {
		return nil, err
	}
	// Allow RSA format only
	var key *rsa.PrivateKey
	switch k := privKey.(type) {
	case *rsa.PrivateKey:
		key = k
	default:
		return nil, fmt.Errorf("the private key data isn't in RSA format")
	}

	return key, nil
}

func ParsePublicKey(data []byte) (*rsa.PublicKey, error) {
	// Parse the private key from a file
	pubKeys, err := utils.ParsePublicKeysPEM(data)
	if err != nil {
		return nil, err
	}

	// Allow RSA format only
	var key *rsa.PublicKey
	switch k := pubKeys[0].(type) {
	case *rsa.PublicKey:
		key = k
	default:
		return nil, fmt.Errorf("the public key data isn't in RSA format")
	}

	return key, nil
}

package certs

import (
	"cert_tool/config"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	certutil "k8s.io/client-go/util/cert"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"

	"k8s.io/kubernetes/cmd/kubeadm/app/util/pkiutil"
	"path"
	"time"

)

func InitCerts(config *config.Config) error {
	if err := LoadCaAndKeys(config); nil != err {
		return err
	}

	//if err := InitServiceAccountKeys(config); nil != err {
	//	return err
	//}

	if err := InitMasterCerts(config); nil != err {
		return err
	}

	return nil
}

func LoadCaAndKeys(config *config.Config) error {
//	certUtilConfig := &certutil.Config{
//		CommonName:   "Kubernetes",
//		Organization: []string{"Alibaba.Inc", "Alipay.Inc"},
//	}
//
//
//
//	apiCaCert, apiCaKey, err := NewCertificateAuthority(certUtilConfig)
//	if nil != err {
//		return err
//	}
//
//	etcdCaCert, etcdCaKey, err := NewCertificateAuthority(certUtilConfig)
//	if nil != err {
//		return err
//	}
//
//	frontendCaCert, frontendCaKey, err := NewCertificateAuthority(certUtilConfig)
//	if nil != err {
//		return err
//	}

	//// API Ca & CaKey
	//if err := pkiutil.WriteCertAndKey(rootPath, kubeadmconstants.CACertAndKeyBaseName, apiCaCert, apiCaKey); nil != err {
	//	return err
	//}
	//
	//// etcd Ca & CaKey
	//if err := pkiutil.WriteCertAndKey(rootPath, kubeadmconstants.EtcdCACertAndKeyBaseName, etcdCaCert, etcdCaKey); nil != err {
	//	return err
	//}
	//
	//// front-end Ca & CaKey
	//if err := pkiutil.WriteCertAndKey(rootPath, kubeadmconstants.FrontProxyCACertAndKeyBaseName, frontendCaCert, frontendCaKey); nil != err {
	//	return err
	//}

	rootPath := path.Join(DefaultCaAndKeyPath)

	apiCaCert, apiCaKey, err := LoadCertificateAuthority(rootPath, kubeadmconstants.CACertAndKeyBaseName)
	if nil != err {
		return err
	}

	etcdCaCert, etcdCaKey, err := LoadCertificateAuthority(rootPath, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if nil != err {
		return err
	}

	frontendCaCert, frontendCaKey, err := LoadCertificateAuthority(rootPath, kubeadmconstants.FrontProxyCACertAndKeyBaseName)
	if nil != err {
		return err
	}

	for _, ip := range config.MasterIPs {
		basePath := path.Join(ip, DefaultNodeCertPath)
		// API Ca & CaKey
		if err := pkiutil.WriteCertAndKey(basePath, kubeadmconstants.CACertAndKeyBaseName, apiCaCert, apiCaKey); nil != err {
			return err
		}

		// etcd Ca & CaKey
		if err := pkiutil.WriteCertAndKey(basePath, kubeadmconstants.EtcdCACertAndKeyBaseName, etcdCaCert, etcdCaKey); nil != err {
			return err
		}

		// front-end Ca & CaKey
		if err := pkiutil.WriteCertAndKey(basePath, kubeadmconstants.FrontProxyCACertAndKeyBaseName, frontendCaCert, frontendCaKey); nil != err {
			return err
		}
	}

	return nil
}


func InitServiceAccountKeys(config *config.Config) error {
	rootPath := path.Join(DefaultCaAndKeyPath)

	key, err := NewServiceAccountSigningKey()

	if nil != err {
		return err
	}

	if err := pkiutil.WriteKey(rootPath, kubeadmconstants.ServiceAccountKeyBaseName, key); err != nil {
		return fmt.Errorf("failure while saving %s key in root path: %v", kubeadmconstants.ServiceAccountPrivateKeyName, err)
	}

	key, err = pkiutil.TryLoadKeyFromDisk(rootPath, kubeadmconstants.ServiceAccountKeyBaseName)
	if err != nil {
		return err
	}

	for _, ip := range config.MasterIPs {
		// Write .key and .pub files to disk
		if err := pkiutil.WriteKey(path.Join(ip, DefaultNodeCertPath), kubeadmconstants.ServiceAccountKeyBaseName, key); err != nil {
			return fmt.Errorf("failure while saving %s key: %v", kubeadmconstants.ServiceAccountPrivateKeyName, err)
		}

		if err := pkiutil.WritePublicKey(path.Join(ip, DefaultNodeCertPath), kubeadmconstants.ServiceAccountKeyBaseName, &key.PublicKey); err != nil {
			return fmt.Errorf("failure while saving %s public key: %v", kubeadmconstants.ServiceAccountPublicKeyName, err)
		}
	}

	return nil
}

func InitMasterCerts(conf *config.Config) error {
	certWithIpActions := []func(certificatesDir string, config *config.Config, advertiseAddress string) error{
		CreateAPIServerCertAndKeyFiles,
		CreateEtcdServerCertAndKeyFiles,
		CreateEtcdPeerCertAndKeyFiles,
	}

	certActions := []func(certificatesDir string,config *config.Config) error{
		CreateEtcdHealthcheckClientCertAndKeyFiles,
		CreateAPIServerEtcdClientCertAndKeyFiles,
		CreateFrontProxyClientCertAndKeyFiles,
		CreateAPIServerKubeletClientCertAndKeyFiles,
	}

	for _, ip := range conf.MasterIPs {
		certificatesDir := path.Join(ip, DefaultNodeCertPath)

		for _, action := range certWithIpActions {
			err := action(certificatesDir, conf,ip)
			if err != nil {
				return err
			}
		}

		for _, action := range certActions {
			err := action(certificatesDir, conf)
			if err != nil {
				return err
			}
		}

		if err := GenerateClientCerts(conf, certificatesDir, "admin", DefaultExternalClients["admin"], false); nil != err {
			return err
		}

		fmt.Printf("[certificates] Valid certificates and keys now exist in %q\n", certificatesDir)
	}

	return nil
}



func CreateAPIServerCertAndKeyFiles(certificatesDir string, config *config.Config, advertiseAddress string) error {

	caCert, caKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.CACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: caCert,
		Key:  caKey,
	}

	apiCert, apiKey, err := NewAPIServerCertAndKey(caCertKeyPair, config, advertiseAddress)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.APIServerCertAndKeyBaseName,
		caCert,
		apiCert,
		apiKey,
	)
}

// CreateEtcdServerCertAndKeyFiles create a new certificate and key file for etcd.
// If the etcd serving certificate and key file already exist in the target folder, they are used only if evaluated equal; otherwise an error is returned.
// It assumes the etcd CA certificate and key file exist in the CertificatesDir
func CreateEtcdServerCertAndKeyFiles(certificatesDir string, config *config.Config, etcdIp string) error {
	etcdCACert, etcdCAKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: etcdCACert,
		Key:  etcdCAKey,
	}

	etcdServerCert, etcdServerKey, err := NewEtcdServerCertAndKey(caCertKeyPair, config, etcdIp)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.EtcdServerCertAndKeyBaseName,
		etcdCACert,
		etcdServerCert,
		etcdServerKey,
	)
}

// CreateEtcdPeerCertAndKeyFiles create a new certificate and key file for etcd peering.
// If the etcd peer certificate and key file already exist in the target folder, they are used only if evaluated equal; otherwise an error is returned.
// It assumes the etcd CA certificate and key file exist in the CertificatesDir
func CreateEtcdPeerCertAndKeyFiles(certificatesDir string, config *config.Config, etcdIp string) error {

	etcdCACert, etcdCAKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: etcdCACert,
		Key:  etcdCAKey,
	}

	etcdPeerCert, etcdPeerKey, err := NewEtcdPeerCertAndKey(caCertKeyPair, config, etcdIp)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.EtcdPeerCertAndKeyBaseName,
		etcdCACert,
		etcdPeerCert,
		etcdPeerKey,
	)
}

// CreateEtcdHealthcheckClientCertAndKeyFiles create a new client certificate for liveness probes to healthcheck etcd
// If the etcd-healthcheck-client certificate and key file already exist in the target folder, they are used only if evaluated equal; otherwise an error is returned.
// It assumes the etcd CA certificate and key file exist in the CertificatesDir
func CreateEtcdHealthcheckClientCertAndKeyFiles(certificatesDir string, config *config.Config) error {

	etcdCACert, etcdCAKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: etcdCACert,
		Key:  etcdCAKey,
	}

	etcdHealthcheckClientCert, etcdHealthcheckClientKey, err := NewEtcdHealthcheckClientCertAndKey(caCertKeyPair, config)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.EtcdHealthcheckClientCertAndKeyBaseName,
		etcdCACert,
		etcdHealthcheckClientCert,
		etcdHealthcheckClientKey,
	)
}

// CreateAPIServerEtcdClientCertAndKeyFiles create a new client certificate for the apiserver calling etcd
// If the apiserver-etcd-client certificate and key file already exist in the target folder, they are used only if evaluated equal; otherwise an error is returned.
// It assumes the etcd CA certificate and key file exist in the CertificatesDir
func CreateAPIServerEtcdClientCertAndKeyFiles(certificatesDir string, config *config.Config) error {

	etcdCACert, etcdCAKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: etcdCACert,
		Key:  etcdCAKey,
	}

	apiEtcdClientCert, apiEtcdClientKey, err := NewAPIServerEtcdClientCertAndKey(caCertKeyPair, config)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.APIServerEtcdClientCertAndKeyBaseName,
		etcdCACert,
		apiEtcdClientCert,
		apiEtcdClientKey,
	)
}

// CreateFrontProxyClientCertAndKeyFiles create a new certificate for proxy server client.
// If the front-proxy-client certificate and key files already exists in the target folder, they are used only if evaluated equals; otherwise an error is returned.
// It assumes the front proxy CA certificate and key files exist in the CertificatesDir.
func CreateFrontProxyClientCertAndKeyFiles(certificatesDir string, config *config.Config) error {

	frontProxyCACert, frontProxyCAKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: frontProxyCACert,
		Key:  frontProxyCAKey,
	}

	frontProxyClientCert, frontProxyClientKey, err := NewFrontProxyClientCertAndKey(caCertKeyPair, config)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.FrontProxyClientCertAndKeyBaseName,
		frontProxyCACert,
		frontProxyClientCert,
		frontProxyClientKey,
	)
}

// CreateAPIServerKubeletClientCertAndKeyFiles create a new certificate for kubelets calling apiserver.
// If the apiserver-kubelet-client certificate and key files already exists in the target folder, they are used only if evaluated equals; otherwise an error is returned.
// It assumes the cluster CA certificate and key files exist in the CertificatesDir.
func CreateAPIServerKubeletClientCertAndKeyFiles(certificatesDir string, config *config.Config) error {

	caCert, caKey, err := LoadCertificateAuthority(certificatesDir, kubeadmconstants.CACertAndKeyBaseName)
	if err != nil {
		return err
	}

	caCertKeyPair := &CACertKeyPair{
		Cert: caCert,
		Key:  caKey,
	}

	apiKubeletClientCert, apiKubeletClientKey, err := NewAPIServerKubeletClientCertAndKey(caCertKeyPair, config)
	if err != nil {
		return err
	}

	return writeCertificateFilesIfNotExist(
		certificatesDir,
		kubeadmconstants.APIServerKubeletClientCertAndKeyBaseName,
		caCert,
		apiKubeletClientCert,
		apiKubeletClientKey,
	)
}


func GenerateClientCerts(config *config.Config, pkiDir string, baseName string, info config.UserInfo, writeCa bool) error {
	// Try to load certificate authority .crt and .key from the PKI directory
	caCert, caKey, err := LoadCertificateAuthority(path.Join(DefaultCaAndKeyPath), kubeadmconstants.CACertAndKeyBaseName)

	if nil != err {
		return err
	}

	certConfig := CertConfig{
		Usages:         []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		ExpireDuration: GetComponentCertExpireDuration(config),
	}

	if "" != info.Username {
		certConfig.CommonName = info.Username
	}
	if 0 < len(info.Groups) {
		certConfig.Organization = info.Groups
	}

	cert, key, err := NewCertAndKey(certConfig, caCert, caKey)

	if nil != err {
		return err
	}

	if err := pkiutil.WriteCertAndKey(pkiDir, baseName, cert, key); nil != err {
		return err
	}

	if writeCa {
		if err := pkiutil.WriteCert(pkiDir, kubeadmconstants.CACertAndKeyBaseName, caCert); nil != err {
			return err
		}
	}

	return nil
}

const (
	rsaKeySize   = 2048
	duration365d = time.Hour * 24 * 365
)

func NewPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config *certutil.Config) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key [%v]", err)
	}

	cert, err := certutil.NewSelfSignedCACert(*config, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed certificate [%v]", err)
	}

	// overwrite NotBefore to avoid timezone issues
	// change to the time that 1 day early
	cert.NotBefore = time.Now().AddDate(0, 0, -1).UTC()

	return cert, key, nil
}

func LoadCertificateAuthority(pkiDir string, baseName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Try to load certificate authority .crt and .key from the PKI directory
	caCert, caKey, err := LoadCertificateAndKey(pkiDir, baseName)

	if nil != err {
		return nil, nil, err
	}

	// Make sure the loaded CA cert actually is a CA
	if !caCert.IsCA {
		return nil, nil, fmt.Errorf("%s certificate is not a certificate authority", baseName)
	}

	return caCert, caKey, nil
}

func LoadCertificateAndKey(pkiDir string, baseName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Checks if certificate authority exists in the PKI directory
	if !pkiutil.CertOrKeyExist(pkiDir, baseName) {
		return nil, nil, fmt.Errorf("couldn't load %s certificate authority from %s", baseName, pkiDir)
	}

	// Try to load certificate authority .crt and .key from the PKI directory
	caCert, caKey, err := pkiutil.TryLoadCertAndKeyFromDisk(pkiDir, baseName)
	if err != nil {
		return nil, nil, fmt.Errorf("failure loading %s certificate authority: %v", baseName, err)
	}

	return caCert, caKey, nil
}


func writeCertificateFilesIfNotExist(pkiDir string, baseName string, signingCert *x509.Certificate, cert *x509.Certificate, key *rsa.PrivateKey) error {

	// Checks if the signed certificate exists in the PKI directory
	if pkiutil.CertOrKeyExist(pkiDir, baseName) {
		// Try to load signed certificate .crt and .key from the PKI directory
		signedCert, _, err := pkiutil.TryLoadCertAndKeyFromDisk(pkiDir, baseName)
		if err != nil {
			return fmt.Errorf("failure loading %s certificate: %v", baseName, err)
		}

		// Check if the existing cert is signed by the given CA
		if err := signedCert.CheckSignatureFrom(signingCert); err != nil {
			return fmt.Errorf("certificate %s is not signed by corresponding CA", baseName)
		}

		// kubeadm doesn't validate the existing certificate more than this;
		// Basically, if we find a certificate file with the same path; and it is signed by
		// the expected certificate authority, kubeadm thinks those files are equal and
		// doesn't bother writing a new file
		fmt.Printf("[certificates] Using the existing %s certificate and key.\n", baseName)
	} else {

		// Write .crt and .key files to disk
		if err := pkiutil.WriteCertAndKey(pkiDir, baseName, cert, key); err != nil {
			return fmt.Errorf("failure while saving %s certificate and key: %v", baseName, err)
		}

		fmt.Printf("[certificates] Generated %s certificate and key.\n", baseName)
		if pkiutil.HasServerAuth(cert) {
			fmt.Printf("[certificates] %s serving cert is signed for DNS names %v and IPs %v\n", baseName, cert.DNSNames, cert.IPAddresses)
		}
	}

	return nil
}

func LoadDefaultCertificateAuthority(config *config.Config) (*x509.Certificate, error) {
	// Try to load certificate authority .crt and .key from the PKI directory
	caCert, err := pkiutil.TryLoadCertFromDisk(path.Join(DefaultCaAndKeyPath), kubeadmconstants.CACertAndKeyBaseName)
	if err != nil {
		return nil, fmt.Errorf("failure loading %s certificate authority: %v", "root", err)
	}

	// Make sure the loaded CA cert actually is a CA
	if !caCert.IsCA {
		return nil, fmt.Errorf("%s certificate is not a certificate authority", "root")
	}

	return caCert, nil
}


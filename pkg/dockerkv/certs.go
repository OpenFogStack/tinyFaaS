package dockerkv

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const bitsize = 2048

// https://stackoverflow.com/questions/64104586/use-golang-to-get-rsa-key-the-same-way-openssl-genrsa/64105068#64105068
func createCert(name string, hosts []net.IP, dir string, caCertPath string, caKeyPath string) (keyPath string, certPath string, err error) {

	// check that the dir exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", "", err
	}

	caCertBytes, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return "", "", err
	}

	caKeyBytes, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return "", "", err
	}

	// Parse the CA certificate and private key
	caCertBlock, _ := pem.Decode(caCertBytes)
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return "", "", err
	}

	caKeyBlock, _ := pem.Decode(caKeyBytes)
	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return "", "", err
	}

	// Generate the private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	keyPath = filepath.Join(dir, fmt.Sprintf("%s.key", name))

	// Write the private key to a file
	keyFile, err := os.Create(keyPath)
	if err != nil {
		return "", "", err
	}
	defer keyFile.Close()

	// write the private key to the file
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return "", "", err
	}

	pem.Encode(keyFile, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	})

	// Create the CSR template
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName:         name,
			Country:            []string{"DE"},
			Locality:           []string{"Berlin"},
			Organization:       []string{"OpenFogStack"},
			OrganizationalUnit: []string{"tinyFaaS"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(5, 0, 0), // 5 years validity
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	template.IPAddresses = append(template.IPAddresses, hosts...)

	// Generate the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &privateKey.PublicKey, caKey)
	if err != nil {
		return "", "", err
	}

	// Write the certificate to a file
	certPath = filepath.Join(dir, fmt.Sprintf("%s.crt", name))

	certFile, err := os.Create(certPath)
	if err != nil {
		return "", "", err
	}

	defer certFile.Close()

	pem.Encode(certFile, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	log.Printf("Created certificate %s and key %s", certPath, keyPath)

	return keyPath, certPath, nil
}

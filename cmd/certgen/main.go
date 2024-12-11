// certgen generates a local development certificate without an OpenSSL dependency.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	cert := flag.String("cert", ".local/server.crt", "new certificate file (must not exist)")
	key := flag.String("key", ".local/server.key", "new private key file (must not exist)")
	hosts := flag.String("hosts", "localhost,127.0.0.1,::1", "comma-separated DNS names and IP addresses")
	flag.Parse()
	if err := generate(*cert, *key, *hosts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Created %s and %s. This self-signed certificate is for local development.\n", *cert, *key)
}

func generate(certPath, keyPath, hosts string) error {
	if filepath.Clean(certPath) == filepath.Clean(keyPath) {
		return fmt.Errorf("certificate and key paths must differ")
	}
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return err
	}
	now := time.Now()
	cert := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Yaplane Community development"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.AddDate(1, 0, 0),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true}
	for _, host := range strings.Split(hosts, ",") {
		host = strings.TrimSpace(host)
		if host == "" {
			return fmt.Errorf("hosts must not contain empty entries")
		}
		if ip := net.ParseIP(host); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, host)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, cert, cert, &private.PublicKey, private)
	if err != nil {
		return err
	}
	key, err := x509.MarshalPKCS8PrivateKey(private)
	if err != nil {
		return err
	}
	write := func(name, kind string, data []byte) error {
		if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return err
		}
		f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		if err := pem.Encode(f, &pem.Block{Type: kind, Bytes: data}); err != nil {
			f.Close()
			os.Remove(name)
			return err
		}
		return f.Close()
	}
	if err := write(keyPath, "PRIVATE KEY", key); err != nil {
		return err
	}
	if err := write(certPath, "CERTIFICATE", der); err != nil {
		os.Remove(keyPath)
		return err
	}
	return nil
}

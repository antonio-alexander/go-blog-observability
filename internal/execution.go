package internal

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func GenerateId() string {
	return uuid.Must(uuid.NewRandom()).String()
}

func GetCertificate(certFile, keyFile string) (tls.Certificate, error) {
	bytesCert, err := os.ReadFile(certFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	bytesKey, err := os.ReadFile(keyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair(bytesCert, bytesKey)
}

func GetCaCert(caCertFile string) (*x509.CertPool, error) {
	caCertPool := x509.NewCertPool()
	if caCertFile == "" {
		return caCertPool, nil
	}
	bytes, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, err
	}
	caCertPool.AppendCertsFromPEM(bytes)
	return caCertPool, nil
}

func GetTlsConfig(certFile, keyFile, caCertFile string) (*tls.Config, error) {
	caCertPool, err := GetCaCert(caCertFile)
	if err != nil {
		return nil, err
	}
	certificate, err := GetCertificate(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		// TLS versions below 1.2 are considered insecure
		// see https://www.rfc-editor.org/rfc/rfc7525.txt for details
		MinVersion:   tls.VersionTLS12,
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{certificate},
	}, nil
}

func GetPrivateSigningKey(privateSigningMethod, privateSigningKey, privateSigningKeyFile string) (any, error) {
	//create the method and keys
	switch jwt.GetSigningMethod(privateSigningMethod) {
	case jwt.SigningMethodHS256:
		return []byte(privateSigningKey), nil
	case jwt.SigningMethodRS256, jwt.SigningMethodRS384, jwt.SigningMethodRS512:
		switch {
		case privateSigningKey != "":
			return jwt.ParseRSAPrivateKeyFromPEM([]byte(privateSigningKey))
		case privateSigningKeyFile != "":
			bytes, err := os.ReadFile(privateSigningKeyFile)
			if err != nil {
				return nil, fmt.Errorf("error while reading private key file (%s): %w",
					privateSigningKeyFile, err)
			}
			return jwt.ParseRSAPrivateKeyFromPEM(bytes)
		}
	}
	return nil, errors.New("unsupported private signing method")
}

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

func LoadTLSCredentials(cfg TLSConfig) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
	}

	if cfg.CAFile != "" {
		caPem, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read ca cert: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPem) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.ClientCAs = pool
	}

	return credentials.NewTLS(tlsConfig), nil
}

func LoadClientTLS(certFile, keyFile, caFile string) (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("client cert: %w", err)
	}

	caPem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("ca cert: %w", err)
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caPem)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(tlsConfig), nil
}

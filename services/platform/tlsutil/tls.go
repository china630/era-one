// Package tlsutil — dev/prod mTLS helpers (GA-1 S5-8, GA-2 S6-9).
package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"

	"google.golang.org/grpc/credentials"
)

type ServerConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string // if set, require client certs (mTLS)
}

func (c ServerConfig) Enabled() bool {
	return c.CertFile != "" && c.KeyFile != ""
}

func (c ServerConfig) Load() (credentials.TransportCredentials, error) {
	if !c.Enabled() {
		return nil, nil
	}
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}
	tcfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	if c.CAFile != "" {
		pool, err := loadPool(c.CAFile)
		if err != nil {
			return nil, err
		}
		tcfg.ClientCAs = pool
		tcfg.ClientAuth = tls.RequireAndVerifyClientCert
	}
	return credentials.NewTLS(tcfg), nil
}

// HTTPServer возвращает http.Server с TLS, если заданы ERA_TLS_CERT/KEY.
func (c ServerConfig) HTTPServer(addr string, handler http.Handler) *http.Server {
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if c.Enabled() {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err == nil {
			srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
			if c.CAFile != "" {
				if pool, err := loadPool(c.CAFile); err == nil {
					srv.TLSConfig.ClientCAs = pool
					srv.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
				}
			}
		}
	}
	return srv
}

// Listen запускает HTTP или HTTPS в зависимости от конфигурации.
func (c ServerConfig) Listen(srv *http.Server) error {
	if srv.TLSConfig != nil {
		return srv.ListenAndServeTLS(c.CertFile, c.KeyFile)
	}
	return srv.ListenAndServe()
}

type ClientConfig struct {
	CAFile   string
	CertFile string
	KeyFile  string
}

func (c ClientConfig) Load() (credentials.TransportCredentials, error) {
	if c.CAFile == "" {
		return nil, nil
	}
	pool, err := loadPool(c.CAFile)
	if err != nil {
		return nil, err
	}
	tcfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, err
		}
		tcfg.Certificates = []tls.Certificate{cert}
	}
	return credentials.NewTLS(tcfg), nil
}

// TLSConfig для net/http клиента (mTLS).
func (c ClientConfig) TLSConfig() (*tls.Config, error) {
	if c.CAFile == "" {
		return nil, nil
	}
	pool, err := loadPool(c.CAFile)
	if err != nil {
		return nil, err
	}
	tcfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, err
		}
		tcfg.Certificates = []tls.Certificate{cert}
	}
	return tcfg, nil
}

// HTTPClient возвращает http.Client с mTLS, если задан ERA_TLS_CA.
func (c ClientConfig) HTTPClient(timeout time.Duration) (*http.Client, error) {
	tcfg, err := c.TLSConfig()
	if err != nil {
		return nil, err
	}
	if tcfg == nil {
		return &http.Client{Timeout: timeout}, nil
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tcfg,
		},
	}, nil
}

func ServerFromEnv() ServerConfig {
	return ServerConfig{
		CertFile: os.Getenv("ERA_TLS_CERT"),
		KeyFile:  os.Getenv("ERA_TLS_KEY"),
		CAFile:   os.Getenv("ERA_TLS_CA"),
	}
}

func ClientFromEnv() ClientConfig {
	return ClientConfig{
		CAFile:   os.Getenv("ERA_TLS_CA"),
		CertFile: os.Getenv("ERA_TLS_CLIENT_CERT"),
		KeyFile:  os.Getenv("ERA_TLS_CLIENT_KEY"),
	}
}

func loadPool(path string) (*x509.CertPool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(b) {
		return nil, fmt.Errorf("invalid CA pem")
	}
	return pool, nil
}

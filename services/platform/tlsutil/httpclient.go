package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
	"time"
)

// HTTPClient возвращает http.Client с mTLS, если заданы ERA_TLS_* переменные.
func HTTPClient(timeout time.Duration) (*http.Client, error) {
	cfg := ClientFromEnv()
	if cfg.CAFile == "" {
		return &http.Client{Timeout: timeout}, nil
	}
	pool, err := loadPool(cfg.CAFile)
	if err != nil {
		return nil, err
	}
	tcfg := &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client cert: %w", err)
		}
		tcfg.Certificates = []tls.Certificate{cert}
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tcfg,
		},
	}, nil
}

// InsecureDevHTTP — true если ERA_TLS_INSECURE=1 (только dev).
func InsecureDevHTTP() bool {
	return os.Getenv("ERA_TLS_INSECURE") == "1"
}

// DevHTTPClient — клиент для локальных тестов (опционально skip verify).
func DevHTTPClient(timeout time.Duration) *http.Client {
	if InsecureDevHTTP() {
		return &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // dev only
			},
		}
	}
	c, err := HTTPClient(timeout)
	if err != nil || c == nil {
		return &http.Client{Timeout: timeout}
	}
	return c
}

// AppendCA добавляет PEM CA в пул (для тестов).
func AppendCA(pool *x509.CertPool, pem []byte) bool {
	return pool.AppendCertsFromPEM(pem)
}

package tlsconfig

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"
)

// Provider загружает TLS-сертификаты из файлов, которые записывает Vault Agent.
// Сертификат перечитывается с диска при каждом handshake (кеш 1 минута),
// поэтому ротация Vault Agent подхватывается без перезапуска сервиса.
type Provider struct {
	certFile string
	keyFile  string
	caPool   *x509.CertPool

	mu      sync.Mutex
	cached  *tls.Certificate
	cacheAt time.Time
}

// New создаёт Provider. caFile загружается один раз при старте — CA не ротируется.
// certFile и keyFile перечитываются при каждом TLS handshake, поэтому Vault Agent
// может заменить их прозрачно для сервиса.
func New(certFile, keyFile, caFile string) (*Provider, error) {
	if certFile == "" || keyFile == "" || caFile == "" {
		return nil, fmt.Errorf("tlsconfig: cert, key and ca file paths are required")
	}

	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("tlsconfig: read ca file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("tlsconfig: no valid certificates in ca file %q", caFile)
	}

	p := &Provider{
		certFile: certFile,
		keyFile:  keyFile,
		caPool:   pool,
	}

	// Validate cert+key are readable at startup.
	if _, err := p.loadCert(); err != nil {
		return nil, err
	}

	return p, nil
}

// ServerConfig возвращает tls.Config для gRPC-серверов с принудительным mTLS.
func (p *Provider) ServerConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return p.cert()
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  p.caPool,
		MinVersion: tls.VersionTLS13,
	}
}

// ClientConfig возвращает tls.Config для исходящих gRPC-соединений с mTLS.
func (p *Provider) ClientConfig() *tls.Config {
	return &tls.Config{
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return p.cert()
		},
		RootCAs:    p.caPool,
		MinVersion: tls.VersionTLS13,
	}
}

func (p *Provider) cert() (*tls.Certificate, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cached != nil && time.Since(p.cacheAt) < time.Minute {
		return p.cached, nil
	}

	cert, err := p.loadCert()
	if err != nil {
		if p.cached != nil {
			// Отдаём устаревший сертификат, чтобы не обрывать активные соединения.
			return p.cached, nil
		}
		return nil, err
	}

	p.cached = cert
	p.cacheAt = time.Now()
	return cert, nil
}

func (p *Provider) loadCert() (*tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(p.certFile, p.keyFile)
	if err != nil {
		return nil, fmt.Errorf("tlsconfig: load key pair: %w", err)
	}
	return &cert, nil
}

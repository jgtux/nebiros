package masquetransport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	masque "github.com/quic-go/masque-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"
)

type ServerConfig struct {
	ListenAddr    string
	ProxyTemplate string
	AllowedTarget string
}

func ListenAndServe(ctx context.Context, cfg ServerConfig) error {
	tpl, err := uritemplate.New(cfg.ProxyTemplate)
	if err != nil {
		return fmt.Errorf("parse proxy template: %w", err)
	}

	u, err := url.Parse(cfg.ProxyTemplate)
	if err != nil {
		return fmt.Errorf("parse proxy template url: %w", err)
	}

	tlsConf, err := generateTLSConfig()
	if err != nil {
		return fmt.Errorf("generate tls config: %w", err)
	}

	proxy := &masque.Proxy{}
	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		req, err := masque.ParseRequest(r, tpl)
		if err != nil {
			var perr *masque.RequestParseError
			if errors.As(err, &perr) {
				w.WriteHeader(perr.HTTPStatus)
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if cfg.AllowedTarget != "" && !sameTarget(req.Target, cfg.AllowedTarget) {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if err := proxy.Proxy(w, req); err != nil && !errors.Is(err, net.ErrClosed) && ctx.Err() == nil {
			log.Printf("masque proxy error: %v", err)
		}
	})

	server := &http3.Server{
		Addr:            cfg.ListenAddr,
		Handler:         mux,
		TLSConfig:       http3.ConfigureTLSConfig(tlsConf),
		EnableDatagrams: true,
	}
	defer server.Close()
	defer proxy.Close()

	go func() {
		<-ctx.Done()
		_ = proxy.Close()
		_ = server.Close()
	}()

	return server.ListenAndServe()
}

func sameTarget(a, b string) bool {
	ah, ap, aerr := net.SplitHostPort(a)
	bh, bp, berr := net.SplitHostPort(b)
	if aerr != nil || berr != nil {
		return a == b
	}
	if ap != bp {
		return false
	}

	ah = strings.Trim(ah, "[]")
	bh = strings.Trim(bh, "[]")

	aip := net.ParseIP(ah)
	bip := net.ParseIP(bh)
	if aip != nil && bip != nil {
		return aip.Equal(bip)
	}

	return strings.EqualFold(ah, bh)
}

func generateTLSConfig() (*tls.Config, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return nil, err
	}

	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: "nebiros-masque",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3},
	}, nil
}

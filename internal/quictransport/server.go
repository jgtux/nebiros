package quictransport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	quic "github.com/quic-go/quic-go"
)

type ServerSession struct {
	conn *quic.Conn
}

func (s *ServerSession) Send(ctx context.Context, b []byte) error {
	_ = ctx
	return s.conn.SendDatagram(append([]byte(nil), b...))
}

func (s *ServerSession) Recv(ctx context.Context) ([]byte, error) {
	return s.conn.ReceiveDatagram(ctx)
}

func (s *ServerSession) Close() error {
	return s.conn.CloseWithError(0, "bye")
}

func Listen(listenAddr string) (*quic.Listener, error) {
	tlsConf, err := generateTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("generate tls config: %w", err)
	}

	quicConf := &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 20 * time.Second,
		MaxIdleTimeout:  60 * time.Second,
	}

	listener, err := quic.ListenAddr(listenAddr, tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("listen quic: %w", err)
	}
	return listener, nil
}

func Accept(ctx context.Context, l *quic.Listener) (*ServerSession, error) {
	conn, err := l.Accept(ctx)
	if err != nil {
		return nil, err
	}

	st := conn.ConnectionState()
	if !st.SupportsDatagrams.Remote || !st.SupportsDatagrams.Local {
		_ = conn.CloseWithError(1, "datagrams not supported")
		return nil, fmt.Errorf("peer does not support QUIC datagrams")
	}

	return &ServerSession{conn: conn}, nil
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
			CommonName: "nebiros-mvp",
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
		NextProtos:   []string{"nebiros-mvp"},
	}, nil
}

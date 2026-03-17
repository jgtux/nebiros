package quictransport

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	quic "github.com/quic-go/quic-go"
)

type Client struct {
	conn *quic.Conn
}

func NewClient(ctx context.Context, serverAddr, serverName string, insecure bool) (*Client, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: insecure,
		NextProtos:         []string{"nebiros-mvp"},
		ServerName:         serverName,
	}

	quicConf := &quic.Config{
		EnableDatagrams: true,
		KeepAlivePeriod: 20 * time.Second,
		MaxIdleTimeout:  60 * time.Second,
	}

	conn, err := quic.DialAddr(ctx, serverAddr, tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("dial quic: %w", err)
	}

	st := conn.ConnectionState()
	if !st.SupportsDatagrams.Remote || !st.SupportsDatagrams.Local {
		_ = conn.CloseWithError(1, "datagrams not supported")
		return nil, fmt.Errorf("peer does not support QUIC datagrams")
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Send(ctx context.Context, b []byte) error {
	_ = ctx
	return c.conn.SendDatagram(append([]byte(nil), b...))
}

func (c *Client) Recv(ctx context.Context) ([]byte, error) {
	return c.conn.ReceiveDatagram(ctx)
}

func (c *Client) Close() error {
	return c.conn.CloseWithError(0, "bye")
}

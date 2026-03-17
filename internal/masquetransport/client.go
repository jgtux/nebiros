package masquetransport

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	masque "github.com/quic-go/masque-go"
	quic "github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"
)

type ClientConfig struct {
	ProxyTemplate      string
	TargetAddr         string
	ServerName         string
	InsecureSkipVerify bool
}

type Client struct {
	client *masque.Client
	flow   net.PacketConn
}

func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	tpl, err := uritemplate.New(cfg.ProxyTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse proxy template: %w", err)
	}

	tlsConf := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		NextProtos:         []string{http3.NextProtoH3},
		ServerName:         cfg.ServerName,
	}

	cl := &masque.Client{
		TLSClientConfig: tlsConf,
		QUICConfig: &quic.Config{
			EnableDatagrams:   true,
			KeepAlivePeriod:   20 * time.Second,
			MaxIdleTimeout:    60 * time.Second,
			InitialPacketSize: 1350,
		},
	}

	flow, rsp, err := cl.DialAddr(ctx, tpl, cfg.TargetAddr)
	if err != nil {
		_ = cl.Close()
		return nil, fmt.Errorf("dial masque proxy: %w", err)
	}
	if rsp == nil {
		if flow != nil {
			_ = flow.Close()
		}
		_ = cl.Close()
		return nil, fmt.Errorf("dial masque proxy: missing HTTP response")
	}
	if rsp.StatusCode < 200 || rsp.StatusCode > 299 {
		if flow != nil {
			_ = flow.Close()
		}
		_ = cl.Close()
		return nil, fmt.Errorf("proxy rejected target %s: HTTP %d", cfg.TargetAddr, rsp.StatusCode)
	}

	return &Client{client: cl, flow: flow}, nil
}

func (c *Client) Send(ctx context.Context, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	_, err := c.flow.WriteTo(append([]byte(nil), b...), nil)
	return err
}

func (c *Client) Recv(ctx context.Context) ([]byte, error) {
	buf := make([]byte, 64*1024)
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = c.flow.SetReadDeadline(time.Now())
		case <-done:
		}
	}()

	n, _, err := c.flow.ReadFrom(buf)
	close(done)
	_ = c.flow.SetReadDeadline(time.Time{})
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	return append([]byte(nil), buf[:n]...), nil
}

func (c *Client) Close() error {
	return errors.Join(c.flow.Close(), c.client.Close())
}

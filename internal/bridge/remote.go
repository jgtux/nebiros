package bridge

import (
	"context"
	"fmt"
	"net"
	"time"

	"nebiros/internal/transport"
)

// RunRemoteBridge forwards packets from the transport to a fixed UDP target and
// sends target responses back into the transport.
func RunRemoteBridge(ctx context.Context, targetAddr string, t transport.PacketConn) error {
	udpTarget, err := net.ResolveUDPAddr("udp", targetAddr)
	if err != nil {
		return fmt.Errorf("resolve target udp addr: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpTarget)
	if err != nil {
		return fmt.Errorf("dial udp target: %w", err)
	}
	defer conn.Close()

	errCh := make(chan error, 2)

	go func() {
		for {
			msg, err := t.Recv(ctx)
			if err != nil {
				errCh <- fmt.Errorf("transport recv: %w", err)
				return
			}
			if len(msg) == 0 {
				continue
			}
			if _, err := conn.Write(msg); err != nil {
				errCh <- fmt.Errorf("write target udp: %w", err)
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					continue
				}
				errCh <- fmt.Errorf("read target udp: %w", err)
				return
			}
			if n == 0 {
				continue
			}
			payload := append([]byte(nil), buf[:n]...)
			if err := t.Send(ctx, payload); err != nil {
				errCh <- fmt.Errorf("transport send: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

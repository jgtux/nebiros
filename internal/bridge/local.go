package bridge

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"nebiros/internal/transport"
)

func RunLocalBridge(ctx context.Context, listenAddr string, t transport.PacketConn) error {
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return fmt.Errorf("resolve local udp addr: %w", err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("listen udp: %w", err)
	}
	defer conn.Close()

	var (
		peerMu     sync.RWMutex
		latestPeer *net.UDPAddr
		once       sync.Once
		errCh      = make(chan error, 1)
	)

	reportErr := func(err error) {
		if err == nil {
			return
		}
		once.Do(func() {
			errCh <- err
		})
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, peer, err := conn.ReadFromUDP(buf)
			if err != nil {
				if ctx.Err() != nil {
					reportErr(ctx.Err())
					return
				}
				reportErr(fmt.Errorf("read local udp: %w", err))
				return
			}
			if n == 0 {
				continue
			}

			peerCopy := *peer

			peerMu.Lock()
			latestPeer = &peerCopy
			peerMu.Unlock()

			payload := append([]byte(nil), buf[:n]...)
			if err := t.Send(ctx, payload); err != nil {
				if ctx.Err() != nil {
					reportErr(ctx.Err())
					return
				}
				reportErr(fmt.Errorf("transport send: %w", err))
				return
			}
		}
	}()

	go func() {
		for {
			msg, err := t.Recv(ctx)
			if err != nil {
				if ctx.Err() != nil {
					reportErr(ctx.Err())
					return
				}
				reportErr(fmt.Errorf("transport recv: %w", err))
				return
			}
			if len(msg) == 0 {
				continue
			}

			peerMu.RLock()
			var peerCopy *net.UDPAddr
			if latestPeer != nil {
				p := *latestPeer
				peerCopy = &p
			}
			peerMu.RUnlock()

			if peerCopy == nil {
				continue
			}

			if _, err := conn.WriteToUDP(msg, peerCopy); err != nil {
				if ctx.Err() != nil {
					reportErr(ctx.Err())
					return
				}
				reportErr(fmt.Errorf("write local udp: %w", err))
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		return err
	}
}

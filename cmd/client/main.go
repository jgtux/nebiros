package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"nebiros/internal/bridge"
	"nebiros/internal/config"
	"nebiros/internal/masquetransport"
	"nebiros/internal/quictransport"
	"nebiros/internal/transport"
)

func main() {
	cfg := config.ParseClient()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		if ctx.Err() != nil {
			log.Printf("shutdown requested")
			return
		}

		var (
			t   transport.PacketConn
			err error
		)

		switch cfg.Mode {
		case config.ModeQUIC:
			log.Printf("client mode=quic listen=%s server=%s", cfg.ListenAddr, cfg.ServerAddr)
			t, err = quictransport.NewClient(ctx, cfg.ServerAddr, cfg.ServerName, cfg.Insecure)
		case config.ModeMASQUE:
			log.Printf("client mode=masque listen=%s proxy=%s target=%s", cfg.ListenAddr, cfg.ProxyTemplate, cfg.TargetAddr)
			t, err = masquetransport.NewClient(ctx, masquetransport.ClientConfig{
				ProxyTemplate:      cfg.ProxyTemplate,
				TargetAddr:         cfg.TargetAddr,
				ServerName:         cfg.ServerName,
				InsecureSkipVerify: cfg.Insecure,
			})
		default:
			log.Fatalf("unsupported mode: %s", cfg.Mode)
		}

		if err != nil {
			log.Printf("connect failed: %v", err)
			if !sleepOrStop(ctx, 2*time.Second) {
				log.Printf("shutdown requested")
				return
			}
			continue
		}

		log.Printf("connected")
		err = bridge.RunLocalBridge(ctx, cfg.ListenAddr, t)

		if closeErr := t.Close(); closeErr != nil {
			log.Printf("transport close error: %v", closeErr)
		}

		if ctx.Err() != nil {
			log.Printf("shutdown requested")
			return
		}

		log.Printf("session ended: %v", err)
		if !sleepOrStop(ctx, 2*time.Second) {
			log.Printf("shutdown requested")
			return
		}
	}
}

func sleepOrStop(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

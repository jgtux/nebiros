package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"nebiros/internal/bridge"
	"nebiros/internal/config"
	"nebiros/internal/masquetransport"
	"nebiros/internal/quictransport"
)

func main() {
	cfg := config.ParseServer()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	switch cfg.Mode {
	case config.ModeQUIC:
		runQUIC(ctx, cfg)
	case config.ModeMASQUE:
		runMASQUE(ctx, cfg)
	default:
		log.Fatalf("unsupported mode: %s", cfg.Mode)
	}
}

func runQUIC(ctx context.Context, cfg config.ServerConfig) {
	listener, err := quictransport.Listen(cfg.ListenAddr)
	if err != nil {
		log.Fatalf("listen transport: %v", err)
	}
	defer listener.Close()

	log.Printf("server mode=quic listen=%s target=%s", cfg.ListenAddr, cfg.TargetAddr)

	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		sess, err := quictransport.Accept(ctx, listener)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Printf("shutdown requested")
				return
			}
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("accept timeout, stopping")
				return
			}
			if ctx.Err() != nil {
				log.Printf("listener closed, shutting down")
				return
			}
			log.Printf("accept error: %v", err)
			continue
		}

		go func() {
			defer sess.Close()
			if err := bridge.RunRemoteBridge(ctx, cfg.TargetAddr, sess); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("session ended: %v", err)
			}
		}()
	}
}

func runMASQUE(ctx context.Context, cfg config.ServerConfig) {
	log.Printf("server mode=masque listen=%s target=%s template=%s", cfg.ListenAddr, cfg.TargetAddr, cfg.ProxyTemplate)
	if err := masquetransport.ListenAndServe(ctx, masquetransport.ServerConfig{
		ListenAddr:    cfg.ListenAddr,
		ProxyTemplate: cfg.ProxyTemplate,
		AllowedTarget: cfg.TargetAddr,
	}); err != nil {
		if ctx.Err() != nil {
			log.Printf("shutdown requested")
			return
		}
		log.Fatalf("listen masque: %v", err)
	}
}

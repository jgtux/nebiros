package transport

import "context"

// PacketConn is the small transport abstraction shared by the bridge layer.
type PacketConn interface {
	Send(ctx context.Context, b []byte) error
	Recv(ctx context.Context) ([]byte, error)
	Close() error
}

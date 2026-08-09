package vole

import (
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/pnet"
)

type pnetPSKContextKey struct{}

func WithPnetPSK(ctx context.Context, psk pnet.PSK) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	cpy := make([]byte, len(psk))
	copy(cpy, psk)
	return context.WithValue(ctx, pnetPSKContextKey{}, pnet.PSK(cpy))
}

func pnetPSKFromContext(ctx context.Context) (pnet.PSK, bool) {
	if ctx == nil {
		return nil, false
	}
	psk, ok := ctx.Value(pnetPSKContextKey{}).(pnet.PSK)
	return psk, ok
}

func LoadPnetPSK(path string) (pnet.PSK, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read pnet swarm key %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	psk, err := pnet.DecodeV1PSK(f)
	if err != nil {
		return nil, fmt.Errorf("decode pnet swarm key %q: %w", path, err)
	}
	return psk, nil
}

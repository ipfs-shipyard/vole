package vole

import (
	"context"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

func libp2pHost(ctx context.Context) (host.Host, error) {
	opts := []libp2p.Option{
		libp2p.EnableHolePunching(),
	}
	if psk, ok := pnetPSKFromContext(ctx); ok {
		opts = append(opts, libp2p.PrivateNetwork(psk))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}
	return h, nil
}

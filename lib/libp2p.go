package vole

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(
		libp2p.EnableHolePunching(),
	)
	if err != nil {
		return nil, err
	}
	return h, nil
}

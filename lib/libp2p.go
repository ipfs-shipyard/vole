package vole

import (
	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

func libp2pHost() (host.Host, error) {
	// Lets us discover our own public address with a single observation
	identify.ActivationThresh = 1

	h, err := libp2p.New(libp2p.DefaultMuxers, libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport), libp2p.EnableHolePunching())
	if err != nil {
		return nil, err
	}
	return h, nil
}

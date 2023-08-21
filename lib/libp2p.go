package vole

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/muxer/yamux"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.DefaultMuxers, libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport))
	if err != nil {
		return nil, err
	}
	return h, nil
}

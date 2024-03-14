package vole

import (
	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/host"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.DefaultMuxers, libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport))
	if err != nil {
		return nil, err
	}
	return h, nil
}

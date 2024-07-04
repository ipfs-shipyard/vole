package vole

import (
	"github.com/libp2p/go-libp2p"
	mplex "github.com/libp2p/go-libp2p-mplex"
	"github.com/libp2p/go-libp2p/core/host"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.DefaultTransports, libp2p.Transport(libp2pwebrtc.New), libp2p.DefaultMuxers, libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport), libp2p.EnableHolePunching())
	if err != nil {
		return nil, err
	}
	return h, nil
}

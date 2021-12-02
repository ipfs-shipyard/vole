package vole

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	quic "github.com/libp2p/go-libp2p-quic-transport"
	"github.com/libp2p/go-tcp-transport"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.Transport(tcp.NewTCPTransport), libp2p.Transport(quic.NewTransport))
	if err != nil {
		return nil, err
	}
	return h, nil
}

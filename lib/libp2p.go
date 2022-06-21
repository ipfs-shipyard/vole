package vole

import (
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/libp2p/go-libp2p/p2p/transport/websocket"
)

func libp2pHost() (host.Host, error) {
	h, err := libp2p.New(libp2p.Transport(tcp.NewTCPTransport), libp2p.Transport(quic.NewTransport), libp2p.Transport(websocket.New))
	if err != nil {
		return nil, err
	}
	return h, nil
}

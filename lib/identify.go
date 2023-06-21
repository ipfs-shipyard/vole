package vole

import (
	"context"
	"sort"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

type IdentifyInfo struct {
	ProtocolVersion string
	AgentVersion    string
	Addresses       []multiaddr.Multiaddr
	Protocols       []protocol.ID
}

func IdentifyRequest(ctx context.Context, maStr string) (*IdentifyInfo, error) {
	ai, err := peer.AddrInfoFromString(maStr)
	if err != nil {
		return nil, err
	}

	h, err := libp2pHost()
	if err != nil {
		return nil, err
	}

	if err := h.Connect(ctx, *ai); err != nil {
		return nil, err
	}

	return extractIdentifyInfo(h.Peerstore(), ai.ID)
}

func extractIdentifyInfo(ps peerstore.Peerstore, p peer.ID) (*IdentifyInfo, error) {
	info := &IdentifyInfo{}

	addrInfo := ps.PeerInfo(p)
	addrs, err := peer.AddrInfoToP2pAddrs(&addrInfo)
	if err != nil {
		return nil, err
	}

	info.Addresses = addrs
	sort.Slice(info.Addresses, func(i, j int) bool { return info.Addresses[i].String() < info.Addresses[j].String() })

	protocols, err := ps.GetProtocols(p)
	if err != nil {
		return nil, err
	}
	info.Protocols = append(info.Protocols, protocols...)
	sort.Slice(info.Protocols, func(i, j int) bool { return info.Protocols[i] < info.Protocols[j] })

	if v, err := ps.Get(p, "ProtocolVersion"); err == nil {
		if vs, ok := v.(string); ok {
			info.ProtocolVersion = vs
		}
	}
	if v, err := ps.Get(p, "AgentVersion"); err == nil {
		if vs, ok := v.(string); ok {
			info.AgentVersion = vs
		}
	}

	return info, nil
}

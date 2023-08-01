package vole

import (
	"context"
	"errors"
	"sort"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/multiformats/go-multiaddr"
)

type IdentifyInfo struct {
	PeerId          peer.ID
	ProtocolVersion string
	AgentVersion    string
	Addresses       []multiaddr.Multiaddr
	Protocols       []protocol.ID
}

func IdentifyRequest(ctx context.Context, maStr string, allowUnknownPeer bool) (*IdentifyInfo, error) {
	usingBogusPeerID := false

	ai, err := peer.AddrInfoFromString(maStr)
	if err != nil {
		if !allowUnknownPeer {
			return nil, err
		}
		ma, err := multiaddr.NewMultiaddr(maStr)
		if err != nil {
			return nil, err
		}

		bogusPeerId, err := peer.Decode("QmadAdJ3f63JyNs65X7HHzqDwV53ynvCcKtNFvdNaz3nhk")
		if err != nil {
			panic("the hard coded bogus peerID is invalid")
		}
		usingBogusPeerID = true
		ai = &peer.AddrInfo{
			ID:    bogusPeerId,
			Addrs: []multiaddr.Multiaddr{ma},
		}
	}

	h, err := libp2pHost()
	if err != nil {
		return nil, err
	}

	if err := h.Connect(ctx, *ai); err != nil {
		if !usingBogusPeerID {
			return nil, err
		}
		newPeerId, err := extractPeerIDFromError(err)
		if err != nil {
			return nil, err
		}
		ai.ID = newPeerId
		if err := h.Connect(ctx, *ai); err != nil {
			return nil, err
		}
	}

	return extractIdentifyInfo(h.Peerstore(), ai.ID)
}

func extractPeerIDFromError(inputErr error) (peer.ID, error) {
	var dialErr *swarm.DialError
	if !errors.As(inputErr, &dialErr) {
		return "", inputErr
	}
	innerErr := dialErr.DialErrors[0].Cause

	var peerIDMismatchErr sec.ErrPeerIDMismatch
	if errors.As(innerErr, &peerIDMismatchErr) {
		return peerIDMismatchErr.Actual, nil
	}

	return "", inputErr
}

func extractIdentifyInfo(ps peerstore.Peerstore, p peer.ID) (*IdentifyInfo, error) {
	info := &IdentifyInfo{}

	addrInfo := ps.PeerInfo(p)
	addrs, err := peer.AddrInfoToP2pAddrs(&addrInfo)
	if err != nil {
		return nil, err
	}

	info.PeerId = addrInfo.ID
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

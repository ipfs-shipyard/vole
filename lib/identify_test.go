package vole

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	tls "github.com/libp2p/go-libp2p/p2p/security/tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	libp2pwebrtc "github.com/libp2p/go-libp2p/p2p/transport/webrtc"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	"github.com/multiformats/go-multiaddr"
)

func TestIdentifyRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const testProtoVersion = "test_protocol_version/0.0.1"
	const testAgent = "test_agent/0.0.1"

	var testMAs = []multiaddr.Multiaddr{
		multiaddr.StringCast("/onion3/vww6ybal4bd7szmgncyruucpgfkqahzddi37ktceo3ah7ngmcopnpyyd:80/http"),
		multiaddr.StringCast("/ip6zone/x/ip6/fe80::1/udp/1234/quic-v1"),
	}

	var testProtos = []protocol.ID{"/alpha/0.1.0", "/alpha/0.0.2", "/beta/0.0.1"}

	h, err := libp2p.New(
		libp2p.ProtocolVersion(testProtoVersion),
		libp2p.UserAgent(testAgent),
		libp2p.AddrsFactory(func(multiaddrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return append(append([]multiaddr.Multiaddr{}, multiaddrs...), testMAs...)
		}),
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/udp/0/quic-v1", "/ip6/::1/tcp/0"),
	)
	if err != nil {
		t.Fatal(err)
	}
	h.SetStreamHandler(testProtos[0], nil)
	h.SetStreamHandler(testProtos[1], nil)
	h.SetStreamHandler(testProtos[2], nil)

	hostAddrs, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := IdentifyRequest(ctx, hostAddrs[0].String(), false)
	if err != nil {
		t.Fatal(err)
	}

	if resp.ProtocolVersion != testProtoVersion {
		t.Fatalf("expected %q, got %q", testProtoVersion, resp.ProtocolVersion)
	}

	if resp.AgentVersion != testAgent {
		t.Fatalf("expected %q, got %q", testAgent, resp.AgentVersion)
	}

	listenAddrs := h.Network().ListenAddresses()
	numExpectedAddrs := len(testMAs) + len(listenAddrs) - 1 // test addresses + listen addresses - /p2p-circuit
	if len(resp.Addresses) != numExpectedAddrs {
		t.Fatalf("expected %d addrs, got %d", len(testMAs)+len(listenAddrs), len(resp.Addresses))
	}

	peerIDComp, err := multiaddr.NewComponent("p2p", h.ID().String())
	if err != nil {
		t.Fatal(err)
	}
	for i, a := range []multiaddr.Multiaddr{testMAs[1], testMAs[0]} {
		idx := i + len(listenAddrs) - 1 // addresses all come lexicographically after the (non p2p-cicruit) listen addresses
		foundAi := resp.Addresses[idx].Decapsulate(peerIDComp)
		if !foundAi.Equal(a) {
			t.Fatalf("expected %s, got %s", a, resp.Addresses[idx])
		}
	}

	// Assume our protocols come first lexicographically compared to the default libp2p ones (e.g. identify, relay, etc.)
	for i, a := range []protocol.ID{testProtos[1], testProtos[0], testProtos[2]} {
		if resp.Protocols[i] != a {
			t.Fatalf("expected %s, got %s", a, resp.Protocols[i])
		}
	}
}

func TestDiscoverPeerID(t *testing.T) {
	runTest := func(t *testing.T, h host.Host) {
		t.Helper()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		resp, err := IdentifyRequest(ctx, h.Addrs()[0].String(), true)
		if err != nil {
			t.Fatal(err)
		}
		if resp.PeerId != h.ID() {
			t.Fatalf("peerID mismatch: expected %s, got %s", h.ID(), resp.PeerId)
		}
	}

	t.Run("tcp+tls", func(t *testing.T) {
		h, err := libp2p.New(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Security(tls.ID, tls.New),
			libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, h)
	})
	t.Run("tcp+noise", func(t *testing.T) {
		h, err := libp2p.New(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Security(noise.ID, noise.New),
			libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, h)
	})
	t.Run("quic-v1", func(t *testing.T) {
		h, err := libp2p.New(
			libp2p.Transport(quic.NewTransport),
			libp2p.ListenAddrStrings("/ip4/127.0.0.1/udp/0/quic-v1"),
		)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, h)
	})
	t.Run("ws", func(t *testing.T) {
		h, err := libp2p.New(
			libp2p.Transport(ws.New),
			libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0/ws"),
		)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, h)
	})
	t.Run("webrtc", func(t *testing.T) {
		h, err := libp2p.New(
			libp2p.Transport(libp2pwebrtc.New),
			libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/webrtc-direct"),
		)
		if err != nil {
			t.Fatal(err)
		}
		runTest(t, h)
	})
}

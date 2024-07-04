package vole

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"github.com/multiformats/go-multiaddr"
)

func Ping(ctx context.Context, forceRelay bool, p *peer.AddrInfo) error {
	if forceRelay {
		// We don't want a direct connection, so set this to a high value so
		// that we don't learn our public address
		identify.ActivationThresh = 100

		for _, addr := range p.Addrs {
			if _, err := addr.ValueForProtocol(multiaddr.P_CIRCUIT); err != nil {
				return fmt.Errorf("force-relay=true but peer is not using a relayed address")
			}
		}
	}

	h, err := libp2pHost()
	if err != nil {
		return err
	}
	defer h.Close()
	h.Connect(ctx, *p)

	times := 3

	if forceRelay {
		ctx = network.WithAllowLimitedConn(ctx, "ping")
	}

	// Reimplementing ping because the default implementation may use a relayed connection instead of a direct one
	in := [32]byte{}
	out := [32]byte{}
	s, err := h.NewStream(ctx, p.ID, "/ipfs/ping/1.0.0")
	if err != nil {
		return err
	}

	for i := 0; i < times; i++ {
		_, err := rand.Reader.Read(in[:])
		if err != nil {
			return err
		}

		start := time.Now()
		s.Write(in[:])
		n, err := s.Read(out[:])
		if err != nil {
			return err
		}
		if n != 32 {
			return fmt.Errorf("expected 32 bytes, got %d", n)
		}
		if !bytes.Equal(in[:], out[:]) {
			return fmt.Errorf("expected %x, got %x", in[:], out[:])
		}

		fmt.Println("Took ", time.Since(start))
		time.Sleep(time.Second)
	}

	return nil
}

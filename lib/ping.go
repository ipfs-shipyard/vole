package vole

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

func Ping(ctx context.Context, direct bool, p *peer.AddrInfo) error {
	// Set the activation threshold to 1 so that we learn our own address with just one observation. This lets us holepunch at all.
	if direct {
		identify.ActivationThresh = 1
	}

	h, err := libp2p.New(libp2p.EnableHolePunching())
	if err != nil {
		return err
	}
	defer h.Close()
	h.Connect(ctx, *p)

	times := 3

	if !direct {
		ctx = network.WithUseTransient(ctx, "ping")
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

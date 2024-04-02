package vole

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestPing(t *testing.T) {
	h, err := libp2p.New()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()

	p := peer.AddrInfo{
		ID:    h.ID(),
		Addrs: h.Addrs(),
	}

	ctx := context.Background()
	if err := Ping(ctx, true, &p); err != nil {
		t.Fatal(err)
	}
}

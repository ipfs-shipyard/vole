package main

import (
	"bytes"
	"context"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	record "github.com/libp2p/go-libp2p-record"
	"testing"
)

func TestDhtPutGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	h, err := libp2p.New(ctx)
	if err != nil {
		t.Fatal(err)
	}

	hostAddrs, err := peer.AddrInfoToP2pAddrs(&peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()})
	if err != nil {
		t.Fatal(err)
	}

	nsval := record.NamespacedValidator{}
	nsval["testval"] = &testVal{}

	d, err := dht.New(ctx, h, dht.Mode(dht.ModeServer), dht.ProtocolPrefix("/test"), dht.Validator(nsval))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	k := []byte("/testval/fookey")
	v := []byte("the data")
	proto := protocol.ID("/test/kad/1.0.0")
	if err := dhtPut(ctx, k, v, proto, hostAddrs[0]); err != nil {
		t.Fatal(err)
	}

	rec, err := dhtGet(ctx, k, proto, hostAddrs[0])
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(rec.GetKey(), k) {
		t.Fatal("record keys not equal")
	}

	if !bytes.Equal(rec.GetValue(), v) {
		t.Fatal("record values not equal")
	}
}

type testVal struct{}

func (n *testVal) Validate(key string, value []byte) error {
	return nil
}

func (n *testVal) Select(key string, values [][]byte) (int, error) {
	panic("implement me")
}

var _ record.Validator = (*testVal)(nil)

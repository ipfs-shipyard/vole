package vole

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ipfs/go-bitswap"
	bsnet "github.com/ipfs/go-bitswap/network"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	nrouting "github.com/ipfs/go-ipfs-routing/none"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
)

func TestBitswapCheckPresent(t *testing.T) {
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

	nilRouter, err := nrouting.ConstructNilRouting(context.TODO(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	bsnetwork := bsnet.NewFromIpfsHost(h, nilRouter)
	bstore := blockstore.NewBlockstore(datastore.NewMapDatastore())

	data := []byte("existing data")
	blk := getBlock(t, data)

	if err := bstore.Put(blk); err != nil {
		t.Fatal(err)
	}

	_ = bitswap.New(ctx, bsnetwork, bstore)

	checkOutput, err := CheckBitswapCID(ctx, blk.Cid(), hostAddrs[0])
	if err != nil {
		t.Fatal(err)
	}

	if checkOutput.Error != nil || !checkOutput.Responded || !checkOutput.Found {
		jsOut, err := json.Marshal(checkOutput)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("expected the data to be reported as found, instead got %v", jsOut)
	}
}

func TestBitswapCheckNotPresent(t *testing.T) {
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

	nilRouter, err := nrouting.ConstructNilRouting(context.TODO(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	bsnetwork := bsnet.NewFromIpfsHost(h, nilRouter)
	bstore := blockstore.NewBlockstore(datastore.NewMapDatastore())

	data := []byte("missing data")
	blk := getBlock(t, data)

	_ = bitswap.New(ctx, bsnetwork, bstore)

	checkOutput, err := CheckBitswapCID(ctx, blk.Cid(), hostAddrs[0])
	if err != nil {
		t.Fatal(err)
	}

	if checkOutput.Error != nil || !checkOutput.Responded || checkOutput.Found {
		jsOut, err := json.Marshal(checkOutput)
		if err != nil {
			t.Fatal(err)
		}
		t.Fatalf("expected the data to be reported as not found, instead got %v", jsOut)
	}
}

func getBlock(t *testing.T, data []byte) blocks.Block {
	t.Helper()
	mh, err := multihash.Sum(data, multihash.SHA2_256, -1)
	if err != nil {
		t.Fatal(err)
	}
	cData := cid.NewCidV1(cid.Raw, mh)

	blk, err := blocks.NewBlockWithCid(data, cData)
	if err != nil {
		t.Fatal(err)
	}
	return blk
}

package vole

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	// importing so we can traverse dag-cbor and dag-json nodes in the `bitswap get` command
	"github.com/cheggaaa/pb/v3"
	_ "github.com/ipld/go-ipld-prime/codec/dagcbor"
	_ "github.com/ipld/go-ipld-prime/codec/dagjson"

	"github.com/ipfs/boxo/bitswap/network"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/ipfs/boxo/bitswap"
	bsmsg "github.com/ipfs/boxo/bitswap/message"
	bsmsgpb "github.com/ipfs/boxo/bitswap/message/pb"
	bsnet "github.com/ipfs/boxo/bitswap/network/bsnet"
	"github.com/ipfs/boxo/blockservice"
	blockstore "github.com/ipfs/boxo/blockstore"
	"github.com/ipfs/boxo/ipld/merkledag"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	rhelp "github.com/libp2p/go-libp2p-routing-helpers"
)

type BsCheckOutput struct {
	Found     bool
	Responded bool
	Error     error
}

func (o *BsCheckOutput) MarshalJSON() ([]byte, error) {
	var errorMsg *string
	if o.Error != nil {
		m := o.Error.Error()
		errorMsg = &m
	}
	anon := struct {
		Found     bool
		Responded bool
		Error     *string
	}{
		Found:     o.Found,
		Responded: o.Responded,
		Error:     errorMsg,
	}
	return json.Marshal(anon)
}

var _ json.Marshaler = (*BsCheckOutput)(nil)

// Passing a libp2p host is optional. Otherwise a temporary host will be created.
// When passing a host, it should be only connceted to the passed multiaddr
func CheckBitswapCID(ctx context.Context, h host.Host, c cid.Cid, ma multiaddr.Multiaddr, getBlock bool) (*BsCheckOutput, error) {
	var err error
	if h == nil {
		h, err = libp2pHost(ctx)
	}

	if err != nil {
		return nil, err
	}

	ai, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, err
	}

	if err := h.Connect(ctx, *ai); err != nil {
		return nil, err
	}

	tctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	// Create a new stream to ensure we wait for hole punching even if it takes longer than the built-in limit in the Bitswap implementation
	_, err = h.NewStream(tctx, ai.ID, "/ipfs/bitswap/1.2.0", "/ipfs/bitswap/1.1.0", "/ipfs/bitswap/1.0.0", "/ipfs/bitswap")
	if err != nil {
		return nil, err
	}

	target := ai.ID

	bs := bsnet.NewFromIpfsHost(h)
	msg := bsmsg.New(false)

	wantType := bsmsgpb.Message_Wantlist_Have
	if getBlock {
		wantType = bsmsgpb.Message_Wantlist_Block
	}
	msg.AddEntry(c, 0, wantType, true)

	rcv := &bsReceiver{
		target: target,
		result: make(chan msgOrErr),
	}

	bs.Start(rcv)
	defer bs.Stop()

	if err := bs.SendMessage(ctx, target, msg); err != nil {
		return nil, err
	}

	// in case for some reason we're sent a bunch of messages (e.g. wants) from a peer without them responding to our query
	// FIXME: Why would this be the case?
	sctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
loop:
	for {
		var res msgOrErr
		select {
		case res = <-rcv.result:
		case <-sctx.Done():
			break loop
		}

		if res.err != nil {
			return &BsCheckOutput{
				Found:     false,
				Responded: true,
				Error:     res.err,
			}, nil
		}

		if res.msg == nil {
			panic("should not be reachable")
		}

		for _, msgC := range res.msg.Blocks() {
			if msgC.Cid().Equals(c) {
				return &BsCheckOutput{
					Found:     true,
					Responded: true,
					Error:     nil,
				}, nil
			}
		}

		for _, msgC := range res.msg.Haves() {
			if msgC.Equals(c) {
				return &BsCheckOutput{
					Found:     true,
					Responded: true,
					Error:     nil,
				}, nil
			}
		}

		for _, msgC := range res.msg.DontHaves() {
			if msgC.Equals(c) {
				return &BsCheckOutput{
					Found:     false,
					Responded: true,
					Error:     nil,
				}, nil
			}
		}
	}

	return &BsCheckOutput{
		Found:     false,
		Responded: false,
		Error:     nil,
	}, nil
}

func GetBitswapCID(ctx context.Context, root cid.Cid, ai *peer.AddrInfo) error {
	h, err := libp2pHost(ctx)
	if err != nil {
		return err
	}

	ds := sync.MutexWrap(datastore.NewMapDatastore())
	bstore := blockstore.NewBlockstore(ds)

	bsnet := bsnet.NewFromIpfsHost(h)
	bswap := bitswap.New(ctx, bsnet, &rhelp.Null{}, bstore)

	bserv := blockservice.New(bstore, bswap)
	dag := merkledag.NewDAGService(bserv)

	// connect to our peer
	if err := h.Connect(ctx, *ai); err != nil {
		return fmt.Errorf("failed to connect to target peer: %w", err)
	}

	bar := pb.StartNew(-1)
	bar.Set(pb.Bytes, true)

	cset := cid.NewSet()

	getLinks := func(ctx context.Context, c cid.Cid) ([]*format.Link, error) {
		node, err := dag.Get(ctx, c)
		if err != nil {
			return nil, err
		}
		bar.Add(len(node.RawData()))

		return node.Links(), nil

	}
	if err := merkledag.Walk(ctx, getLinks, root, cset.Visit, merkledag.Concurrency(500)); err != nil {
		return err
	}

	bar.Finish()

	return nil
}

type bsReceiver struct {
	target peer.ID
	result chan msgOrErr
}

type msgOrErr struct {
	msg bsmsg.BitSwapMessage
	err error
}

func (r *bsReceiver) ReceiveMessage(ctx context.Context, sender peer.ID, incoming bsmsg.BitSwapMessage) {
	if r.target != sender {
		select {
		case <-ctx.Done():
		case r.result <- msgOrErr{err: fmt.Errorf("expected peerID %v, got %v", r.target, sender)}:
		}
		return
	}

	select {
	case <-ctx.Done():
	case r.result <- msgOrErr{msg: incoming}:
	}
}

func (r *bsReceiver) ReceiveError(err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	select {
	case <-ctx.Done():
	case r.result <- msgOrErr{err: err}:
	}
}

func (r *bsReceiver) PeerConnected(id peer.ID) {}

func (r *bsReceiver) PeerDisconnected(id peer.ID) {}

var _ network.Receiver = (*bsReceiver)(nil)

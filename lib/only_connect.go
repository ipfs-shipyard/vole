package vole

import (
	"context"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
)

func OnlyConnect(ctx context.Context, p *peer.AddrInfo) error {
	h, err := libp2pHost()
	if err != nil {
		return err
	}
	defer h.Close()

	start := time.Now()
	h.Peerstore().AddAddrs(p.ID, p.Addrs, peerstore.TempAddrTTL)
	_, err = h.Network().DialPeer(ctx, p.ID)
	if err != nil {
		return err
	}
	connectTime := time.Since(start)
	fmt.Println("Connect took:", connectTime)

	pingService := ping.NewPingService(h)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	resCh := pingService.Ping(ctx, p.ID)
	var avg time.Duration
	trials := 3
	for i := 0; i < trials; i++ {
		res := <-resCh
		if res.Error != nil {
			return res.Error
		}
		avg += res.RTT
	}
	avg /= time.Duration(trials)
	fmt.Println("Average RTT:", avg)
	fmt.Println("Number of roundtrips to Connect:", float64(connectTime)/float64(avg))
	return nil
}

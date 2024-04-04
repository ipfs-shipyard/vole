package vole

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2phttp "github.com/libp2p/go-libp2p/p2p/http"
	"github.com/multiformats/go-multiaddr"
)

func Libp2pHTTPSocketProxy(ctx context.Context, p multiaddr.Multiaddr, unixSocketPath string) error {
	h, err := libp2pHost()
	if err != nil {
		return err
	}

	httpHost := libp2phttp.Host{StreamHost: h}

	ai := peer.AddrInfo{
		Addrs: []multiaddr.Multiaddr{p},
	}
	idStr, err := p.ValueForProtocol(multiaddr.P_P2P)
	if err == nil {
		id, err := peer.Decode(idStr)
		if err != nil {
			return err
		}
		ai.ID = id
	}

	rt, err := httpHost.NewConstrainedRoundTripper(ai)
	if err != nil {
		return err
	}
	rp := &httputil.ReverseProxy{
		Transport: rt,
		Director:  func(r *http.Request) {},
	}

	// Serves an HTTP server on the given path using unix sockets
	server := &http.Server{
		Handler: rp,
	}

	l, err := net.Listen("unix", unixSocketPath)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		server.Close()
	}()

	return server.Serve(l)
}

// Libp2pHTTPServer serves an libp2p enabled HTTP server
func Libp2pHTTPServer() (host.Host, *libp2phttp.Host, error) {
	h, err := libp2pHost()
	if err != nil {
		return nil, nil, err
	}

	httpHost := &libp2phttp.Host{StreamHost: h}
	return h, httpHost, nil
}

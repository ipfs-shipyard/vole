package vole

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

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

	ai, err := peer.AddrInfoFromP2pAddr(p)
	if err == peer.ErrInvalidAddr {
		ai = &peer.AddrInfo{Addrs: []multiaddr.Multiaddr{p}} // No peer id
		err = nil
	}
	if err != nil {
		return err
	}

	hasTLS := false
	hasHTTP := false
	host := ""
	port := 0
	multiaddr.ForEach(p, func(c multiaddr.Component) bool {
		switch c.Protocol().Code {
		case multiaddr.P_TLS:
			hasTLS = true
		case multiaddr.P_HTTP:
			hasHTTP = true
		case multiaddr.P_HTTPS:
			hasHTTP = true
			hasTLS = true
		case multiaddr.P_IP4, multiaddr.P_IP6, multiaddr.P_DNS4, multiaddr.P_DNS6, multiaddr.P_DNS:
			host = c.Value()
		case multiaddr.P_TCP, multiaddr.P_UDP:
			port, err = strconv.Atoi(c.Value())
			return false
		}
		return true
	})
	if err != nil {
		return err
	}
	if port == 0 && hasHTTP {
		port = 80
		if hasTLS {
			port = 443
		}
	}

	rt, err := httpHost.NewConstrainedRoundTripper(*ai)
	if err != nil {
		return err
	}

	var rp http.Handler
	if hasTLS && hasHTTP {
		u, err := url.Parse("https://" + host + ":" + strconv.Itoa(port) + "/")
		if err != nil {
			return err
		}
		revProxy := httputil.NewSingleHostReverseProxy(u)
		rp = revProxy
	} else {
		rp = &httputil.ReverseProxy{
			Transport: rt,
			Director:  func(r *http.Request) {},
		}
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

	if hasTLS && hasHTTP {
		c, err := selfSignedTLSConfig()
		if err != nil {

			return err
		}
		server.TLSConfig = c

		fmt.Println("Endpoint is an HTTPS endpoint. Using a self signed cert locally to proxy.")
		fmt.Println("Curl will only work with -k flag. This is only for debugging. Do *not* use this in production.")

		return server.ServeTLS(l, "", "")
	}

	return server.Serve(l)
}

// libp2pHTTPServer serves an libp2p enabled HTTP server
func libp2pHTTPServer() (host.Host, *libp2phttp.Host, error) {
	h, err := libp2pHost()
	if err != nil {
		return nil, nil, err
	}

	httpHost := &libp2phttp.Host{StreamHost: h}
	return h, httpHost, nil
}

func selfSignedTLSConfig() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	return tlsConfig, nil
}

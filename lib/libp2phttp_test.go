package vole

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/multiformats/go-multiaddr"
)

func TestHTTPProxyAndServer(t *testing.T) {
	// Start libp2p HTTP server
	h, hh, err := libp2pHTTPServer()
	if err != nil {
		t.Fatal(err)
	}
	hh.SetHTTPHandlerAtPath("/hello", "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	go hh.Serve()
	defer hh.Close()

	serverAddr := h.Addrs()[0].Encapsulate(multiaddr.StringCast("/p2p/" + h.ID().String()))
	port, err := serverAddr.ValueForProtocol(multiaddr.P_TCP)
	if err != nil || port == "" {
		port, err = serverAddr.ValueForProtocol(multiaddr.P_UDP)
		if err != nil || port == "" {
			t.Fatal("could not get port from server address")
		}
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	socketFile, err := os.CreateTemp("", "libp2phttp-*.sock")
	if err != nil {
		t.Fatal(err)
	}

	socketFile.Close()
	os.Remove(socketFile.Name())

	go func() {
		err := Libp2pHTTPSocketProxy(ctx, serverAddr, socketFile.Name())
		if err != http.ErrServerClosed && err != nil {
			panic(err)
		}
	}()

	// Wait a bit to let the proxy start up.
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		_, err := os.Stat(socketFile.Name())
		if err == nil {
			break
		}
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketFile.Name())
			},
		},
	}

	resp, err := client.Get("http://127.0.0.1:" + port + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

func TestHTTPProxyAndServerOverHTTPTransport(t *testing.T) {
	// Start a basic http server
	s := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	}
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve(l)
	defer s.Close()

	// get port of listener
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	serverAddr := multiaddr.StringCast("/ip4/127.0.0.1/tcp/" + port + "/http")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	socketFile, err := os.CreateTemp("", "libp2phttp-*.sock")
	if err != nil {
		t.Fatal(err)
	}

	socketFile.Close()
	os.Remove(socketFile.Name())

	go func() {
		err := Libp2pHTTPSocketProxy(ctx, serverAddr, socketFile.Name())
		if err != http.ErrServerClosed && err != nil {
			panic(err)
		}
	}()

	// Wait a bit to let the proxy start up.
	for i := 0; i < 10; i++ {
		time.Sleep(100 * time.Millisecond)
		_, err := os.Stat(socketFile.Name())
		if err == nil {
			break
		}
	}

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketFile.Name())
			},
		},
	}

	resp, err := client.Get("http://127.0.0.1:" + port + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", resp.StatusCode)
	}
}

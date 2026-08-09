package vole

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	corepnet "github.com/libp2p/go-libp2p/core/pnet"
)

const testSwarmKeyV1 = "/key/swarm/psk/1.0.0/\n/base16/\n0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
const testSwarmKeyV1Alt = "/key/swarm/psk/1.0.0/\n/base16/\nfedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210\n"

func writeTempFile(t *testing.T, contents string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "swarmkey-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.WriteString(contents); err != nil {
		_ = f.Close()
		t.Fatalf("WriteString: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return f.Name()
}

func TestLoadPnetPSK_MissingFile(t *testing.T) {
	_, err := LoadPnetPSK("does-not-exist")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadPnetPSK_InvalidFile(t *testing.T) {
	path := writeTempFile(t, "not a swarm key")
	_, err := LoadPnetPSK(path)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestPnet_AllowsOnlySameKey(t *testing.T) {
	path := writeTempFile(t, testSwarmKeyV1)
	psk, err := LoadPnetPSK(path)
	if err != nil {
		t.Fatalf("LoadPnetPSK: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hNo, err := libp2pHost(ctx)
	if err != nil {
		t.Fatalf("libp2pHost(no pnet): %v", err)
	}
	defer hNo.Close()

	hYesA, err := libp2pHost(WithPnetPSK(ctx, psk))
	if err != nil {
		t.Fatalf("libp2pHost(pnet A): %v", err)
	}
	defer hYesA.Close()

	hYesB, err := libp2pHost(WithPnetPSK(ctx, psk))
	if err != nil {
		t.Fatalf("libp2pHost(pnet B): %v", err)
	}
	defer hYesB.Close()

	// pnet->no-pnet should fail.
	aiNo := peer.AddrInfo{ID: hNo.ID(), Addrs: hNo.Addrs()}
	dialCtx, dialCancel := context.WithTimeout(ctx, 3*time.Second)
	err = hYesA.Connect(dialCtx, aiNo)
	dialCancel()
	if err == nil {
		t.Fatalf("expected pnet dial to non-pnet host to fail")
	}
	// Depending on the transport/OS, the pnet failure may be wrapped in dial/security negotiation errors.
	if !corepnet.IsPNetError(err) && !strings.Contains(strings.ToLower(err.Error()), "privnet") {
		// Don't fail the test on classification; the key property is that the dial fails.
		t.Logf("dial failed with non-pnet-classified error: %T: %v", err, err)
	}

	// pnet->pnet (same key) should succeed.
	aiYesB := peer.AddrInfo{ID: hYesB.ID(), Addrs: hYesB.Addrs()}
	dialCtx2, dialCancel2 := context.WithTimeout(ctx, 5*time.Second)
	err = hYesA.Connect(dialCtx2, aiYesB)
	dialCancel2()
	if err != nil {
		// If this fails, surface the most useful root cause.
		var pnetErr corepnet.Error
		if errors.As(err, &pnetErr) {
			t.Fatalf("unexpected pnet error connecting same-key hosts: %v", err)
		}
		t.Fatalf("expected same-key hosts to connect, got: %v", err)
	}
}

func TestPnet_DifferentKeysFail(t *testing.T) {
	pathA := writeTempFile(t, testSwarmKeyV1)
	pskA, err := LoadPnetPSK(pathA)
	if err != nil {
		t.Fatalf("LoadPnetPSK(A): %v", err)
	}

	pathB := writeTempFile(t, testSwarmKeyV1Alt)
	pskB, err := LoadPnetPSK(pathB)
	if err != nil {
		t.Fatalf("LoadPnetPSK(B): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hA, err := libp2pHost(WithPnetPSK(ctx, pskA))
	if err != nil {
		t.Fatalf("libp2pHost(pnet A): %v", err)
	}
	defer hA.Close()

	hB, err := libp2pHost(WithPnetPSK(ctx, pskB))
	if err != nil {
		t.Fatalf("libp2pHost(pnet B): %v", err)
	}
	defer hB.Close()

	aiB := peer.AddrInfo{ID: hB.ID(), Addrs: hB.Addrs()}
	dialCtx, dialCancel := context.WithTimeout(ctx, 3*time.Second)
	err = hA.Connect(dialCtx, aiB)
	dialCancel()
	if err == nil {
		t.Fatalf("expected different-key pnet connect to fail")
	}
}

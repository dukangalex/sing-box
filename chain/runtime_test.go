package chain

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type runtimeTestDialer struct {
	mu          sync.Mutex
	dialCalls   int
	lastNetwork string
	lastTarget  M.Socksaddr
}

func (d *runtimeTestDialer) DialContext(_ context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	d.mu.Lock()
	d.dialCalls++
	d.lastNetwork = network
	d.lastTarget = destination
	d.mu.Unlock()
	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}

func (d *runtimeTestDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented in test dialer")
}

type testEntryResolver struct {
	mu      sync.Mutex
	entries []N.Dialer
	index   int
	calls   int
}

func (r *testEntryResolver) ResolveEntry(context.Context) (N.Dialer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if len(r.entries) == 0 {
		return nil, errors.New("no entry")
	}
	entry := r.entries[r.index%len(r.entries)]
	r.index++
	return entry, nil
}

type testLandingFactory struct {
	mu       sync.Mutex
	entries  []N.Dialer
	landings []*runtimeTestDialer
	newCalls int
}

func (f *testLandingFactory) NewLanding(_ context.Context, entry N.Dialer) (N.Dialer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.newCalls++
	f.entries = append(f.entries, entry)
	landing := &runtimeTestDialer{}
	f.landings = append(f.landings, landing)
	return landing, nil
}

func TestRuntimeResolvesEntryPerConnection(t *testing.T) {
	entryA := &runtimeTestDialer{}
	entryB := &runtimeTestDialer{}
	factory := &testLandingFactory{}
	resolver := &testEntryResolver{entries: []N.Dialer{entryA, entryB}}
	runtime := NewRuntime(resolver, factory)

	firstTarget := M.Socksaddr{Fqdn: "example.com", Port: 443}
	secondTarget := M.Socksaddr{Fqdn: "example.org", Port: 443}

	if conn, err := runtime.DialContext(context.Background(), "tcp", firstTarget); err != nil {
		t.Fatalf("first chain dial: %v", err)
	} else {
		_ = conn.Close()
	}
	if conn, err := runtime.DialContext(context.Background(), "tcp", secondTarget); err != nil {
		t.Fatalf("second chain dial: %v", err)
	} else {
		_ = conn.Close()
	}

	if resolver.calls != 2 {
		t.Fatalf("expected 2 entry resolutions, got %d", resolver.calls)
	}
	if len(factory.entries) != 2 {
		t.Fatalf("expected 2 landing constructions, got %d", len(factory.entries))
	}
	if factory.entries[0] != entryA {
		t.Fatal("first connection did not use entry A")
	}
	if factory.entries[1] != entryB {
		t.Fatal("second connection did not use entry B")
	}
	if factory.landings[0].dialCalls != 1 || factory.landings[0].lastTarget != firstTarget {
		t.Fatal("first connection did not reach its landing dialer")
	}
	if factory.landings[1].dialCalls != 1 || factory.landings[1].lastTarget != secondTarget {
		t.Fatal("second connection did not reach its landing dialer")
	}
}

func TestRuntimeBlocksUnavailableEntry(t *testing.T) {
	runtime := NewRuntime(&testEntryResolver{}, &testLandingFactory{})
	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrEntryUnavailable) {
		t.Fatalf("expected ErrEntryUnavailable, got %v", err)
	}
}

func TestRuntimePropagatesEntryError(t *testing.T) {
	entryErr := errors.New("entry resolver failed")
	runtime := NewRuntime(&failingEntryResolver{err: entryErr}, &testLandingFactory{})
	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrEntryUnavailable) {
		t.Fatalf("expected ErrEntryUnavailable, got %v", err)
	}
	if !errors.Is(err, entryErr) {
		t.Fatalf("expected original entry error to be preserved, got %v", err)
	}
}

type failingEntryResolver struct {
	err error
}

func (r *failingEntryResolver) ResolveEntry(context.Context) (N.Dialer, error) {
	return nil, r.err
}

func TestRuntimeBlocksUnavailableLanding(t *testing.T) {
	entry := &runtimeTestDialer{}
	resolver := &testEntryResolver{entries: []N.Dialer{entry}}
	factory := &failingLandingFactory{}
	runtime := NewRuntime(resolver, factory)

	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrLandingUnavailable) {
		t.Fatalf("expected ErrLandingUnavailable, got %v", err)
	}
}

type failingLandingFactory struct{}

func (*failingLandingFactory) NewLanding(context.Context, N.Dialer) (N.Dialer, error) {
	return nil, errors.New("landing unavailable")
}

func TestRuntimeDisabled(t *testing.T) {
	runtime := NewRuntime(&testEntryResolver{}, &testLandingFactory{})
	runtime.SetEnabled(false)
	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrDisabled) {
		t.Fatalf("expected ErrDisabled, got %v", err)
	}
}

func TestRuntimeRejectsUDP(t *testing.T) {
	entry := &runtimeTestDialer{}
	factory := &testLandingFactory{}
	runtime := NewRuntime(&testEntryResolver{entries: []N.Dialer{entry}}, factory)

	_, err := runtime.DialContext(context.Background(), "udp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrUnsupportedNetwork) {
		t.Fatalf("expected ErrUnsupportedNetwork, got %v", err)
	}
	if len(factory.landings) != 0 {
		t.Fatal("UDP rejection must happen before constructing a landing")
	}
}

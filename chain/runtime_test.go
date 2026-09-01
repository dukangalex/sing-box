package chain

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"

	M "github.com/sagernet/sing/common/metadata"
)

type testDialer struct {
	mu          sync.Mutex
	dialCalls   int
	lastNetwork string
	lastTarget  M.Socksaddr
}

func (d *testDialer) DialContext(_ context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	d.mu.Lock()
	d.dialCalls++
	d.lastNetwork = network
	d.lastTarget = destination
	d.mu.Unlock()
	client, server := net.Pipe()
	_ = server.Close()
	return client, nil
}

func (d *testDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented in test dialer")
}

type testEntryResolver struct {
	mu      sync.Mutex
	entries []netDialer
	index   int
	calls   int
}

type netDialer interface {
	DialContext(context.Context, string, M.Socksaddr) (net.Conn, error)
	ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error)
}

func (r *testEntryResolver) ResolveEntry(context.Context) (netDialer, error) {
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
	mu        sync.Mutex
	entries   []netDialer
	landings  []*testDialer
	newCalls  int
}

func (f *testLandingFactory) NewLanding(_ context.Context, entry netDialer) (netDialer, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.newCalls++
	f.entries = append(f.entries, entry)
	landing := &testDialer{}
	f.landings = append(f.landings, landing)
	return landing, nil
}

func TestRuntimeResolvesEntryPerConnection(t *testing.T) {
	entryA := &testDialer{}
	entryB := &testDialer{}
	factory := &testLandingFactory{}
	resolver := &testEntryResolver{entries: []netDialer{entryA, entryB}}
	runtime := NewRuntime(resolver, factory)

	if _, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443}); err != nil {
		t.Fatalf("first chain dial: %v", err)
	}
	if _, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.org", Port: 443}); err != nil {
		t.Fatalf("second chain dial: %v", err)
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
}

func TestRuntimeBlocksUnavailableEntry(t *testing.T) {
	runtime := NewRuntime(&testEntryResolver{}, &testLandingFactory{})
	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrEntryUnavailable) {
		t.Fatalf("expected ErrEntryUnavailable, got %v", err)
	}
}

func TestRuntimeBlocksUnavailableLanding(t *testing.T) {
	entry := &testDialer{}
	resolver := &testEntryResolver{entries: []netDialer{entry}}
	factory := &failingLandingFactory{}
	runtime := NewRuntime(resolver, factory)

	_, err := runtime.DialContext(context.Background(), "tcp", M.Socksaddr{Fqdn: "example.com", Port: 443})
	if !errors.Is(err, ErrLandingUnavailable) {
		t.Fatalf("expected ErrLandingUnavailable, got %v", err)
	}
}

type failingLandingFactory struct{}

func (*failingLandingFactory) NewLanding(context.Context, netDialer) (netDialer, error) {
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

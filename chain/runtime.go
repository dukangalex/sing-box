package chain

import (
	"context"
	"errors"
	"net"
	"sync"

	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

var (
	ErrDisabled           = errors.New("chain is disabled")
	ErrEntryUnavailable   = errors.New("chain entry outbound is unavailable")
	ErrLandingUnavailable = errors.New("chain landing outbound is unavailable")
	ErrUnsupportedNetwork = errors.New("chain network is unsupported")
)

// EntryResolver resolves the outbound that is currently selected by the
// user's existing strategy. Implementations must resolve it at connection
// time so selector and URLTest changes are reflected by new connections.
type EntryResolver interface {
	ResolveEntry(ctx context.Context) (N.Dialer, error)
}

// LandingFactory creates the landing-side dialer for one chain connection.
// The returned dialer must use the supplied entry as its underlying transport.
// This keeps the entry dynamic without mutating the landing outbound itself.
type LandingFactory interface {
	NewLanding(ctx context.Context, entry N.Dialer) (N.Dialer, error)
}

// Runtime is the connection-level Chain runtime. It deliberately does not
// implement adapter.Outbound: Chain is a path capability layered on top of an
// already-selected outbound, not a replacement strategy/outbound.
type Runtime struct {
	entry   EntryResolver
	landing LandingFactory

	mu      sync.RWMutex
	enabled bool
}

func NewRuntime(entry EntryResolver, landing LandingFactory) *Runtime {
	return &Runtime{
		entry:   entry,
		landing: landing,
		enabled: true,
	}
}

func (r *Runtime) SetEnabled(enabled bool) {
	r.mu.Lock()
	r.enabled = enabled
	r.mu.Unlock()
}

func (r *Runtime) Enabled() bool {
	r.mu.RLock()
	enabled := r.enabled
	r.mu.RUnlock()
	return enabled
}

// DialContext creates a fresh chain path for every connection. No entry or
// landing object is permanently rebound, which is required for dynamic
// Selector/URLTest results. Chain is intentionally TCP-only in this phase.
func (r *Runtime) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if !r.Enabled() {
		return nil, ErrDisabled
	}
	if network != "tcp" {
		return nil, ErrUnsupportedNetwork
	}
	if r.entry == nil {
		return nil, ErrEntryUnavailable
	}
	if r.landing == nil {
		return nil, ErrLandingUnavailable
	}

	entry, err := r.entry.ResolveEntry(ctx)
	if err != nil || entry == nil {
		if err != nil {
			return nil, errors.Join(ErrEntryUnavailable, err)
		}
		return nil, ErrEntryUnavailable
	}

	landing, err := r.landing.NewLanding(ctx, entry)
	if err != nil || landing == nil {
		if err != nil {
			return nil, errors.Join(ErrLandingUnavailable, err)
		}
		return nil, ErrLandingUnavailable
	}

	return landing.DialContext(ctx, network, destination)
}

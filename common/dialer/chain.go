package dialer

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// ChainEntryResolver resolves the outbound currently selected by the user's
// existing strategy. Chain deliberately does not implement strategy selection.
type ChainEntryResolver interface {
	ResolveChainEntry(ctx context.Context) (adapter.Outbound, error)
}

// ChainDialer is a strict dynamic detour used by a Chain landing outbound.
// The resolver is evaluated for every new connection so Selector/URLTest
// changes are observed without changing their native semantics.
type ChainDialer struct {
	resolver ChainEntryResolver
	landing  M.Socksaddr
}

func NewChain(resolver ChainEntryResolver, landing M.Socksaddr) N.Dialer {
	return &ChainDialer{resolver: resolver, landing: landing}
}

func (d *ChainDialer) DialContext(ctx context.Context, network string, _ M.Socksaddr) (net.Conn, error) {
	if d.resolver == nil {
		return nil, E.New("chain: entry resolver is nil")
	}
	entry, err := d.resolver.ResolveChainEntry(ctx)
	if err != nil {
		return nil, E.Cause(err, "chain: resolve entry")
	}
	if entry == nil {
		return nil, E.New("chain: resolved entry outbound is nil")
	}
	if entry.Tag() == "" {
		return nil, E.New("chain: resolved entry outbound has empty tag")
	}
	return entry.DialContext(ctx, network, d.landing)
}

func (d *ChainDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, E.New("chain: packet dialing is not implemented")
}

func (d *ChainDialer) Upstream() any {
	if d.resolver == nil {
		return nil
	}
	entry, err := d.resolver.ResolveChainEntry(context.Background())
	if err != nil {
		return nil
	}
	return entry
}

package dialer

import (
	"context"
	"net"
	"sync"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// ChainEntryResolver resolves the outbound selected by the user's existing strategy.
// Chain does not implement strategy selection itself.
type ChainEntryResolver interface {
	ResolveChainEntry(ctx context.Context) (adapter.Outbound, error)
}

// ChainDialer is a strict dynamic detour used by a Chain landing outbound.
// The resolved Entry is used only to reach the Landing; it is never used as a
// direct fallback to the final destination.
type ChainDialer struct {
	resolver ChainEntryResolver
	landing  M.Socksaddr

	initOnce sync.Once
	entry    adapter.Outbound
	initErr  error
}

func NewChain(resolver ChainEntryResolver, landing M.Socksaddr) N.Dialer {
	return &ChainDialer{resolver: resolver, landing: landing}
}

func (d *ChainDialer) Dialer() (adapter.Outbound, error) {
	d.initOnce.Do(func() {
		if d.resolver == nil {
			d.initErr = E.New("chain entry resolver is nil")
			return
		}
		d.entry, d.initErr = d.resolver.ResolveChainEntry(context.Background())
		if d.initErr != nil {
			return
		}
		if d.entry == nil {
			d.initErr = E.New("chain entry outbound is nil")
		}
	})
	return d.entry, d.initErr
}

func (d *ChainDialer) DialContext(ctx context.Context, network string, _ M.Socksaddr) (net.Conn, error) {
	entry, err := d.Dialer()
	if err != nil {
		return nil, err
	}
	return entry.DialContext(ctx, network, d.landing)
}

func (d *ChainDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, E.New("chain: packet dialing is not implemented")
}

func (d *ChainDialer) Upstream() any {
	entry, _ := d.Dialer()
	return common.PtrOrNil(entry)
}

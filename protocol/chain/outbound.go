package chain

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.ChainOutboundOptions](registry, C.TypeChain, NewOutbound)
}

var _ adapter.Outbound = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	manager adapter.OutboundManager
	entry   string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ChainOutboundOptions) (adapter.Outbound, error) {
	if len(options.Outbounds) == 0 {
		return nil, E.New("chain outbound requires at least one outbound")
	}
	if options.EntryOutbound == "" {
		return nil, E.New("chain outbound [", tag, "] was not compiled (internal error)")
	}
	manager := service.FromContext[adapter.OutboundManager](ctx)
	if manager == nil {
		return nil, E.New("missing outbound manager in context")
	}
	if _, loaded := manager.Outbound(options.EntryOutbound); !loaded {
		return nil, E.New("chain entry outbound not found: ", options.EntryOutbound)
	}
	// Dependencies point only to the compiled entry hop.
	// The full hop list is already validated and wired at compile time.
	return &Outbound{
		Adapter: outbound.NewAdapter(C.TypeChain, tag, nil, []string{options.EntryOutbound}),
		manager: manager,
		entry:   options.EntryOutbound,
	}, nil
}

func (o *Outbound) entryOutbound() (adapter.Outbound, error) {
	entry, loaded := o.manager.Outbound(o.entry)
	if !loaded || entry == nil {
		return nil, E.New("chain entry outbound unavailable: ", o.entry)
	}
	return entry, nil
}

func (o *Outbound) Network() []string {
	entry, err := o.entryOutbound()
	if err != nil {
		return nil
	}
	return entry.Network()
}

func (o *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	entry, err := o.entryOutbound()
	if err != nil {
		return nil, err
	}
	return entry.DialContext(ctx, network, destination)
}

func (o *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	entry, err := o.entryOutbound()
	if err != nil {
		return nil, err
	}
	return entry.ListenPacket(ctx, destination)
}

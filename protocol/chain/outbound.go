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
	final   string
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ChainOutboundOptions) (adapter.Outbound, error) {
	if len(options.Outbounds) < 2 {
		return nil, E.New("chain outbound requires at least 2 outbounds")
	}
	if options.FinalOutbound == "" {
		return nil, E.New("chain outbound was not compiled")
	}
	manager := service.FromContext[adapter.OutboundManager](ctx)
	if manager == nil {
		return nil, E.New("missing outbound manager in context")
	}
	if finalOutbound, loaded := manager.Outbound(options.FinalOutbound); !loaded || finalOutbound == nil {
		return nil, E.New("chain final outbound not found: ", options.FinalOutbound)
	}
	return &Outbound{
		Adapter: outbound.NewAdapter(C.TypeChain, tag, nil, []string{options.FinalOutbound}),
		manager: manager,
		final:   options.FinalOutbound,
	}, nil
}

func (o *Outbound) finalOutbound() (adapter.Outbound, error) {
	finalOutbound, loaded := o.manager.Outbound(o.final)
	if !loaded || finalOutbound == nil {
		return nil, E.New("chain final outbound unavailable: ", o.final)
	}
	return finalOutbound, nil
}

func (o *Outbound) Network() []string {
	finalOutbound, err := o.finalOutbound()
	if err != nil {
		return nil
	}
	return finalOutbound.Network()
}

func (o *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	finalOutbound, err := o.finalOutbound()
	if err != nil {
		return nil, err
	}
	return finalOutbound.DialContext(ctx, network, destination)
}

func (o *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	finalOutbound, err := o.finalOutbound()
	if err != nil {
		return nil, err
	}
	return finalOutbound.ListenPacket(ctx, destination)
}

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
	if len(options.Outbounds) == 0 {
		return nil, E.New("chain outbound requires at least 1 outbound")
	}
	manager := service.FromContext[adapter.OutboundManager](ctx)
	if manager == nil {
		return nil, E.New("missing outbound manager in context")
	}
	final := options.Outbounds[len(options.Outbounds)-1]
	derivedFinal := tag + ":chain:" + indexString(len(options.Outbounds)-1)
	if final == "" {
		return nil, E.New("chain final outbound is empty")
	}
	if _, loaded := manager.Outbound(derivedFinal); loaded {
		final = derivedFinal
	} else if _, loaded := manager.Outbound(final); !loaded {
		return nil, E.New("chain final outbound not found: ", final)
	}
	return &Outbound{
		Adapter: outbound.NewAdapter(C.TypeChain, tag, nil, []string{final}),
		manager: manager,
		final:   final,
	}, nil
}

func indexString(index int) string {
	if index < 10 {
		return string(rune('0' + index))
	}
	return indexString(index/10) + string(rune('0' + index%10))
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

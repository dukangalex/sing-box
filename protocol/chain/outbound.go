package chain

import (
	"context"
	"net"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	chainruntime "github.com/sagernet/sing-box/chain"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	singSocks "github.com/sagernet/sing/protocol/socks"
	E "github.com/sagernet/sing/common/exceptions"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
	"github.com/sagernet/sing/service"
)

func RegisterOutbound(registry *outbound.Registry) {
	outbound.Register[option.ChainOutboundOptions](registry, C.TypeChain, NewOutbound)
}

var _ adapter.Outbound = (*Outbound)(nil)

type Outbound struct {
	outbound.Adapter
	ctx     context.Context
	manager adapter.OutboundManager
	runtime *chainruntime.Runtime
}

type entryResolver struct {
	manager adapter.OutboundManager
	tag     string
}

func (r *entryResolver) ResolveEntry(ctx context.Context) (N.Dialer, error) {
	if r.manager == nil {
		return nil, chainruntime.ErrEntryUnavailable
	}
	entry, loaded := r.manager.Outbound(r.tag)
	if !loaded || entry == nil {
		return nil, E.New("chain entry outbound not found: ", r.tag)
	}
	return entry, nil
}

type landingFactory struct {
	version  string
	server   M.Socksaddr
	username string
	password string
}

func (f *landingFactory) NewLanding(ctx context.Context, entry N.Dialer) (N.Dialer, error) {
	if entry == nil {
		return nil, E.New("chain entry dialer is nil")
	}

	version := singSocks.Version5
	if f.version != "" {
		parsed, err := singSocks.ParseVersion(f.version)
		if err != nil {
			return nil, err
		}
		version = parsed
	}

	return singSocks.NewClient(entry, f.server, version, f.username, f.password), nil
}

func NewOutbound(ctx context.Context, router adapter.Router, logger log.ContextLogger, tag string, options option.ChainOutboundOptions) (adapter.Outbound, error) {
	if options.Entry == "" {
		return nil, E.New("missing chain entry")
	}
	if options.Server == "" || options.ServerPort == 0 {
		return nil, E.New("missing chain landing server")
	}

	manager := service.FromContext[adapter.OutboundManager](ctx)
	if manager == nil {
		return nil, E.New("missing outbound manager in context")
	}

	entry, loaded := manager.Outbound(options.Entry)
	if !loaded || entry == nil {
		return nil, E.New("chain entry outbound not found: ", options.Entry)
	}

	if !entry.NetworkSupports(N.NetworkTCP) {
		return nil, E.New("chain entry outbound does not support TCP: ", options.Entry)
	}

	resolver := &entryResolver{
		manager: manager,
		tag:     options.Entry,
	}

	landing := &landingFactory{
		version:  options.Version,
		server:   options.ServerOptions.Build(),
		username: options.Username,
		password: options.Password,
	}

	runtime := chainruntime.NewRuntime(resolver, landing)

	return &Outbound{
		Adapter: outbound.NewAdapter(C.TypeChain, tag, []string{N.NetworkTCP}, []string{options.Entry}),
		ctx:     ctx,
		manager: manager,
		runtime: runtime,
	}, nil
}

func (o *Outbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	if N.NetworkName(network) != N.NetworkTCP {
		return nil, E.Extend(N.ErrUnknownNetwork, network)
	}
	return o.runtime.DialContext(ctx, network, destination)
}

func (o *Outbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	return nil, E.New("chain: UDP is not implemented")
}

package chain

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	M "github.com/sagernet/sing/common/metadata"
	"github.com/sagernet/sing/service"
)

type chainRuntimeTestManager struct {
	outbound adapter.Outbound
}

func (m *chainRuntimeTestManager) Start(adapter.StartStage) error { return nil }
func (m *chainRuntimeTestManager) Close() error                   { return nil }
func (m *chainRuntimeTestManager) Outbounds() []adapter.Outbound {
	if m.outbound == nil {
		return nil
	}
	return []adapter.Outbound{m.outbound}
}
func (m *chainRuntimeTestManager) Outbound(tag string) (adapter.Outbound, bool) {
	if m.outbound != nil && m.outbound.Tag() == tag {
		return m.outbound, true
	}
	return nil, false
}
func (m *chainRuntimeTestManager) Default() adapter.Outbound { return m.outbound }
func (m *chainRuntimeTestManager) Remove(string) error        { return nil }
func (m *chainRuntimeTestManager) Create(context.Context, adapter.Router, log.ContextLogger, string, string, any) error {
	return nil
}

type chainRuntimeTestOutbound struct {
	tag          string
	network      []string
	dial         func(context.Context, string, M.Socksaddr) (net.Conn, error)
	packet       func(context.Context, M.Socksaddr) (net.PacketConn, error)
	dependencies []string
}

func (o *chainRuntimeTestOutbound) Type() string        { return "chain-test" }
func (o *chainRuntimeTestOutbound) Tag() string         { return o.tag }
func (o *chainRuntimeTestOutbound) Network() []string   { return o.network }
func (o *chainRuntimeTestOutbound) Dependencies() []string { return o.dependencies }
func (o *chainRuntimeTestOutbound) DialContext(ctx context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	return o.dial(ctx, network, destination)
}
func (o *chainRuntimeTestOutbound) ListenPacket(ctx context.Context, destination M.Socksaddr) (net.PacketConn, error) {
	if o.packet == nil {
		return nil, errors.New("packet dialing not configured")
	}
	return o.packet(ctx, destination)
}

func TestChainOutboundDialContextForwardsToFinalOutbound(t *testing.T) {
	var called bool
	final := &chainRuntimeTestOutbound{
		tag:     "final",
		network: []string{"tcp"},
		dial: func(context.Context, string, M.Socksaddr) (net.Conn, error) {
			called = true
			return net.Pipe()
		},
	}
	manager := &chainRuntimeTestManager{outbound: final}
	ctx := service.ContextWith[adapter.OutboundManager](context.Background(), manager)

	chain, err := NewOutbound(ctx, nil, nil, "chain", option.ChainOutboundOptions{
		Outbounds:     []string{"a"},
		FinalOutbound: "final",
	})
	if err != nil {
		t.Fatal(err)
	}

	conn, err := chain.DialContext(context.Background(), "tcp", M.Socksaddr{})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("chain did not forward DialContext to final outbound")
	}
	_ = conn.Close()
}

func TestChainOutboundFailsClosedWhenFinalOutboundDisappears(t *testing.T) {
	final := &chainRuntimeTestOutbound{
		tag:     "final",
		network: []string{"tcp"},
		dial: func(context.Context, string, M.Socksaddr) (net.Conn, error) {
			return net.Pipe()
		},
	}
	manager := &chainRuntimeTestManager{outbound: final}
	ctx := service.ContextWith[adapter.OutboundManager](context.Background(), manager)

	chain, err := NewOutbound(ctx, nil, nil, "chain", option.ChainOutboundOptions{
		Outbounds:     []string{"a"},
		FinalOutbound: "final",
	})
	if err != nil {
		t.Fatal(err)
	}

	manager.outbound = nil
	if _, err = chain.DialContext(context.Background(), "tcp", M.Socksaddr{}); err == nil {
		t.Fatal("expected DialContext to fail when final outbound is unavailable")
	}
}

func TestChainOutboundRejectsMissingFinalOutbound(t *testing.T) {
	manager := &chainRuntimeTestManager{}
	ctx := service.ContextWith[adapter.OutboundManager](context.Background(), manager)

	_, err := NewOutbound(ctx, nil, nil, "chain", option.ChainOutboundOptions{
		Outbounds:     []string{"a"},
		FinalOutbound: "missing",
	})
	if err == nil {
		t.Fatal("expected construction to fail when final outbound is unavailable")
	}
}

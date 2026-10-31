package group

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/protocol/chain"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type chainPacketGuard struct{ chain.Outbound }

func (*chainPacketGuard) Network() []string { return []string{N.NetworkTCP, N.NetworkUDP} }

type fallbackPacketCounter struct {
	stubOutbound
	calls int
}

func (o *fallbackPacketCounter) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	o.calls++
	return nil, nil
}

func TestURLTestDoesNotFallbackAfterChainUDPRefusal(t *testing.T) {
	selected := &chainPacketGuard{chain.Outbound{Adapter: outbound.NewAdapter(C.TypeChain, "chosen-chain", nil, nil)}}
	fallback := &fallbackPacketCounter{stubOutbound: stubOutbound{tag: "different-exit", nets: []string{N.NetworkUDP}}}
	g := &URLTestGroup{selectedOutboundUDP: selected, outbounds: []adapter.Outbound{selected, fallback}}
	// This regression isolates connection fallback, not background URL probes.
	g.checking.Store(true)
	s := &URLTest{ctx: context.Background(), group: g, logger: log.NewNOPFactory().Logger()}
	conn, err := s.listenFirstAvailable(context.Background(), M.Socksaddr{})
	if conn != nil {
		conn.Close()
		t.Error("unexpected packet connection")
	}
	if !errors.Is(err, adapter.ErrOutboundPolicyDenied) {
		t.Error("chain UDP denial must reach caller")
	}
	if g.selectedOutboundUDP != selected {
		t.Error("policy denial must not forget the selected chain")
	}
	if fallback.calls != 0 {
		t.Fatalf("UDP escaped the selected chain via another exit (%d fallback calls)", fallback.calls)
	}
}

func (o *fallbackPacketCounter) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	o.calls++
	return nil, nil
}

func TestURLTestDialDoesNotFallbackAfterChainUDPRefusal(t *testing.T) {
	for _, network := range []string{"udp", "udp4", "udp6"} {
		t.Run(network, func(t *testing.T) {
			selected := &chainPacketGuard{chain.Outbound{Adapter: outbound.NewAdapter(C.TypeChain, "chosen-chain", nil, nil)}}
			fallback := &fallbackPacketCounter{stubOutbound: stubOutbound{tag: "different-exit", nets: []string{N.NetworkUDP}}}
			g := &URLTestGroup{selectedOutboundUDP: selected, outbounds: []adapter.Outbound{selected, fallback}}
			g.checking.Store(true)
			s := &URLTest{ctx: context.Background(), group: g, logger: log.NewNOPFactory().Logger()}
			_, err := s.dialFirstAvailable(context.Background(), network, N.NetworkUDP, M.Socksaddr{})
			if !errors.Is(err, adapter.ErrOutboundPolicyDenied) {
				t.Errorf("policy denial missing: %v", err)
			}
			if g.selectedOutboundUDP != selected {
				t.Error("policy denial must not forget the selected chain")
			}
			if fallback.calls != 0 {
				t.Errorf("UDP escaped chain via %d fallback calls", fallback.calls)
			}
		})
	}
}

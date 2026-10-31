package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/adapter/outbound"
	C "github.com/sagernet/sing-box/constant"
	M "github.com/sagernet/sing/common/metadata"
)

// UDP does not follow the chain: the landing hop's ListenPacket dials with the
// default dialer, so the packets would not leave from the landing node. The
// chain outbound must refuse rather than report a wrong exit address.
func TestChainOutboundRefusesUDP(t *testing.T) {
	o := &Outbound{
		Adapter: outbound.NewAdapter(C.TypeChain, "chain-test", nil, nil),
	}
	conn, err := o.ListenPacket(context.Background(), M.Socksaddr{})
	if !errors.Is(err, adapter.ErrOutboundPolicyDenied) {
		if conn != nil {
			conn.Close()
		}
		t.Fatal("chain outbound must not create a packet conn")
	}
	if conn != nil {
		conn.Close()
		t.Fatal("chain outbound must not return a packet conn alongside an error")
	}
}

func TestChainOutboundRefusesUDPDial(t *testing.T) {
	for _, network := range []string{"udp", "udp4", "udp6"} {
		t.Run(network, func(t *testing.T) {
			o := &Outbound{Adapter: outbound.NewAdapter(C.TypeChain, "chain-test", nil, nil)}
			conn, err := o.DialContext(context.Background(), network, M.Socksaddr{})
			if conn != nil {
				conn.Close()
				t.Error("chain must not return a UDP dial connection")
			}
			if !errors.Is(err, adapter.ErrOutboundPolicyDenied) {
				t.Fatalf("missing UDP policy refusal: %v", err)
			}
		})
	}
}

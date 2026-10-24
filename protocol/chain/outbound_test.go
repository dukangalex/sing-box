package chain

import (
	"context"
	"testing"

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
	if err == nil {
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

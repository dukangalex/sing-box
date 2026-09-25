package group

import (
	"context"
	"net"
	"testing"

	"github.com/sagernet/sing-box/adapter"
	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

type stubOutbound struct {
	tag  string
	nets []string
}

func (s stubOutbound) Type() string           { return "stub" }
func (s stubOutbound) Tag() string            { return s.tag }
func (s stubOutbound) Network() []string      { return s.nets }
func (s stubOutbound) Dependencies() []string { return nil }
func (s stubOutbound) DialContext(context.Context, string, M.Socksaddr) (net.Conn, error) {
	return nil, nil
}
func (s stubOutbound) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, nil
}

func TestOrderURLTestDialCandidatesPrefersStickyThenDelay(t *testing.T) {
	dead := stubOutbound{tag: "dead-cf", nets: []string{N.NetworkTCP}}
	fast := stubOutbound{tag: "fast", nets: []string{N.NetworkTCP}}
	slow := stubOutbound{tag: "slow", nets: []string{N.NetworkTCP}}
	fresh := stubOutbound{tag: "fresh", nets: []string{N.NetworkTCP}}
	udpOnly := stubOutbound{tag: "udp", nets: []string{N.NetworkUDP}}
	delays := map[string]uint16{"dead-cf": 20, "fast": 40, "slow": 90}
	got := orderURLTestDialCandidates(dead, []adapter.Outbound{udpOnly, fresh, slow, fast, dead}, N.NetworkTCP, func(o adapter.Outbound) (uint16, bool) {
		delay, ok := delays[o.Tag()]
		return delay, ok
	})
	want := []string{"dead-cf", "fast", "slow", "fresh"}
	if len(got) != len(want) {
		t.Fatalf("len %d, want %d (%v)", len(got), len(want), tagsOf(got))
	}
	for i, tag := range want {
		if got[i].Tag() != tag {
			t.Fatalf("order %v, want %v", tagsOf(got), want)
		}
	}
}

func TestOrderURLTestDialCandidatesSkipsEmptySelection(t *testing.T) {
	a := stubOutbound{tag: "a", nets: []string{N.NetworkTCP}}
	b := stubOutbound{tag: "b", nets: []string{N.NetworkTCP}}
	got := orderURLTestDialCandidates(nil, []adapter.Outbound{b, a}, N.NetworkTCP, func(adapter.Outbound) (uint16, bool) {
		return 0, false
	})
	if len(got) != 2 || got[0].Tag() != "b" || got[1].Tag() != "a" {
		t.Fatalf("untagged order %v", tagsOf(got))
	}
}

func tagsOf(outbounds []adapter.Outbound) []string {
	tags := make([]string, len(outbounds))
	for i, outbound := range outbounds {
		tags[i] = outbound.Tag()
	}
	return tags
}

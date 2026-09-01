package option

import (
	"context"
	"testing"

	C "github.com/sagernet/sing-box/constant"
)

type chainTestHopOptions struct {
	DialerOptions
	Name string `json:"name,omitempty"`
}

func TestCompileChainOutbounds(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "a", Options: &chainTestHopOptions{Name: "a"}},
		{Type: "test-hop", Tag: "b", Options: &chainTestHopOptions{Name: "b"}},
		{Type: "test-hop", Tag: "c", Options: &chainTestHopOptions{Name: "c"}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b", "c"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 6 {
		t.Fatalf("expected 6 outbounds, got %d", len(compiled))
	}

	wantTags := []string{"a", "b", "c", "chain:chain:0", "chain:chain:1", "chain"}
	for i, want := range wantTags {
		if compiled[i].Tag != want {
			t.Fatalf("outbound[%d]: expected %q, got %q", i, want, compiled[i].Tag)
		}
	}

	a := compiled[3].Options.(*chainTestHopOptions)
	b := compiled[4].Options.(*chainTestHopOptions)
	if a.Detour != "chain:chain:1" {
		t.Fatalf("first hop detour = %q", a.Detour)
	}
	if b.Detour != "c" {
		t.Fatalf("second hop detour = %q", b.Detour)
	}

	chain := compiled[5].Options.(*ChainOutboundOptions)
	if chain.EntryOutbound != "chain:chain:0" {
		t.Fatalf("chain entry = %q", chain.EntryOutbound)
	}
}

func TestCompileChainOutboundsSingleHop(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "a", Options: &chainTestHopOptions{Name: "a"}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"a"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 2 {
		t.Fatalf("expected 2 outbounds, got %d", len(compiled))
	}
	chain := compiled[1].Options.(*ChainOutboundOptions)
	if chain.EntryOutbound != "a" {
		t.Fatalf("chain entry = %q", chain.EntryOutbound)
	}
}

func TestCompileChainOutboundsRejectsNestedAndCycles(t *testing.T) {
	outbounds := []Outbound{
		{Type: C.TypeChain, Tag: "inner", Options: &ChainOutboundOptions{Outbounds: []string{"a"}}},
		{Type: C.TypeChain, Tag: "outer", Options: &ChainOutboundOptions{Outbounds: []string{"inner", "a"}}},
		{Type: "test-hop", Tag: "a", Options: &chainTestHopOptions{Name: "a"}},
	}
	if _, err := CompileChainOutbounds(context.Background(), outbounds); err == nil {
		t.Fatal("expected nested chain rejection")
	}
}

func TestCompileChainOutboundsRejectsExistingDetour(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "a", Options: &chainTestHopOptions{DialerOptions: DialerOptions{Detour: "existing"}}},
		{Type: "test-hop", Tag: "b", Options: &chainTestHopOptions{Name: "b"}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b"}}},
	}
	if _, err := CompileChainOutbounds(context.Background(), outbounds); err == nil {
		t.Fatal("expected existing detour rejection")
	}
}

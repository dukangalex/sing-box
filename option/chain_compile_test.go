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
	if a.Detour != "a" {
		t.Fatalf("second hop (b clone) detour = %q, want first hop a", a.Detour)
	}
	if b.Detour != "chain:chain:0" {
		t.Fatalf("exit hop (c clone) detour = %q, want b clone", b.Detour)
	}

	chain := compiled[5].Options.(*ChainOutboundOptions)
	if chain.EntryOutbound != "chain:chain:1" {
		t.Fatalf("chain entry = %q, want last-hop clone", chain.EntryOutbound)
	}
}

func TestCompileChainOutboundsTwoHopEntryLanding(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: "test-hop", Tag: "landing", Options: &chainTestHopOptions{Name: "landing"}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "landing"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 4 {
		t.Fatalf("expected 4 outbounds, got %d", len(compiled))
	}
	var landingClone *chainTestHopOptions
	var entryTag string
	for _, o := range compiled {
		if o.Tag == "chain:chain:0" {
			landingClone = o.Options.(*chainTestHopOptions)
		}
		if o.Tag == "chain" {
			entryTag = o.Options.(*ChainOutboundOptions).EntryOutbound
		}
	}
	if landingClone == nil {
		t.Fatal("missing landing clone")
	}
	if landingClone.Detour != "entry" {
		t.Fatalf("landing detour = %q, want entry (packet path client→entry→landing)", landingClone.Detour)
	}
	if entryTag != "chain:chain:0" {
		t.Fatalf("chain dials %q, want landing clone so public IP is landing", entryTag)
	}
}

func TestCompileChainOutboundsURLTestFront(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: "test-hop", Tag: "n2", Options: &chainTestHopOptions{Name: "n2"}},
		{Type: "test-hop", Tag: "exit", Options: &chainTestHopOptions{Name: "exit"}},
		{Type: C.TypeURLTest, Tag: "auto", Options: &URLTestOutboundOptions{Outbounds: []string{"n1", "n2"}}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"auto", "exit"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	// originals: n1,n2,exit,auto + cloned exit + chain.
	// First hop (auto urltest) is used as-is as the detour of the exit clone.
	if len(compiled) != 6 {
		t.Fatalf("expected 6 outbounds, got %d", len(compiled))
	}

	var foundExitClone bool
	for _, o := range compiled {
		if o.Tag == "chain:chain:0" {
			h := o.Options.(*chainTestHopOptions)
			if h.Detour != "auto" {
				t.Fatalf("exit clone detour = %q, want auto (entry group)", h.Detour)
			}
			foundExitClone = true
		}
		if o.Tag == "chain" {
			c := o.Options.(*ChainOutboundOptions)
			if c.EntryOutbound != "chain:chain:0" {
				t.Fatalf("entry = %q, want exit clone", c.EntryOutbound)
			}
		}
	}
	if !foundExitClone {
		t.Fatal("missing cloned exit hop")
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

func TestCompileChainOutboundsRejectsSyntheticTagCollision(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "a", Options: &chainTestHopOptions{Name: "a"}},
		{Type: "test-hop", Tag: "chain:chain:0", Options: &chainTestHopOptions{Name: "collision"}},
		{Type: "test-hop", Tag: "b", Options: &chainTestHopOptions{Name: "b"}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b"}}},
	}
	if _, err := CompileChainOutbounds(context.Background(), outbounds); err == nil {
		t.Fatal("expected synthetic tag collision rejection")
	}
}

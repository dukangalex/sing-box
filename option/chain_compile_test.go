package option

import (
	"context"
	"strings"
	"testing"
	"time"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/common/json/badoption"
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

func TestCompileChainOutboundsSelectorFrontPreserved(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: "test-hop", Tag: "n2", Options: &chainTestHopOptions{Name: "n2"}},
		{Type: "test-hop", Tag: "exit", Options: &chainTestHopOptions{Name: "exit"}},
		{Type: C.TypeSelector, Tag: "pick", Options: &SelectorOutboundOptions{
			Outbounds:                 []string{"n1", "n2"},
			Default:                   "n2",
			InterruptExistConnections: true,
		}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"pick", "exit"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	// originals: n1,n2,exit,pick + cloned exit + chain.
	// The front (index 0) selector is used as-is: no expansion, no field loss.
	if len(compiled) != 6 {
		t.Fatalf("expected 6 outbounds, got %d", len(compiled))
	}
	var found bool
	for _, o := range compiled {
		if o.Tag == "pick" {
			sel := o.Options.(*SelectorOutboundOptions)
			if sel.Default != "n2" {
				t.Fatalf("front selector default = %q, want n2 (untouched)", sel.Default)
			}
			if !sel.InterruptExistConnections {
				t.Fatal("front selector interrupt_exist_connections lost")
			}
			if len(sel.Outbounds) != 2 || sel.Outbounds[0] != "n1" || sel.Outbounds[1] != "n2" {
				t.Fatalf("front selector members = %v, want [n1 n2]", sel.Outbounds)
			}
			found = true
		}
		if o.Tag == "chain:chain:0" {
			h := o.Options.(*chainTestHopOptions)
			if h.Detour != "pick" {
				t.Fatalf("exit clone detour = %q, want pick (entry group)", h.Detour)
			}
		}
	}
	if !found {
		t.Fatal("missing front selector")
	}
}

func TestCompileChainOutboundsSelectorLandingDefaultRemapped(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: "test-hop", Tag: "mid", Options: &chainTestHopOptions{Name: "mid"}},
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: "test-hop", Tag: "n2", Options: &chainTestHopOptions{Name: "n2"}},
		{Type: C.TypeSelector, Tag: "pick", Options: &SelectorOutboundOptions{
			Outbounds:                 []string{"n1", "n2"},
			Default:                   "n2",
			InterruptExistConnections: true,
		}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "mid", "pick"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	// originals: entry,mid,n1,n2,pick + mid clone + n1 clone + n2 clone +
	// synthetic selector + chain.
	if len(compiled) != 10 {
		t.Fatalf("expected 10 outbounds, got %d", len(compiled))
	}
	var found bool
	for _, o := range compiled {
		if o.Tag == "chain:chain:1" {
			if o.Type != C.TypeSelector {
				t.Fatalf("synthetic landing type = %q, want selector", o.Type)
			}
			sel := o.Options.(*SelectorOutboundOptions)
			if sel.Default != "chain:chain:1:n2" {
				t.Fatalf("synthetic default = %q, want chain:chain:1:n2 (remapped through member-tag mapping)", sel.Default)
			}
			if !sel.InterruptExistConnections {
				t.Fatal("synthetic selector interrupt_exist_connections not copied")
			}
			if len(sel.Outbounds) != 2 || sel.Outbounds[0] != "chain:chain:1:n1" || sel.Outbounds[1] != "chain:chain:1:n2" {
				t.Fatalf("synthetic members = %v, want [chain:chain:1:n1 chain:chain:1:n2]", sel.Outbounds)
			}
			found = true
		}
		if o.Tag == "chain:chain:1:n2" {
			h := o.Options.(*chainTestHopOptions)
			if h.Detour != "chain:chain:0" {
				t.Fatalf("landing leaf detour = %q, want mid clone chain:chain:0", h.Detour)
			}
		}
	}
	if !found {
		t.Fatal("missing synthetic landing selector")
	}
}

func TestCompileChainOutboundsSelectorLandingDefaultFilteredMemberDropped(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: C.TypeDirect, Tag: "d", Options: &chainTestHopOptions{Name: "d"}},
		{Type: C.TypeSelector, Tag: "pick", Options: &SelectorOutboundOptions{
			Outbounds: []string{"n1", "d"},
			Default:   "d",
		}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "pick"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	// Default pointed at a direct member, which flattening filters out: the
	// synthetic selector must compile with an empty Default, never with the
	// original tag "d".
	var found bool
	for _, o := range compiled {
		if o.Tag == "chain:chain:0" {
			sel := o.Options.(*SelectorOutboundOptions)
			if sel.Default != "" {
				t.Fatalf("synthetic default = %q, want empty (default member was filtered out)", sel.Default)
			}
			if len(sel.Outbounds) != 1 || sel.Outbounds[0] != "chain:chain:0:n1" {
				t.Fatalf("synthetic members = %v, want [chain:chain:0:n1]", sel.Outbounds)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing synthetic landing selector")
	}
}

func TestCompileChainOutboundsSelectorLandingDefaultNestedGroupDropped(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: "test-hop", Tag: "n2", Options: &chainTestHopOptions{Name: "n2"}},
		{Type: "test-hop", Tag: "n3", Options: &chainTestHopOptions{Name: "n3"}},
		{Type: C.TypeSelector, Tag: "inner", Options: &SelectorOutboundOptions{Outbounds: []string{"n1", "n2"}}},
		{Type: C.TypeSelector, Tag: "pick", Options: &SelectorOutboundOptions{
			Outbounds: []string{"inner", "n3"},
			Default:   "inner",
		}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "pick"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	// Default pointed at the nested group itself: the group is flattened
	// away, so the synthetic selector gets its leaves but an empty Default.
	var found bool
	for _, o := range compiled {
		if o.Tag == "chain:chain:0" {
			sel := o.Options.(*SelectorOutboundOptions)
			if sel.Default != "" {
				t.Fatalf("synthetic default = %q, want empty (default was a nested group)", sel.Default)
			}
			want := []string{"chain:chain:0:n1", "chain:chain:0:n2", "chain:chain:0:n3"}
			if len(sel.Outbounds) != len(want) {
				t.Fatalf("synthetic members = %v, want %v", sel.Outbounds, want)
			}
			for i := range want {
				if sel.Outbounds[i] != want[i] {
					t.Fatalf("synthetic members = %v, want %v", sel.Outbounds, want)
				}
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing synthetic landing selector")
	}
}

func TestCompileChainOutboundsURLTestLandingInterruptCopied(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: "test-hop", Tag: "n1", Options: &chainTestHopOptions{Name: "n1"}},
		{Type: "test-hop", Tag: "n2", Options: &chainTestHopOptions{Name: "n2"}},
		{Type: C.TypeURLTest, Tag: "auto", Options: &URLTestOutboundOptions{
			Outbounds:                 []string{"n1", "n2"},
			URL:                       "https://www.gstatic.com/generate_204",
			Interval:                  badoption.Duration(3 * time.Minute),
			Tolerance:                 42,
			IdleTimeout:               badoption.Duration(10 * time.Minute),
			InterruptExistConnections: true,
		}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "auto"}}},
	}
	compiled, err := CompileChainOutbounds(context.Background(), outbounds)
	if err != nil {
		t.Fatal(err)
	}

	var found bool
	for _, o := range compiled {
		if o.Tag == "chain:chain:0" {
			if o.Type != C.TypeURLTest {
				t.Fatalf("synthetic landing type = %q, want urltest", o.Type)
			}
			ut := o.Options.(*URLTestOutboundOptions)
			if !ut.InterruptExistConnections {
				t.Fatal("synthetic urltest interrupt_exist_connections not copied")
			}
			if ut.URL != "https://www.gstatic.com/generate_204" {
				t.Fatalf("synthetic urltest URL = %q, want original", ut.URL)
			}
			if ut.Tolerance != 42 {
				t.Fatalf("synthetic urltest tolerance = %d, want 42", ut.Tolerance)
			}
			if ut.Interval != badoption.Duration(3*time.Minute) {
				t.Fatalf("synthetic urltest interval = %v, want 3m", ut.Interval)
			}
			if ut.IdleTimeout != badoption.Duration(10*time.Minute) {
				t.Fatalf("synthetic urltest idle_timeout = %v, want 10m", ut.IdleTimeout)
			}
			if len(ut.Outbounds) != 2 || ut.Outbounds[0] != "chain:chain:0:n1" || ut.Outbounds[1] != "chain:chain:0:n2" {
				t.Fatalf("synthetic urltest members = %v, want [chain:chain:0:n1 chain:chain:0:n2]", ut.Outbounds)
			}
			found = true
		}
	}
	if !found {
		t.Fatal("missing synthetic landing urltest")
	}
}

func TestCompileChainOutboundsGroupCycleNoUsableMembers(t *testing.T) {
	outbounds := []Outbound{
		{Type: "test-hop", Tag: "entry", Options: &chainTestHopOptions{Name: "entry"}},
		{Type: C.TypeSelector, Tag: "g1", Options: &SelectorOutboundOptions{Outbounds: []string{"g2"}}},
		{Type: C.TypeSelector, Tag: "g2", Options: &SelectorOutboundOptions{Outbounds: []string{"g1"}}},
		{Type: C.TypeChain, Tag: "chain", Options: &ChainOutboundOptions{Outbounds: []string{"entry", "g1"}}},
	}
	// A membership cycle yields no leaves: flattening breaks the cycle
	// quietly, then compilation fails closed with the no-usable-members error.
	_, err := CompileChainOutbounds(context.Background(), outbounds)
	if err == nil {
		t.Fatal("expected compile error for cyclic group with no usable members")
	}
	if !strings.Contains(err.Error(), "no usable proxy members") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package option

import (
	"context"
	"strings"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/service"
)

type chainTestOutboundOptions struct {
	DialerOptions
	Name string `json:"name,omitempty"`
}

type chainTestOutboundRegistry struct{}

func (chainTestOutboundRegistry) OptionTypes() []string {
	return []string{"chain-test-hop"}
}

func (chainTestOutboundRegistry) CreateOptions(outboundType string) (any, bool) {
	if outboundType != "chain-test-hop" {
		return nil, false
	}
	return new(chainTestOutboundOptions), true
}

func chainTestContext() context.Context {
	return service.ContextWith[OutboundOptionsRegistry](context.Background(), chainTestOutboundRegistry{})
}

func TestChainDerivedTag(t *testing.T) {
	if got := chainDerivedTag("my-chain", 2); got != "chain-internal:my-chain:2" {
		t.Fatalf("unexpected derived tag: %q", got)
	}
}

func TestCompileChainOutboundsBuildsDetourTopology(t *testing.T) {
	ctx := chainTestContext()
	outbounds := []Outbound{
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: "chain-test-hop", Tag: "b", Options: &chainTestOutboundOptions{Name: "b"}},
		{Type: "chain-test-hop", Tag: "c", Options: &chainTestOutboundOptions{Name: "c"}},
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b", "c"}}},
	}

	compiled, err := CompileChainOutbounds(ctx, outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 7 {
		t.Fatalf("expected 7 outbounds, got %d", len(compiled))
	}

	expectedTags := []string{
		"a", "b", "c",
		"chain-internal:my-chain:0",
		"chain-internal:my-chain:1",
		"chain-internal:my-chain:2",
		"my-chain",
	}
	for i, want := range expectedTags {
		if compiled[i].Tag != want {
			t.Fatalf("outbound[%d]: expected tag %q, got %q", i, want, compiled[i].Tag)
		}
	}

	for i, wantDetour := range []string{"", "chain-internal:my-chain:0", "chain-internal:my-chain:1"} {
		clone, ok := compiled[3+i].Options.(*chainTestOutboundOptions)
		if !ok {
			t.Fatalf("derived outbound[%d] has unexpected options type %T", i, compiled[3+i].Options)
		}
		if clone.Detour != wantDetour {
			t.Fatalf("derived outbound[%d]: expected detour %q, got %q", i, wantDetour, clone.Detour)
		}
	}

	chain, ok := compiled[6].Options.(*ChainOutboundOptions)
	if !ok {
		t.Fatalf("chain has unexpected options type %T", compiled[6].Options)
	}
	if chain.FinalOutbound != "chain-internal:my-chain:2" {
		t.Fatalf("expected final outbound chain-internal:my-chain:2, got %q", chain.FinalOutbound)
	}

	originalB := outbounds[1].Options.(*chainTestOutboundOptions)
	if originalB.Detour != "" {
		t.Fatal("original outbound options were modified")
	}
}

func TestCompileChainOutboundsAllowsSingleHop(t *testing.T) {
	ctx := chainTestContext()
	outbounds := []Outbound{
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"a"}}},
	}

	compiled, err := CompileChainOutbounds(ctx, outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 3 {
		t.Fatalf("expected 3 outbounds, got %d", len(compiled))
	}
	chain := compiled[2].Options.(*ChainOutboundOptions)
	if chain.FinalOutbound != "chain-internal:my-chain:0" {
		t.Fatalf("expected final outbound chain-internal:my-chain:0, got %q", chain.FinalOutbound)
	}
}

func TestCompileChainOutboundsRejectsEmptyChain(t *testing.T) {
	ctx := chainTestContext()
	base := []Outbound{{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{}}}
	_, err := CompileChainOutbounds(ctx, base)
	if err == nil || !strings.Contains(err.Error(), "at least one outbound is required") {
		t.Fatalf("expected empty-chain error, got %v", err)
	}
}

func TestCompileChainOutboundsIsIdempotent(t *testing.T) {
	ctx := chainTestContext()
	outbounds := []Outbound{
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: "chain-test-hop", Tag: "b", Options: &chainTestOutboundOptions{Name: "b"}},
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b"}}},
	}

	first, err := CompileChainOutbounds(ctx, outbounds)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileChainOutbounds(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	if len(second) != len(first) {
		t.Fatalf("second compilation changed outbound count: %d -> %d", len(first), len(second))
	}
	for i := range first {
		if second[i].Tag != first[i].Tag {
			t.Fatalf("outbound[%d]: second compilation changed tag from %q to %q", i, first[i].Tag, second[i].Tag)
		}
	}
	chain := second[len(second)-1].Options.(*ChainOutboundOptions)
	if chain.FinalOutbound != "chain-internal:my-chain:1" {
		t.Fatalf("unexpected final outbound after second compilation: %q", chain.FinalOutbound)
	}
}

func TestCompileChainOutboundsRejectsInvalidReferences(t *testing.T) {
	ctx := chainTestContext()
	base := []Outbound{
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "missing"}}},
	}
	_, err := CompileChainOutbounds(ctx, base)
	if err == nil || !strings.Contains(err.Error(), "outbound not found: missing") {
		t.Fatalf("expected missing outbound error, got %v", err)
	}
}

func TestCompileChainOutboundsRejectsDuplicateHop(t *testing.T) {
	ctx := chainTestContext()
	base := []Outbound{
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: "chain-test-hop", Tag: "b", Options: &chainTestOutboundOptions{Name: "b"}},
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b", "a"}}},
	}
	_, err := CompileChainOutbounds(ctx, base)
	if err == nil || !strings.Contains(err.Error(), "duplicate hop: a") {
		t.Fatalf("expected duplicate hop error, got %v", err)
	}
}

func TestCompileChainOutboundsRejectsSelfReference(t *testing.T) {
	ctx := chainTestContext()
	base := []Outbound{
		{Type: C.TypeChain, Tag: "my-chain", Options: &ChainOutboundOptions{Outbounds: []string{"my-chain", "a"}}},
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
	}
	_, err := CompileChainOutbounds(ctx, base)
	if err == nil || !strings.Contains(err.Error(), "self reference is not allowed") {
		t.Fatalf("expected self-reference error, got %v", err)
	}
}

func TestCompileChainOutboundsRejectsNestedChain(t *testing.T) {
	ctx := chainTestContext()
	base := []Outbound{
		{Type: C.TypeChain, Tag: "inner", Options: &ChainOutboundOptions{Outbounds: []string{"a", "b"}}},
		{Type: C.TypeChain, Tag: "outer", Options: &ChainOutboundOptions{Outbounds: []string{"inner", "a"}}},
		{Type: "chain-test-hop", Tag: "a", Options: &chainTestOutboundOptions{Name: "a"}},
		{Type: "chain-test-hop", Tag: "b", Options: &chainTestOutboundOptions{Name: "b"}},
	}
	_, err := CompileChainOutbounds(ctx, base)
	if err == nil || !strings.Contains(err.Error(), "nested chain hop is not yet supported: inner") {
		t.Fatalf("expected nested-chain error, got %v", err)
	}
}

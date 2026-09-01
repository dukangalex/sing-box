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
	ctx := context.Background()
	return service.ContextWith[OutboundOptionsRegistry](ctx, chainTestOutboundRegistry{})
}

func chainTestOutbound(tag, name string) Outbound {
	return Outbound{
		Type: C.TypeSOCKS,
		Tag:  tag,
		Options: &chainTestOutboundOptions{
			Name: name,
		},
	}
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

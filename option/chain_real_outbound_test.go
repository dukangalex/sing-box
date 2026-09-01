package option

import (
	"context"
	"testing"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing/service"
)

type chainRealOutboundRegistry struct{}

func (chainRealOutboundRegistry) OptionTypes() []string {
	return []string{C.TypeSOCKS, "chain-test-hop"}
}

func (chainRealOutboundRegistry) CreateOptions(outboundType string) (any, bool) {
	switch outboundType {
	case C.TypeSOCKS:
		return new(SOCKSOutboundOptions), true
	case "chain-test-hop":
		return new(chainTestOutboundOptions), true
	default:
		return nil, false
	}
}

func chainRealOutboundContext() context.Context {
	return service.ContextWith[OutboundOptionsRegistry](context.Background(), chainRealOutboundRegistry{})
}

func TestCompileChainOutboundsClonesOfficialSOCKSOptions(t *testing.T) {
	ctx := chainRealOutboundContext()
	outbounds := []Outbound{
		{
			Type: C.TypeSOCKS,
			Tag:  "socks-a",
			Options: &SOCKSOutboundOptions{
				ServerOptions: ServerOptions{Server: "127.0.0.1", ServerPort: 1080},
				Version:       "5",
				Username:      "user-a",
				Password:      "pass-a",
			},
		},
		{
			Type: C.TypeSOCKS,
			Tag:  "socks-b",
			Options: &SOCKSOutboundOptions{
				ServerOptions: ServerOptions{Server: "127.0.0.2", ServerPort: 1081},
				Version:       "5",
				Username:      "user-b",
				Password:      "pass-b",
			},
		},
		{
			Type: C.TypeChain,
			Tag:  "socks-chain",
			Options: &ChainOutboundOptions{
				Outbounds: []string{"socks-a", "socks-b"},
			},
		},
	}

	compiled, err := CompileChainOutbounds(ctx, outbounds)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled) != 5 {
		t.Fatalf("expected 5 outbounds, got %d", len(compiled))
	}

	first, ok := compiled[2].Options.(*SOCKSOutboundOptions)
	if !ok {
		t.Fatalf("first derived outbound has unexpected options type %T", compiled[2].Options)
	}
	if first.Server != "127.0.0.1" || first.ServerPort != 1080 || first.Version != "5" || first.Username != "user-a" || first.Password != "pass-a" {
		t.Fatalf("first derived SOCKS options were not preserved: %+v", first)
	}
	if first.Detour != "" {
		t.Fatalf("first derived outbound unexpectedly has detour %q", first.Detour)
	}

	second, ok := compiled[3].Options.(*SOCKSOutboundOptions)
	if !ok {
		t.Fatalf("second derived outbound has unexpected options type %T", compiled[3].Options)
	}
	if second.Server != "127.0.0.2" || second.ServerPort != 1081 || second.Version != "5" || second.Username != "user-b" || second.Password != "pass-b" {
		t.Fatalf("second derived SOCKS options were not preserved: %+v", second)
	}
	if second.Detour != "chain-internal:socks-chain:0" {
		t.Fatalf("expected second derived detour chain-internal:socks-chain:0, got %q", second.Detour)
	}

	chain, ok := compiled[4].Options.(*ChainOutboundOptions)
	if !ok {
		t.Fatalf("chain has unexpected options type %T", compiled[4].Options)
	}
	if chain.FinalOutbound != "chain-internal:socks-chain:1" {
		t.Fatalf("expected final outbound chain-internal:socks-chain:1, got %q", chain.FinalOutbound)
	}

	originalSecond := outbounds[1].Options.(*SOCKSOutboundOptions)
	if originalSecond.Detour != "" {
		t.Fatal("original SOCKS outbound was modified")
	}
}

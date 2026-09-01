package option

import (
	"context"
	"strconv"

	C "github.com/sagernet/sing-box/constant"
	E "github.com/sagernet/sing/common/exceptions"
)

const chainDerivedTagPrefix = "chain-internal:"

// CompileChainOutbounds expands generic chain outbounds into derived outbounds
// using the official dialer detour mechanism. Original user outbounds are not modified.
func CompileChainOutbounds(ctx context.Context, outbounds []Outbound) ([]Outbound, error) {
	if len(outbounds) == 0 {
		return outbounds, nil
	}
	result := make([]Outbound, 0, len(outbounds))
	used := make(map[string]struct{}, len(outbounds))
	for i := range outbounds {
		if outbounds[i].Tag == "" {
			outbounds[i].Tag = strconv.Itoa(i)
		}
		if _, ok := used[outbounds[i].Tag]; ok {
			return nil, E.New("duplicate outbound/endpoint tag: ", outbounds[i].Tag)
		}
		used[outbounds[i].Tag] = struct{}{}
		result = append(result, outbounds[i])
	}
	for i := range outbounds {
		if outbounds[i].Type != C.TypeChain {
			continue
		}
		o, ok := outbounds[i].Options.(*ChainOutboundOptions)
		if !ok {
			return nil, E.New("invalid chain outbound options")
		}
		if len(o.Outbounds) < 2 {
			return nil, E.New("chain outbound requires at least 2 outbounds")
		}
		chainTag := outbounds[i].Tag
		previous := ""
		seenHops := make(map[string]struct{}, len(o.Outbounds))
		for index, hopTag := range o.Outbounds {
			if hopTag == "" {
				return nil, E.New("chain ", chainTag, ": empty outbound at index ", index)
			}
			if hopTag == chainTag {
				return nil, E.New("chain ", chainTag, ": self reference is not allowed")
			}
			if _, ok := seenHops[hopTag]; ok {
				return nil, E.New("chain ", chainTag, ": duplicate hop: ", hopTag)
			}
			seenHops[hopTag] = struct{}{}
			original, ok := findOutbound(outbounds, hopTag)
			if !ok {
				return nil, E.New("chain ", chainTag, ": outbound not found: ", hopTag)
			}
			if original.Type == C.TypeSelector || original.Type == C.TypeURLTest || original.Type == C.TypeChain {
				return nil, E.New("chain ", chainTag, ": group or nested chain hop is not yet supported: ", hopTag)
			}
			clone, err := cloneOutbound(ctx, original)
			if err != nil {
				return nil, E.Cause(err, "clone chain hop ", hopTag)
			}
			derived := chainDerivedTag(chainTag, index)
			if _, ok := used[derived]; ok {
				return nil, E.New("chain ", chainTag, ": derived tag collision: ", derived)
			}
			clone.Tag = derived
			if index > 0 {
				wrapper, ok := clone.Options.(DialerOptionsWrapper)
				if !ok {
					return nil, E.New("chain ", chainTag, ": outbound does not expose dialer options: ", hopTag)
				}
				dialerOptions := wrapper.TakeDialerOptions()
				dialerOptions.Detour = previous
				wrapper.ReplaceDialerOptions(dialerOptions)
			}
			used[derived] = struct{}{}
			result = append(result, clone)
			previous = derived
		}
		o.FinalOutbound = previous
	}
	return result, nil
}

func cloneOutbound(ctx context.Context, original Outbound) (Outbound, error) {
	content, err := original.MarshalJSONContext(ctx)
	if err != nil {
		return Outbound{}, err
	}
	var clone Outbound
	if err = clone.UnmarshalJSONContext(ctx, content); err != nil {
		return Outbound{}, err
	}
	return clone, nil
}

func findOutbound(outbounds []Outbound, tag string) (Outbound, bool) {
	for _, outbound := range outbounds {
		if outbound.Tag == tag {
			return outbound, true
		}
	}
	return Outbound{}, false
}

func chainDerivedTag(chainTag string, index int) string {
	return chainDerivedTagPrefix + chainTag + ":" + strconv.Itoa(index)
}

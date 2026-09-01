package option

import (
	"context"
	"reflect"

	C "github.com/sagernet/sing-box/constant"
	E "github.com/sagernet/sing/common/exceptions"
)

func CompileChainOutbounds(ctx context.Context, outbounds []Outbound) ([]Outbound, error) {
	_ = ctx
	original := make([]Outbound, len(outbounds))
	copy(original, outbounds)

	tags := make(map[string]int, len(original))
	for i := range original {
		if original[i].Tag == "" {
			original[i].Tag = chainIndex(i)
		}
		if _, exists := tags[original[i].Tag]; exists {
			return nil, E.New("duplicate outbound tag: ", original[i].Tag)
		}
		tags[original[i].Tag] = i
	}

	// Reserve every synthetic tag before mutating the topology. A user-defined
	// outbound must never be able to alias an internal Chain hop.
	reservedTags := make(map[string]struct{}, len(tags))
	for tag := range tags {
		reservedTags[tag] = struct{}{}
	}
	for i := range original {
		if original[i].Type != C.TypeChain {
			continue
		}
		options, ok := original[i].Options.(*ChainOutboundOptions)
		if !ok || len(options.Outbounds) < 2 {
			continue
		}
		for hopIndex := 0; hopIndex < len(options.Outbounds)-1; hopIndex++ {
			syntheticTag := chainDerivedTag(original[i].Tag, hopIndex)
			if _, exists := reservedTags[syntheticTag]; exists {
				return nil, E.New("chain outbound [", original[i].Tag, "] synthetic tag collides with outbound tag: ", syntheticTag)
			}
			reservedTags[syntheticTag] = struct{}{}
		}
	}

	result := make([]Outbound, 0, len(original))
	for _, outbound := range original {
		if outbound.Type != C.TypeChain {
			result = append(result, outbound)
		}
	}

	chainOutbounds := make([]Outbound, 0)
	for i := range original {
		if original[i].Type != C.TypeChain {
			continue
		}
		chain := original[i]
		options, ok := chain.Options.(*ChainOutboundOptions)
		if !ok {
			return nil, E.New("invalid chain options for outbound[", chain.Tag, "]")
		}
		if len(options.Outbounds) == 0 {
			return nil, E.New("chain outbound [", chain.Tag, "] requires at least one outbound")
		}

		seen := make(map[string]struct{}, len(options.Outbounds))
		for _, hopTag := range options.Outbounds {
			if _, duplicate := seen[hopTag]; duplicate {
				return nil, E.New("chain outbound [", chain.Tag, "] contains duplicate hop tag: ", hopTag)
			}
			seen[hopTag] = struct{}{}
			hopIndex, loaded := tags[hopTag]
			if !loaded {
				return nil, E.New("chain outbound [", chain.Tag, "] references unknown outbound tag: ", hopTag)
			}
			if hopTag == chain.Tag {
				return nil, E.New("chain outbound [", chain.Tag, "] self reference is not allowed")
			}
			if original[hopIndex].Type == C.TypeChain {
				return nil, E.New("nested chain is not supported (hop [", hopTag, "] is itself a chain)")
			}
		}

		internalTags := make([]string, len(options.Outbounds)-1)
		for hopIndex := range internalTags {
			internalTags[hopIndex] = chainDerivedTag(chain.Tag, hopIndex)
		}

		for hopIndex := 0; hopIndex < len(options.Outbounds)-1; hopIndex++ {
			hopTag := options.Outbounds[hopIndex]
			hop := original[tags[hopTag]]
			if hop.Type == C.TypeDirect {
				return nil, E.New("direct outbound cannot be used as a non-final hop in chain [", chain.Tag, "]")
			}
			cloned, err := cloneChainOptions(hop.Options)
			if err != nil {
				return nil, E.Cause(err, "chain outbound [", chain.Tag, "] hop [", hopTag, "]")
			}
			wrapper, ok := cloned.(DialerOptionsWrapper)
			if !ok {
				return nil, E.New("outbound type [", hop.Type, "] cannot be used as an intermediate hop in chain (missing DialerOptions support)")
			}
			dialerOptions := wrapper.TakeDialerOptions()
			if dialerOptions.Detour != "" {
				return nil, E.New("outbound [", hopTag, "] already has a detour and cannot be used as an intermediate hop in chain [", chain.Tag, "]")
			}
			if hopIndex+1 < len(internalTags) {
				dialerOptions.Detour = internalTags[hopIndex+1]
			} else {
				dialerOptions.Detour = options.Outbounds[len(options.Outbounds)-1]
			}
			wrapper.ReplaceDialerOptions(dialerOptions)
			result = append(result, Outbound{Type: hop.Type, Tag: internalTags[hopIndex], Options: cloned})
		}

		optionsCopy := *options
		optionsCopy.Outbounds = append([]string(nil), options.Outbounds...)
		if len(internalTags) > 0 {
			optionsCopy.EntryOutbound = internalTags[0]
		} else {
			optionsCopy.EntryOutbound = options.Outbounds[0]
		}
		chainOutbounds = append(chainOutbounds, Outbound{Type: C.TypeChain, Tag: chain.Tag, Options: &optionsCopy})
	}

	result = append(result, chainOutbounds...)
	return result, nil
}

func cloneChainOptions(options any) (any, error) {
	if options == nil {
		return nil, E.New("missing outbound options")
	}
	value := reflect.ValueOf(options)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, E.New("outbound options are not pointer-backed")
	}
	clone := reflect.New(value.Elem().Type())
	clone.Elem().Set(value.Elem())
	return clone.Interface(), nil
}

func chainDerivedTag(tag string, index int) string {
	return tag + ":chain:" + chainIndex(index)
}

func chainIndex(index int) string {
	if index < 10 {
		return string(rune('0' + index))
	}
	return chainIndex(index/10) + string(rune('0'+index%10))
}

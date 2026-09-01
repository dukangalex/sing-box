package option

import (
	"context"
	"reflect"

	C "github.com/sagernet/sing-box/constant"
	E "github.com/sagernet/sing/common/exceptions"
)

func CompileChainOutbounds(ctx context.Context, outbounds []Outbound) ([]Outbound, error) {
	result := make([]Outbound, 0, len(outbounds))
	result = append(result, outbounds...)

	// Resolve implicit tags exactly as the runtime does for top-level outbounds.
	tags := make(map[string]int, len(result))
	for i := range result {
		if result[i].Tag == "" {
			result[i].Tag = formatChainIndex(i)
		}
		if _, exists := tags[result[i].Tag]; exists {
			return nil, E.New("duplicate outbound tag: ", result[i].Tag)
		}
		tags[result[i].Tag] = i
	}

	for i := 0; i < len(outbounds); i++ {
		if outbounds[i].Type != C.TypeChain {
			continue
		}
		options, ok := result[i].Options.(*ChainOutboundOptions)
		if !ok {
			return nil, E.New("invalid chain options for outbound[", result[i].Tag, "]")
		}
		if len(options.Outbounds) == 0 {
			return nil, E.New("chain outbound [", result[i].Tag, "] requires at least 1 outbound")
		}

		seen := make(map[string]struct{}, len(options.Outbounds))
		for _, hopTag := range options.Outbounds {
			if _, duplicate := seen[hopTag]; duplicate {
				return nil, E.New("chain outbound [", result[i].Tag, "] has duplicate hop: ", hopTag)
			}
			seen[hopTag] = struct{}{}

			hopIndex, loaded := tags[hopTag]
			if !loaded {
				return nil, E.New("chain outbound [", result[i].Tag, "] references unknown outbound: ", hopTag)
			}
			if hopTag == result[i].Tag {
				return nil, E.New("chain outbound [", result[i].Tag, "] self reference is not allowed")
			}
			if result[hopIndex].Type == C.TypeChain {
				return nil, E.New("nested chain outbound is not supported: ", hopTag)
			}
		}

		internalTags := make([]string, len(options.Outbounds))
		for hopIndex := range options.Outbounds {
			internalTags[hopIndex] = chainDerivedTag(result[i].Tag, hopIndex)
		}
		for hopIndex, hopTag := range options.Outbounds {
			hop := result[tags[hopTag]]
			if hopIndex == len(options.Outbounds)-1 {
				continue
			}
			cloned, err := cloneChainOptions(hop.Options)
			if err != nil {
				return nil, E.Cause(err, "chain outbound [", result[i].Tag, "] hop [", hopTag, "]")
			}
			wrapper, ok := cloned.(DialerOptionsWrapper)
			if !ok {
				return nil, E.New("outbound type [", hop.Type, "] cannot be used as a non-final chain hop because it has no DialerOptions")
			}
			dialerOptions := wrapper.TakeDialerOptions()
			if dialerOptions.Detour != "" {
				return nil, E.New("outbound [", hopTag, "] already has detour and cannot be used as a chain hop")
			}
			dialerOptions.Detour = internalTags[hopIndex+1]
			wrapper.ReplaceDialerOptions(dialerOptions)
			result = append(result, Outbound{
				Type:    hop.Type,
				Tag:     internalTags[hopIndex],
				Options: cloned,
			})
		}

		finalTag := internalTags[len(internalTags)-1]
		finalHop := result[tags[options.Outbounds[len(options.Outbounds)-1]]]
		optionsCopy := *options
		optionsCopy.Outbounds = append([]string(nil), options.Outbounds...)
		_ = finalHop
		result = append(result, Outbound{
			Type:    C.TypeChain,
			Tag:     result[i].Tag,
			Options: &optionsCopy,
		})
		// The chain outbound itself will resolve the final hop through the manager.
		_ = finalTag
	}

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
	return tag + ":chain:" + formatChainIndex(index)
}

func formatChainIndex(index int) string {
	if index == 0 {
		return "0"
	}
	return formatChainIndexRecursive(index)
}

func formatChainIndexRecursive(index int) string {
	if index < 10 {
		return string(rune('0' + index))
	}
	return formatChainIndexRecursive(index/10) + string(rune('0' + index%10))
}

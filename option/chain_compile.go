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

			hopTag := options.Outbounds[hopIndex]
			hopIdx, loaded := tags[hopTag]
			if !loaded {
				continue
			}
			hop := original[hopIdx]
			memberTags, err := groupMemberTags(hop)
			if err != nil || memberTags == nil {
				continue
			}
			// Pre-reserve tags for both direct members and recursively flattened leaves
			flat, err := collectLeafProxyTags(memberTags, tags, original, nil)
			if err != nil {
				continue
			}
			for _, memberTag := range flat {
				mt := chainGroupMemberTag(original[i].Tag, hopIndex, memberTag)
				if _, exists := reservedTags[mt]; exists {
					return nil, E.New("chain outbound [", original[i].Tag, "] synthetic member tag collides: ", mt)
				}
				reservedTags[mt] = struct{}{}
			}
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

			nextDetour := options.Outbounds[len(options.Outbounds)-1]
			if hopIndex+1 < len(internalTags) {
				nextDetour = internalTags[hopIndex+1]
			}

			expanded, err := expandChainIntermediateHop(chain.Tag, hopIndex, hop, tags, original, nextDetour, internalTags[hopIndex])
			if err != nil {
				return nil, err
			}
			result = append(result, expanded...)
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

// collectLeafProxyTags recursively walks selector/urltest members and returns
// only concrete proxy tags (skips direct/block/dns/chain). Used so intermediate
// hops can accept nested groups the way v2rayNG/NekoBox/mihomo users expect.
func collectLeafProxyTags(memberTags []string, tags map[string]int, original []Outbound, visiting map[string]struct{}) ([]string, error) {
	if visiting == nil {
		visiting = make(map[string]struct{})
	}
	var leaves []string
	seen := make(map[string]struct{})
	var walk func(tag string) error
	walk = func(tag string) error {
		if tag == "" {
			return nil
		}
		if _, dup := seen[tag]; dup {
			return nil
		}
		if _, cycle := visiting[tag]; cycle {
			return nil // break cycles quietly
		}
		idx, loaded := tags[tag]
		if !loaded {
			return E.New("unknown member: ", tag)
		}
		member := original[idx]
		switch member.Type {
		case C.TypeDirect, C.TypeBlock, C.TypeDNS, C.TypeChain:
			return nil
		case C.TypeSelector, C.TypeURLTest:
			visiting[tag] = struct{}{}
			nested, err := groupMemberTags(member)
			if err != nil {
				delete(visiting, tag)
				return err
			}
			for _, n := range nested {
				if err := walk(n); err != nil {
					delete(visiting, tag)
					return err
				}
			}
			delete(visiting, tag)
			return nil
		default:
			seen[tag] = struct{}{}
			leaves = append(leaves, tag)
			return nil
		}
	}
	for _, t := range memberTags {
		if err := walk(t); err != nil {
			return nil, err
		}
	}
	return leaves, nil
}

// expandChainIntermediateHop clones a hop so traffic goes through nextDetour.
// Leaf dialers get detour set directly. selector/urltest groups expand each
// usable member (recursively flattening nested groups) with detour, then emit
// a synthetic group pointing at those members.
// direct/block/dns members are skipped (they cannot be intermediate chain hops).
func expandChainIntermediateHop(
	chainTag string,
	hopIndex int,
	hop Outbound,
	tags map[string]int,
	original []Outbound,
	nextDetour string,
	syntheticHopTag string,
) ([]Outbound, error) {
	members, err := groupMemberTags(hop)
	if err != nil {
		return nil, E.Cause(err, "chain outbound [", chainTag, "] hop [", hop.Tag, "]")
	}
	if members != nil {
		if len(members) == 0 {
			return nil, E.New("chain outbound [", chainTag, "] hop [", hop.Tag, "] group has no members")
		}
		// Recursively flatten nested selector/urltest so nested groups work
		// (matches user expectation from v2rayNG / NekoBox / mihomo).
		leafTags, err := collectLeafProxyTags(members, tags, original, nil)
		if err != nil {
			return nil, E.Cause(err, "chain outbound [", chainTag, "] hop [", hop.Tag, "]")
		}
		if len(leafTags) == 0 {
			return nil, E.New("chain outbound [", chainTag, "] hop [", hop.Tag, "] has no usable proxy members after excluding direct/block/dns and expanding nested groups")
		}

		out := make([]Outbound, 0, len(leafTags)+1)
		syntheticMembers := make([]string, 0, len(leafTags))
		for _, memberTag := range leafTags {
			idx, loaded := tags[memberTag]
			if !loaded {
				return nil, E.New("chain outbound [", chainTag, "] hop [", hop.Tag, "] references unknown member: ", memberTag)
			}
			member := original[idx]
			cloned, err := cloneChainOptions(member.Options)
			if err != nil {
				return nil, E.Cause(err, "chain outbound [", chainTag, "] member [", memberTag, "]")
			}
			wrapper, ok := cloned.(DialerOptionsWrapper)
			if !ok {
				return nil, E.New("outbound type [", member.Type, "] cannot be used as a chain hop member (missing DialerOptions support)")
			}
			dialerOptions := wrapper.TakeDialerOptions()
			if dialerOptions.Detour != "" {
				return nil, E.New("outbound [", memberTag, "] already has a detour and cannot be used in chain [", chainTag, "]")
			}
			dialerOptions.Detour = nextDetour
			wrapper.ReplaceDialerOptions(dialerOptions)
			sTag := chainGroupMemberTag(chainTag, hopIndex, memberTag)
			syntheticMembers = append(syntheticMembers, sTag)
			out = append(out, Outbound{Type: member.Type, Tag: sTag, Options: cloned})
		}

		// Synthetic intermediate hop: always a selector over flattened leaves so
		// the user can still switch nodes after connect (dashboard).
		groupClone := &SelectorOutboundOptions{
			Outbounds: append([]string(nil), syntheticMembers...),
		}
		// Preserve urltest preference by keeping type when hop was urltest
		hopType := C.TypeSelector
		if hop.Type == C.TypeURLTest {
			if ut, ok := hop.Options.(*URLTestOutboundOptions); ok {
				hopType = C.TypeURLTest
				urlClone := &URLTestOutboundOptions{
					Outbounds: append([]string(nil), syntheticMembers...),
					URL:       ut.URL,
					Interval:  ut.Interval,
					Tolerance: ut.Tolerance,
					IdleTimeout: ut.IdleTimeout,
				}
				out = append(out, Outbound{Type: hopType, Tag: syntheticHopTag, Options: urlClone})
				return out, nil
			}
		}
		out = append(out, Outbound{Type: hopType, Tag: syntheticHopTag, Options: groupClone})
		return out, nil
	}

	cloned, err := cloneChainOptions(hop.Options)
	if err != nil {
		return nil, E.Cause(err, "chain outbound [", chainTag, "] hop [", hop.Tag, "]")
	}
	wrapper, ok := cloned.(DialerOptionsWrapper)
	if !ok {
		return nil, E.New("outbound type [", hop.Type, "] cannot be used as an intermediate hop in chain (missing DialerOptions support)")
	}
	dialerOptions := wrapper.TakeDialerOptions()
	if dialerOptions.Detour != "" {
		return nil, E.New("outbound [", hop.Tag, "] already has a detour and cannot be used as an intermediate hop in chain [", chainTag, "]")
	}
	dialerOptions.Detour = nextDetour
	wrapper.ReplaceDialerOptions(dialerOptions)
	return []Outbound{{Type: hop.Type, Tag: syntheticHopTag, Options: cloned}}, nil
}

func groupMemberTags(hop Outbound) ([]string, error) {
	switch hop.Type {
	case C.TypeSelector:
		opt, ok := hop.Options.(*SelectorOutboundOptions)
		if !ok {
			return nil, E.New("invalid selector options")
		}
		return append([]string(nil), opt.Outbounds...), nil
	case C.TypeURLTest:
		opt, ok := hop.Options.(*URLTestOutboundOptions)
		if !ok {
			return nil, E.New("invalid urltest options")
		}
		return append([]string(nil), opt.Outbounds...), nil
	default:
		return nil, nil
	}
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

func chainGroupMemberTag(chainTag string, hopIndex int, memberTag string) string {
	return chainDerivedTag(chainTag, hopIndex) + ":" + memberTag
}

func chainIndex(index int) string {
	if index < 10 {
		return string(rune('0' + index))
	}
	return chainIndex(index/10) + string(rune('0'+index%10))
}

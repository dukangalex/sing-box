package chain

import (
	"fmt"

	"github.com/sagernet/sing-box/adapter"
)

// GroupResolver resolves the leaf outbound currently selected by a Sing-box
// OutboundGroup. It relies only on the public OutboundGroup/OutboundManager
// contracts, so Selector, URLTest and future groups keep their native behavior.
type GroupResolver struct {
	manager adapter.OutboundManager
	maxDepth int
}

func NewGroupResolver(manager adapter.OutboundManager) *GroupResolver {
	return &GroupResolver{manager: manager, maxDepth: 32}
}

func (r *GroupResolver) Resolve(group adapter.Outbound) (adapter.Outbound, error) {
	if group == nil {
		return nil, fmt.Errorf("chain: outbound is nil")
	}
	if r.manager == nil {
		return nil, fmt.Errorf("chain: outbound manager is nil")
	}

	current := group
	for depth := 0; depth < r.maxDepth; depth++ {
		outboundGroup, ok := current.(adapter.OutboundGroup)
		if !ok {
			return current, nil
		}

		tag := outboundGroup.Now()
		if tag == "" {
			return nil, fmt.Errorf("chain: group %q has no selected outbound", current.Tag())
		}

		next, loaded := r.manager.Outbound(tag)
		if !loaded || next == nil {
			return nil, fmt.Errorf("chain: selected outbound not found: %s", tag)
		}
		current = next
	}

	return nil, fmt.Errorf("chain: outbound group nesting exceeds %d levels", r.maxDepth)
}

// ResolveEntry implements the legacy Resolver interface using a strategy tag.
func (r *GroupResolver) ResolveEntry(strategyTag string) (string, error) {
	if r.manager == nil {
		return "", fmt.Errorf("chain: outbound manager is nil")
	}
	if strategyTag == "" {
		return "", fmt.Errorf("chain: strategy tag is empty")
	}
	outbound, loaded := r.manager.Outbound(strategyTag)
	if !loaded || outbound == nil {
		return "", fmt.Errorf("chain: strategy outbound not found: %s", strategyTag)
	}
	resolved, err := r.Resolve(outbound)
	if err != nil {
		return "", err
	}
	return resolved.Tag(), nil
}

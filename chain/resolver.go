package chain

import "fmt"

// Resolver resolves the outbound that the user's existing strategy currently selects.
// It deliberately does not implement strategy selection itself.
type Resolver interface {
	ResolveEntry(tag string) (string, error)
}

// StaticResolver is the minimal resolver used by unit tests and by integration code
// that already has the strategy result. Production strategy adapters can implement Resolver.
type StaticResolver struct {
	Entry string
}

func (r StaticResolver) ResolveEntry(_ string) (string, error) {
	if r.Entry == "" {
		return "", fmt.Errorf("chain: selected entry is empty")
	}
	return r.Entry, nil
}

// ResolvePlan resolves the user's existing strategy result and validates the strict path.
func ResolvePlan(p Plan, strategyTag string, resolver Resolver) (Plan, error) {
	if p.Mode == ModeDisabled {
		return p, nil
	}
	if resolver == nil {
		return Plan{}, fmt.Errorf("chain: entry resolver is nil")
	}
	entry, err := resolver.ResolveEntry(strategyTag)
	if err != nil {
		return Plan{}, err
	}
	p.Entry = entry
	if err := p.Validate(); err != nil {
		return Plan{}, err
	}
	return p, nil
}

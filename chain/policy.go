package chain

import "fmt"

// Policy decides whether Chain wraps a particular existing strategy group.
// It never changes that group's members or selection algorithm.
type Policy struct {
	Mode    Mode
	Landing string
	Groups  map[string]struct{}
}

func NewPolicy(mode Mode, landing string, groups []string) (Policy, error) {
	p := Policy{Mode: mode, Landing: landing, Groups: make(map[string]struct{}, len(groups))}
	for _, group := range groups {
		if group == "" {
			return Policy{}, fmt.Errorf("chain: empty group name")
		}
		p.Groups[group] = struct{}{}
	}
	if mode != ModeDisabled && landing == "" {
		return Policy{}, fmt.Errorf("chain: landing is required")
	}
	if mode == ModeLocal && len(p.Groups) == 0 {
		return Policy{}, fmt.Errorf("chain: local mode requires at least one group")
	}
	if mode != ModeDisabled && mode != ModeLocal && mode != ModeGlobal {
		return Policy{}, fmt.Errorf("chain: unsupported mode %q", mode)
	}
	return p, nil
}

func (p Policy) Applies(group string) bool {
	switch p.Mode {
	case ModeGlobal:
		return true
	case ModeLocal:
		_, ok := p.Groups[group]
		return ok
	default:
		return false
	}
}

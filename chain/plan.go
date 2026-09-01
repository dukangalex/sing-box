package chain

import "fmt"

// Mode controls where the Chain policy is applied.
type Mode uint8

const (
	ModeDisabled Mode = iota
	ModeLocal
	ModeGlobal
)

// Plan describes a strict connection path without becoming an outbound itself.
type Plan struct {
	Entry   string
	Landing string
	Mode    Mode
}

// Validate rejects incomplete or recursive Chain paths before dialing.
func (p Plan) Validate() error {
	if p.Mode == ModeDisabled {
		return nil
	}
	if p.Entry == "" {
		return fmt.Errorf("chain: entry is required")
	}
	if p.Landing == "" {
		return fmt.Errorf("chain: landing is required")
	}
	if p.Entry == p.Landing {
		return fmt.Errorf("chain: entry and landing must differ")
	}
	return nil
}

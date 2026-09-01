package option

// ChainMode controls how the Chain capability is applied.
type ChainMode string

const (
	ChainModeDisabled ChainMode = "disabled"
	ChainModeLocal    ChainMode = "local"
	ChainModeGlobal   ChainMode = "global"
)

// ChainOptions describes the additional chain policy without replacing any
// existing Sing-box strategy configuration.
type ChainOptions struct {
	Mode    ChainMode `json:"mode,omitempty" yaml:"mode,omitempty"`
	Landing string    `json:"landing,omitempty" yaml:"landing,omitempty"`
	Groups  []string  `json:"groups,omitempty" yaml:"groups,omitempty"`
}

func (o *ChainOptions) IsEnabled() bool {
	return o != nil && o.Mode != ChainModeDisabled && o.Landing != ""
}

func (o *ChainOptions) AppliesToGroup(group string) bool {
	if o == nil || o.Mode == ChainModeDisabled {
		return false
	}
	if o.Mode == ChainModeGlobal {
		return true
	}
	for _, name := range o.Groups {
		if name == group {
			return true
		}
	}
	return false
}

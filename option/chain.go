package option

// ChainOutboundOptions describes a generic ordered chain of existing outbounds.
// The order is user-defined and carries no entry/landing role semantics.
type ChainOutboundOptions struct {
	Outbounds     []string `json:"outbounds,omitempty" yaml:"outbounds,omitempty"`
	FinalOutbound string   `json:"-" yaml:"-"`
}

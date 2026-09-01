package option

// ChainOutboundOptions describes a TCP chain:
//
//	entry -> SOCKS landing -> destination
//
// Entry is an outbound tag and may point to a Selector or URLTest outbound.
// The selected result is resolved by the Chain runtime for each connection.
type ChainOutboundOptions struct {
	Entry string `json:"entry" reference:"outbound"`

	ServerOptions

	Version  string `json:"version,omitempty" enum:"4,4a,5"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

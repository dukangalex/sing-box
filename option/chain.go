package option

type ChainOutboundOptions struct {
	Outbounds     []string `json:"outbounds" reference:"outbound"`
	EntryOutbound string   `json:"-"`
}

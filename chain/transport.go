package chain

import (
	"context"
	"fmt"
	"net"
)

// Dialer is the minimal interface required by Chain transport.
// Keeping this interface small prevents Chain from depending on a concrete
// Sing-box dialer implementation and makes the transport independently testable.
type Dialer interface {
	DialContext(ctx context.Context, network string, destination net.Addr) (net.Conn, error)
}

// Transport represents the strict Entry -> Landing connection path.
// The Landing dialer is deliberately separate from the Entry dialer: the Entry
// establishes the connection to the Landing endpoint, while the Landing side
// is responsible for forwarding that connection to the final destination.
type Transport struct {
	Entry   Dialer
	Landing net.Addr
}

// Dial establishes the first leg of a Chain path. It never falls back to the
// Entry as a direct Target connection when the Landing leg fails.
func (t Transport) Dial(ctx context.Context, network string) (net.Conn, error) {
	if t.Entry == nil {
		return nil, fmt.Errorf("chain: entry dialer is nil")
	}
	if t.Landing == nil {
		return nil, fmt.Errorf("chain: landing address is nil")
	}
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("chain: unsupported network %q", network)
	}
	conn, err := t.Entry.DialContext(ctx, network, t.Landing)
	if err != nil {
		return nil, fmt.Errorf("chain: entry to landing: %w", err)
	}
	return conn, nil
}

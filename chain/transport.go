package chain

import (
	"context"
	"fmt"
	"net"

	M "github.com/sagernet/sing/common/metadata"
	N "github.com/sagernet/sing/common/network"
)

// Dialer is the minimal Sing-box network dialer required by Chain transport.
type Dialer interface {
	N.Dialer
}

// Transport represents the strict first leg of a Chain path.
// The Entry establishes a connection to the Landing endpoint. It never falls
// back to dialing the final Target directly.
type Transport struct {
	Entry   Dialer
	Landing M.Socksaddr
}

// Dial establishes Entry -> Landing. The second leg, Landing -> Target, is
// performed by the Landing outbound's own protocol implementation.
func (t Transport) Dial(ctx context.Context, network string) (net.Conn, error) {
	if t.Entry == nil {
		return nil, fmt.Errorf("chain: entry dialer is nil")
	}
	if t.Landing.IsZero() {
		return nil, fmt.Errorf("chain: landing address is empty")
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

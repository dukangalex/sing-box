package openconnect

import (
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestFortinetBoundedStack(t *testing.T) {
	previous := debug.SetMaxStack(8 << 20)
	defer debug.SetMaxStack(previous)
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><extension>`
	suffix := `</extension></sslvpn-tunnel>`
	for _, tc := range []struct {
		depth  int
		reject bool
	}{{254, false}, {255, true}, {9998, true}, {14998, true}} {
		xml := prefix + strings.Repeat("<x>", tc.depth) + strings.Repeat("</x>", tc.depth) + suffix
		_, err := parseFortinetXMLConfiguration([]byte(xml), time.Now())
		if (err != nil) != tc.reject {
			t.Fatalf("nested=%d err=%v", tc.depth, err)
		}
		if tc.reject && !strings.Contains(err.Error(), "exceeds nesting limit") {
			t.Fatalf("wrong error: %v", err)
		}
		t.Logf("document depth=%d rejected=%v", tc.depth+2, tc.reject)
	}
}

package openconnect

import (
	"strings"
	"testing"
	"time"
)

func TestGuardDepth(t *testing.T) {
	for _, tt := range []struct {
		n         int
		wantError bool
	}{{1, false}, {256, false}, {257, true}, {10000, true}, {15000, true}} {
		s := strings.Repeat("<a>", tt.n) + strings.Repeat("</a>", tt.n)
		err := validateFortinetDepth([]byte(s))
		if (err != nil) != tt.wantError {
			t.Fatalf("depth %d: %v", tt.n, err)
		}
		if tt.wantError {
			_, err = parseFortinetXMLConfiguration([]byte(s), time.Now())
			if err == nil || !strings.Contains(err.Error(), "exceeds nesting limit") {
				t.Fatalf("parse depth=%d err=%v", tt.n, err)
			}
		}
		t.Logf("depth=%d rejected=%v", tt.n, err != nil)
	}
}
func TestGuardMalformedAndTrailing(t *testing.T) {
	for _, s := range []string{"<a><b></a>", "<a>", "<a></a><b></b>"} {
		_, err := parseFortinetXMLConfiguration([]byte(s), time.Now())
		if err == nil {
			t.Fatalf("expected failure: %s", s)
		}
		t.Logf("input=%s err=%v", s, err)
	}
}

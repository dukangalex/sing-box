package openconnect

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// This is a host allocation observation, not Android peak RSS or production acceptance.
func TestExperimentalTwoMiBCompatibilityResource(t *testing.T) {
	for _, n := range []int{(1 << 20) + 1, fortinetBudgetLexicalBytes - 128} {
		doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><ignored note='` + strings.Repeat("x", n) + `'/></sslvpn-tunnel>`)
		baseline, err := parseFortinetBaseline(doc, time.Unix(0, 0))
		if err != nil {
			t.Fatal(err)
		}
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		got, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
		runtime.ReadMemStats(&b)
		if err != nil || !reflect.DeepEqual(baseline, got) {
			t.Fatalf("n=%d err=%v unequal=%t", n, err, !reflect.DeepEqual(baseline, got))
		}
		t.Logf("input=%d alloc=%d matching baseline", len(doc), b.TotalAlloc-a.TotalAlloc)
	}
}

// The existing HTTP body limit remains an application-policy boundary.
func TestExperimentalBodyBoundaryRejectsAboveLimit(t *testing.T) {
	doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><ignored note='` + strings.Repeat("x", fortinetBudgetLexicalBytes+1) + `'/></sslvpn-tunnel>`)
	baseline, e1 := parseFortinetBaseline(doc, time.Unix(0, 0))
	candidate, e2 := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
	if e1 != nil || baseline == nil || e2 == nil || candidate != nil || !strings.Contains(e2.Error(), "body bytes") {
		t.Fatalf("baseline=%v candidate=%v", e1, e2)
	}
	t.Logf("input=%d baseline accepted; experimental candidate rejected: %v", len(doc), e2)
}

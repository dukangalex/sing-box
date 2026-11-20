package openconnect

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

// Experimental policy tests: lexical and construct limits are application budgets,
// not Fortinet protocol maxima. Originals are retained in xml-balanced-xnet062-module.
func TestV2LexicalBudgetAndKnownCompatibility(t *testing.T) {
	for _, n := range []int{65536, (1 << 20) + 64, fortinetBudgetLexicalBytes - 128, fortinetBudgetLexicalBytes + 64} {
		doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/></ipv4><ignored a="` + strings.Repeat("x", n) + `"/></sslvpn-tunnel>`)
		cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
		want := n < fortinetBudgetLexicalBytes
		if (err == nil) != want || (cfg != nil) != want {
			t.Fatalf("length=%d accepted=%v err=%v", n, cfg != nil, err)
		}
	}
}
func TestV2OversizeBodyFormsRejectBeforeDecoder(t *testing.T) {
	forms := []string{
		`<ignored a="` + strings.Repeat("x", fortinetBudgetLexicalBytes+1) + `"/>`,
		`<!--` + strings.Repeat("x", fortinetBudgetLexicalBytes+1) + `-->`,
		`<![CDATA[` + strings.Repeat("x", fortinetBudgetLexicalBytes+1) + `]]>`,
		`<?p ` + strings.Repeat("x", fortinetBudgetLexicalBytes+1) + `?>`,
		strings.Repeat("x", fortinetBudgetLexicalBytes+1),
	}
	for i, s := range forms {
		doc := []byte(`<sslvpn-tunnel>` + s + `</sslvpn-tunnel>`)
		runtime.GC()
		var a, b runtime.MemStats
		runtime.ReadMemStats(&a)
		cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
		runtime.ReadMemStats(&b)
		if err == nil || cfg != nil || !strings.Contains(err.Error(), "body bytes") {
			t.Fatalf("form %d: %v", i, err)
		}
		if b.TotalAlloc-a.TotalAlloc > 1<<20 {
			t.Fatalf("form %d allocated %d", i, b.TotalAlloc-a.TotalAlloc)
		}
		t.Logf("form %d input=%d rejection alloc=%d", i, len(doc), b.TotalAlloc-a.TotalAlloc)
	}
}
func TestV2RetainsLargeValidDomainDocument(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	item := `<dns domain="` + strings.Repeat("x", 50) + `"/>`
	doc := []byte(prefix + strings.Repeat(item, 65535) + `</ipv4></sslvpn-tunnel>`)
	runtime.GC()
	var a, b runtime.MemStats
	runtime.ReadMemStats(&a)
	cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
	runtime.ReadMemStats(&b)
	if err != nil || cfg == nil {
		t.Fatalf("large domain config rejected: %v", err)
	}
	t.Logf("valid 65,535-domain input=%d accepted, cumulative allocation=%d", len(doc), b.TotalAlloc-a.TotalAlloc)
}

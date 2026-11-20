package openconnect

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBudgetPreflightLexicalAndBoundary(t *testing.T) {
	attr := ` a=""`
	for _, n := range []int{fortinetBudgetAttributes, fortinetBudgetAttributes + 1} {
		b := []byte(`<sslvpn-tunnel` + strings.Repeat(attr, n) + `/>`)
		err := preflightFortinetXMLBudget(b)
		if (err == nil) != (n == fortinetBudgetAttributes) {
			t.Fatalf("attributes %d: %v", n, err)
		}
	}
	// Quoted equals and markup in comments/CDATA/PI do not create attributes.
	for _, x := range []string{
		`<sslvpn-tunnel><ignored x="` + strings.Repeat("=", fortinetBudgetAttributes+1) + `"/></sslvpn-tunnel>`,
		`<!--` + strings.Repeat(attr, fortinetBudgetAttributes+1) + `--><x/>`,
		`<![CDATA[` + strings.Repeat(attr, fortinetBudgetAttributes+1) + `]]><x/>`,
		`<?p ` + strings.Repeat(attr, fortinetBudgetAttributes+1) + `?><x/>`,
		`<!DOCTYPE x [<!-- [ ' > -->]><x/>`,
	} {
		if err := preflightFortinetXMLBudget([]byte(x)); err != nil {
			t.Fatal(err)
		}
	}
	// DTD comments must not mask a real hostile token following the directive.
	for _, p := range []string{`<!DOCTYPE x [<!-- [ ' > -->]>`, `<!DOCTYPE x [<!ELEMENT x ANY>]>`, `<!DOCTYPE x SYSTEM 'a>b'>`} {
		if err := preflightFortinetXMLBudget([]byte(p + `<x` + strings.Repeat(attr, fortinetBudgetAttributes+1) + `/>`)); err == nil {
			t.Fatal("directive hid attributes")
		}
	}
	for _, s := range []string{`<!--`, `<![CDATA[`, `<?p`, `<x a='`, `<!DOCTYPE x [<!--`} {
		if preflightFortinetXMLBudget([]byte(s)) == nil {
			t.Fatal("accepted unterminated", s)
		}
	}
	if err := preflightFortinetXMLBudget([]byte(strings.Repeat(`<x/>`, fortinetBudgetMarkup+1))); err == nil || !strings.Contains(err.Error(), "markup constructs") {
		t.Fatalf("markup cap not enforced: %v", err)
	}
	if err := preflightFortinetXMLBudget([]byte(strings.Repeat(`<!--c-->`, fortinetBudgetMarkup+1))); err == nil || !strings.Contains(err.Error(), "markup constructs") {
		t.Fatalf("comment markup cap not enforced: %v", err)
	}
}

func TestBudgetSemanticRejectsAtomically(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	for _, doc := range []string{
		prefix + strings.Repeat(`<dns domain="x.test"/>`, fortinetBudgetSemantic) + suffix,
		prefix + `<split-dns domains="` + strings.Repeat("x,", fortinetBudgetSemantic) + `x" dnsserver1="192.0.2.53"/>` + suffix,
		prefix + `<ignored ` + strings.Repeat(`a="" `, fortinetBudgetAttributes+1) + `/>` + suffix,
		prefix + strings.Repeat(`<x/>`, fortinetBudgetMarkup) + suffix,
	} {
		cfg, err := parseFortinetXMLConfiguration([]byte(doc), time.Unix(0, 0))
		if err == nil || cfg != nil || !strings.Contains(err.Error(), "resource budget") {
			t.Fatalf("cfg nil=%v error=%v", cfg == nil, err)
		}
	}
}

func TestBudgetDenseRejectBeforeDecoderAllocation(t *testing.T) {
	doc := []byte(`<sslvpn-tunnel><ignored ` + strings.Repeat(`a="" `, 1500000) + `/></sslvpn-tunnel>`)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
	runtime.ReadMemStats(&after)
	delta := after.TotalAlloc - before.TotalAlloc
	if cfg != nil || err == nil || !strings.Contains(err.Error(), "attributes") {
		t.Fatal(err)
	}
	if delta > 1<<20 {
		t.Fatalf("rejection allocated %d bytes", delta)
	}
	t.Logf("dense token bytes=%d rejected before decoder, allocated=%d", len(doc), delta)
}

func FuzzBudgetPreflightDoesNotPanic(f *testing.F) {
	for _, s := range []string{`<x a="b"/>`, `<!DOCTYPE x [<!-- ] -->]><x/>`, `<x a='>='/>`, `<![CDATA[<>=]]>`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) { _ = preflightFortinetXMLBudget(b) })
}

func TestBudgetResourceProbe(t *testing.T) {
	kind := os.Getenv("FORTINET_BUDGET_KIND")
	if kind == "" {
		t.Skip("explicit resource probe only")
	}
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	var item string
	accepted := false
	switch kind {
	case "maxattrs":
		item = `<ignored ` + strings.Repeat(`a="" `, fortinetBudgetAttributes-1) + `/>`
		accepted = true
	case "maxdomains":
		item = strings.Repeat(`<dns domain="x.test"/>`, fortinetBudgetSemantic-1)
		accepted = true
	case "maxignored":
		item = strings.Repeat(`<x/>`, fortinetBudgetMarkup+1)
		accepted = false
	case "maxvalue":
		item = `<ignored a="` + strings.Repeat("x", fortinetMaximumAuthenticationBody-len(prefix)-len(suffix)-50) + `"/>`
		accepted = true
	case "density":
		item = `<ignored ` + strings.Repeat(`a="" `, 1500000) + `/>`
	case "domains":
		item = `<dns domain="x.test"/>`
	case "ignored":
		item = `<x/>`
	case "attribute":
		item = `<ignored a="` + strings.Repeat("x", 1<<20) + `"/>`
		accepted = true
	case "unique_attributes":
		var b strings.Builder
		b.WriteString(`<ignored `)
		for i := 0; i < 200000; i++ {
			fmt.Fprintf(&b, `a%d="" `, i)
		}
		b.WriteString(`/>`)
		item = b.String()
	default:
		t.Fatal(kind)
	}
	n := (fortinetMaximumAuthenticationBody - len(prefix) - len(suffix)) / len(item)
	if strings.HasPrefix(kind, "max") {
		n = 1
	}
	doc := bytes.Join([][]byte{[]byte(prefix), bytes.Repeat([]byte(item), n), []byte(suffix)}, nil)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
	runtime.ReadMemStats(&after)
	if (err == nil) != accepted {
		t.Fatalf("kind %s error=%v", kind, err)
	}
	fmt.Printf("kind=%s bytes=%d accepted=%v duration=%s alloc=%d heap=%d\n", kind, len(doc), err == nil, time.Since(start), after.TotalAlloc-before.TotalAlloc, after.HeapAlloc)
	runtime.KeepAlive(doc)
	runtime.KeepAlive(cfg)
}

package openconnect

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Candidate-only regression: preserve supported input, rather than raise arbitrary caps.
func TestReleaseCompatibilityCounterexamples(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	for _, c := range []struct{ name, xml string }{
		{"4097_domains", prefix + strings.Repeat(`<dns domain="x.test"/>`, 4097) + suffix},
		{"65536_ignored_attribute", prefix + `<ignored a="` + strings.Repeat("x", 65536) + `"/>` + suffix},
		{"duplicate_domains_preserved", prefix + strings.Repeat(`<dns domain="x.test"/>`, 3) + suffix},
	} {
		t.Run(c.name, func(t *testing.T) {
			a, ea := parseFortinetBaseline([]byte(c.xml), time.Unix(0, 0))
			b, eb := parseFortinetXMLConfiguration([]byte(c.xml), time.Unix(0, 0))
			if ea != nil || eb != nil || !reflect.DeepEqual(a, b) {
				t.Fatalf("baseline=%v streamed=%v equality=%v", ea, eb, reflect.DeepEqual(a, b))
			}
			t.Logf("accepted bytes=%d domains=%d, exact output matches baseline", len(c.xml), len(b.configuration.SearchDomains))
		})
	}
}

// Run separately; this measures Linux process peak, not Android or a hard bound.
func TestReleaseResourceProbe(t *testing.T) {
	kind := os.Getenv("FORTINET_RESOURCE_KIND")
	if kind == "" {
		t.Skip("isolated probe only")
	}
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	var item string
	switch kind {
	case "domains":
		item = `<dns domain="x.test"/>`
	case "ignored":
		item = `<x/>`
	case "attribute_density":
		item = `<ignored ` + strings.Repeat(`a="" `, 1500000) + `/>`
	case "attribute":
		item = `<ignored a="` + strings.Repeat("x", 1<<20) + `"/>`
	default:
		t.Fatal(kind)
	}
	n := (fortinetMaximumAuthenticationBody - len(prefix) - len(suffix)) / len(item)
	doc := []byte(prefix + strings.Repeat(item, n) + suffix)
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	cfg, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	fmt.Printf("kind=%s bytes=%d entries=%d domains=%d heap_alloc=%d heap_sys=%d allocated_delta=%d\n", kind, len(doc), n, len(cfg.configuration.SearchDomains), after.HeapAlloc, after.HeapSys, after.TotalAlloc-before.TotalAlloc)
	runtime.KeepAlive(doc)
	runtime.KeepAlive(cfg)
}

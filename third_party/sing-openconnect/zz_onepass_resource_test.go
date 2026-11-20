package openconnect

import (
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOnePassResourceCompatibility(t *testing.T) {
	mode := os.Getenv("XML_MODE")
	which := os.Getenv("XML_CASE")
	base := `<ipv4><assigned-addr ipv4="192.0.2.8"/></ipv4>`
	body := ""
	switch which {
	case "32764":
		body = strings.Repeat("<a/>", 32764) + base
	case "million":
		body = strings.Repeat("<a/>", 1000000) + base
	case "largeattr":
		body = `<ignored value="` + strings.Repeat("x", (16<<20)-256) + `"/>` + base
	case "denseattrs":
		body = `<ignored ` + strings.Repeat(`a="x" `, 500000) + `/>` + base
	default:
		t.Skip("explicit XML_CASE resource probe only")
	}
	in := []byte(`<sslvpn-tunnel>` + body + `</sslvpn-tunnel>`)
	body = ""
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	parse := parseFortinetXMLConfiguration
	if mode == "baseline" {
		parse = parseFortinetBaseline
	}
	got, err := parse(in, time.Unix(0, 0))
	runtime.ReadMemStats(&after)
	t.Logf("mode=%s case=%s input_bytes=%d alloc_bytes=%d heap_inuse=%d duration=%v err=%v", mode, which, len(in), after.TotalAlloc-before.TotalAlloc, after.HeapInuse, time.Since(start), err)
	if err != nil {
		t.Fatal(err)
	}
	if which == "32764" {
		want, e := parseFortinetBaseline(in, time.Unix(0, 0))
		if e != nil || !reflect.DeepEqual(want, got) {
			t.Fatalf("baseline mismatch: %v", e)
		}
	}
}

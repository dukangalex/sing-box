package openconnect

import (
	"bytes"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFollowupResourceCounterexamples(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	cases := []struct {
		name, middle string
		accept       bool
	}{
		{"prior_4p2mb_65535_domains", strings.Repeat(`<dns domain="`+strings.Repeat("x", 50)+`"/>`, 65535), true},
		{"near_16mib_attribute", `<ignored a="` + strings.Repeat("x", fortinetMaximumAuthenticationBody-len(prefix)-len(suffix)-50) + `"/>`, true},
		{"32763_small_elements", strings.Repeat(`<x/>`, 32763), true},
		{"32764_small_elements", strings.Repeat(`<x/>`, 32764), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := bytes.Join([][]byte{[]byte(prefix), []byte(c.middle), []byte(suffix)}, nil)
			runtime.GC()
			var a, b runtime.MemStats
			runtime.ReadMemStats(&a)
			start := time.Now()
			cfg, err := parseFortinetXMLConfiguration(data, time.Unix(0, 0))
			runtime.ReadMemStats(&b)
			t.Logf("bytes=%d accepted=%t cumulative_alloc=%d duration=%s error=%v", len(data), err == nil, b.TotalAlloc-a.TotalAlloc, time.Since(start), err)
			if (err == nil) != c.accept {
				t.Fatalf("expected accept=%t", c.accept)
			}
			if c.name == "near_16mib_attribute" {
				baseline, baseErr := parseFortinetBaseline(data, time.Unix(0, 0))
				if baseErr != nil || !reflect.DeepEqual(baseline, cfg) {
					t.Fatalf("baseline=%v; results differ=%t", baseErr, !reflect.DeepEqual(baseline, cfg))
				}
			}
			runtime.KeepAlive(cfg)
			runtime.KeepAlive(data)
		})
	}
}

func TestFollowupMarkupCompatibilityCounterexample(t *testing.T) {
	s := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>` + strings.Repeat(`<x/>`, 32764) + `</ipv4></sslvpn-tunnel>`)
	old, oe := parseFortinetBaseline(s, time.Unix(0, 0))
	cur, ce := parseFortinetXMLConfiguration(s, time.Unix(0, 0))
	if oe != nil || old == nil || ce != nil || cur == nil {
		t.Fatalf("old=%v new=%v", oe, ce)
	}
	t.Logf("baseline accepted %d-byte valid XML; v2 candidate also accepted: %v", len(s), ce)
}

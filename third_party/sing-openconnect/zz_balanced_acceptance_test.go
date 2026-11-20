package openconnect

import (
	"bytes"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBalancedRetainedCompatibility(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	for _, middle := range []string{strings.Repeat(`<x/>`, 32764), strings.Repeat(`<dns domain="x.test"/>`, 4097), `<ignored a="` + strings.Repeat("x", 65536) + `"/>`} {
		data := []byte(prefix + middle + suffix)
		a, ae := parseFortinetBaseline(data, time.Unix(0, 0))
		b, be := parseFortinetXMLConfiguration(data, time.Unix(0, 0))
		if ae != nil || be != nil || !reflect.DeepEqual(a, b) {
			t.Fatalf("bytes=%d baseline=%v candidate=%v equal=%t", len(data), ae, be, reflect.DeepEqual(a, b))
		}
		t.Logf("baseline-equivalent %d bytes", len(data))
	}
}
func TestBalancedDenseReject(t *testing.T) {
	data := bytes.Join([][]byte{[]byte(`<sslvpn-tunnel><ignored `), bytes.Repeat([]byte(`a="" `), 500000), []byte(`/></sslvpn-tunnel>`)}, nil)
	runtime.GC()
	var a, b runtime.MemStats
	runtime.ReadMemStats(&a)
	cfg, e := parseFortinetXMLConfiguration(data, time.Unix(0, 0))
	runtime.ReadMemStats(&b)
	if cfg != nil || e == nil || !strings.Contains(e.Error(), "attributes") {
		t.Fatalf("error=%v cfg=%v", e, cfg)
	}
	if b.TotalAlloc-a.TotalAlloc > 1<<20 {
		t.Fatalf("rejection allocated %d", b.TotalAlloc-a.TotalAlloc)
	}
	t.Logf("input=%d reject-allocation=%d error=%v", len(data), b.TotalAlloc-a.TotalAlloc, e)
}

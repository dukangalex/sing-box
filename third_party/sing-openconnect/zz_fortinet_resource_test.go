package openconnect

import (
	"strings"
	"testing"
	"time"
)

// Benchmark one untrusted but well-formed, shallow response. This records resource
// scaling for review; it is not an Android memory test or a safety guarantee.
func BenchmarkFortinetShallowManyNodes(b *testing.B) {
	doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/></ipv4>` + strings.Repeat(`<x/>`, 100000) + `</sslvpn-tunnel>`)
	b.SetBytes(int64(len(doc)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := parseFortinetXMLConfiguration(doc, time.Unix(0, 0)); err != nil {
			b.Fatal(err)
		}
	}
}

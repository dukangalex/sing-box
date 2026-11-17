package openconnect

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Compare same synthetic, shallow response with the original and guarded parser.
// Numbers are cumulative allocations, not live Android heap or peak RSS.
func BenchmarkFortinetShallowManyNodes(b *testing.B) {
	for _, count := range []int{1000, 100000} {
		doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/></ipv4>` + strings.Repeat(`<x/>`, count) + `</sslvpn-tunnel>`)
		for _, candidate := range []bool{false, true} {
			name := "original"
			if candidate {
				name = "guarded"
			}
			b.Run(fmt.Sprintf("nodes_%d/%s", count, name), func(b *testing.B) {
				b.SetBytes(int64(len(doc)))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var err error
					if candidate {
						_, err = parseFortinetXMLConfiguration(doc, time.Unix(0, 0))
					} else {
						_, err = parseFortinetOriginalForComparison(doc, time.Unix(0, 0))
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

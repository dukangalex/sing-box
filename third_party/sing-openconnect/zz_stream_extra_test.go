package openconnect

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStreamMixedPathDifferential(t *testing.T) {
	r := rand.New(rand.NewSource(20261117))
	now := time.Unix(1, 0)
	roots := []string{`<dtls-config heartbeat-interval="13"/>`, `<idle-timeout val="30"/>`, `<auth-timeout val="5"/>`, `<auth-timeout val="0"/>`, `<auth-ses tun-connect-without-reauth="1" check-src-ip="1" tun-user-ses-timeout="8"/>`, `<fos platform="a" minor="2"/>`, `<other><ipv4><dns ip="bad"/></ipv4></other>`, `<!-- hi -->`, `<idle-timeout val="-1"/>`}
	ip4 := []string{`<assigned-addr ipv4="192.0.2.8"/>`, `<dns ip="192.0.2.53" domain="example.test"/>`, `<split-dns domains="a.test,b.test" dnsserver1="192.0.2.54"/>`, `<split-tunnel-info negate="0"><addr ip="198.51.100.3" mask="255.255.255.0"/><addr ip="203.0.113.1" mask="255.255.255.255"/></split-tunnel-info>`, `<split-tunnel-info negate="bad"/>`, `<addr ip="bad"/>`, `<junk><dns ip="bad"/></junk>`}
	ip6 := []string{`<assigned-addr ipv6="2001:db8::1" prefix-len="64"/>`, `<dns ipv6="2001:db8::53"/>`, `<split-tunnel-info negate="1"><addr ipv6="2001:db8:1::" prefix-len="48"/></split-tunnel-info>`, `<junk/>`}
	for i := 0; i < 1000; i++ {
		var b strings.Builder
		b.WriteString(`<sslvpn-tunnel dtls="1">`)
		for j := 0; j < r.Intn(16); j++ {
			switch r.Intn(3) {
			case 0:
				b.WriteString(roots[r.Intn(len(roots))])
			case 1:
				b.WriteString(`<ipv4>`)
				for k := 0; k < r.Intn(8); k++ {
					b.WriteString(ip4[r.Intn(len(ip4))])
				}
				b.WriteString(`</ipv4>`)
			case 2:
				b.WriteString(`<ipv6>`)
				for k := 0; k < r.Intn(6); k++ {
					b.WriteString(ip6[r.Intn(len(ip6))])
				}
				b.WriteString(`</ipv6>`)
			}
		}
		b.WriteString(`</sslvpn-tunnel>`)
		a, ea := parseFortinetBaseline([]byte(b.String()), now)
		x, ex := parseFortinetXMLConfiguration([]byte(b.String()), now)
		if (ea == nil) != (ex == nil) || ea == nil && !reflect.DeepEqual(a, x) {
			t.Fatal(fmt.Sprintf("case %d baseline %v (%+v) streamed %v (%+v) xml %s", i, ea, a, ex, x, b.String()))
		}
	}
	t.Log("1000 mixed root/IP ordered cases matched success/error and successful values")
}

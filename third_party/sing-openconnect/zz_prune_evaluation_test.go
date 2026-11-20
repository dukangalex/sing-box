package openconnect

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPruneDifferential(t *testing.T) {
	now := time.Unix(0, 0)
	corpus := []string{
		`<sslvpn-tunnel dtls="1"><dtls-config heartbeat-interval="5"/><fos platform="Forti" major="7"/><ipv4><assigned-addr ipv4="192.0.2.8"/><dns ip="192.0.2.53" domain="example.org"/><split-tunnel-info><addr ip="198.51.100.0" mask="255.255.255.0"/></split-tunnel-info></ipv4><idle-timeout val="30"/></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv6><assigned-addr ipv6="2001:db8::1" prefix-len="64"/><dns ipv6="2001:db8::53"/><split-tunnel-info negate="1"><addr ipv6="2001:db8:1::" prefix-len="48"/></split-tunnel-info></ipv6></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/><split-dns domains="example.org,example.net" dnsserver1="192.0.2.53"/></ipv4><auth-ses tun-connect-without-reauth="1" check-src-ip="1" tun-user-ses-timeout="5"/></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="bad"/></ipv4></sslvpn-tunnel>`, `<sslvpn-tunnel><ipv4></sslvpn-tunnel>`, `<sslvpn-tunnel/><junk/>`,
	}
	r := rand.New(rand.NewSource(741))
	options := []string{`<junk a="abc"><ipv4><assigned-addr ipv4="10.1.1.1"/></ipv4></junk>`, `<dns ip="192.0.2.3"/>`, `<assigned-addr ipv4="192.0.2.8"/>`, `<split-tunnel-info><addr ip="192.0.2.1" mask="255.255.255.255"/><unknown><addr/></unknown></split-tunnel-info>`, `<dns ip="bad"/>`, `text`, `<!-- comment -->`}
	for i := 0; i < 500; i++ {
		var b strings.Builder
		b.WriteString(`<sslvpn-tunnel><ipv4>`)
		for j := 0; j < r.Intn(20); j++ {
			b.WriteString(options[r.Intn(len(options))])
		}
		b.WriteString(`</ipv4></sslvpn-tunnel>`)
		corpus = append(corpus, b.String())
	}
	for i, s := range corpus {
		a, ea := parseFortinetBaseline([]byte(s), now)
		b, eb := parseFortinetXMLConfiguration([]byte(s), now)
		if (ea == nil) != (eb == nil) || ea == nil && !reflect.DeepEqual(a, b) {
			t.Fatalf("case %d: baseline=%+v err=%v new=%+v err=%v input=%s", i, a, ea, b, eb, s)
		}
	}
	t.Logf("%d cases matched success/error and successful config", len(corpus))
}
func BenchmarkPrune(b *testing.B) {
	for _, kind := range []string{"unknown", "known"} {
		var x string
		if kind == "unknown" {
			x = `<extension x="123"/>`
		} else {
			x = `<dns ip="192.0.2.53"/>`
		}
		doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.8"/>` + strings.Repeat(x, 100000) + `</ipv4></sslvpn-tunnel>`)
		for _, fn := range []struct {
			name string
			f    func([]byte, time.Time) (*fortinetTunnelConfiguration, error)
		}{{"baseline", parseFortinetBaseline}, {"pruned", parseFortinetXMLConfiguration}} {
			b.Run(fmt.Sprintf("%s/%s", kind, fn.name), func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(doc)))
				for i := 0; i < b.N; i++ {
					_, err := fn.f(doc, time.Unix(0, 0))
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

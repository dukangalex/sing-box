package openconnect

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func compareReaudit(t *testing.T, s string) {
	t.Helper()
	if len(s) > 1000000 {
		t.Skip()
	}
	now := time.Unix(1700000000, 0)
	a, ea := parseFortinetBaseline([]byte(s), now)
	b, eb := parseFortinetXMLConfiguration([]byte(s), now)
	if (ea == nil) != (eb == nil) || ea == nil && !reflect.DeepEqual(a, b) {
		t.Fatalf("behavior divergence input=%q baseline=%+v err=%v streaming=%+v err=%v", s, a, ea, b, eb)
	}
}
func TestReauditMalformed(t *testing.T) {
	base := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/></ipv4></sslvpn-tunnel>`
	cases := []string{"", "  ", base, base + `<x/>`, base + `<!-- ok -->`, base + `trailing text`, base + ` &bad;`, `\xef\xbb\xbf` + base,
		`<?xml version="1.0"?>` + base, `<!DOCTYPE sslvpn-tunnel [<!ENTITY x "x">]>` + base,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/></ipv4>`, `<sslvpn-tunnel><ipv4></sslvpn-tunnel>`,
		`<ns:sslvpn-tunnel xmlns:ns="urn:test"><ns:ipv4><ns:assigned-addr ns:ipv4="192.0.2.1"/></ns:ipv4></ns:sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/><split-tunnel-info negate="1"><unknown/><addr ip="198.51.100.0" mask="255.255.255.0"/></split-tunnel-info></ipv4></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/><split-tunnel-info negate="1"><other><split-tunnel-info><addr ip="bad"/></split-tunnel-info></other><addr ip="198.51.100.0" mask="255.255.255.0"/></split-tunnel-info></ipv4></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/><dns domain="a&amp;b"/></ipv4></sslvpn-tunnel>`,
		`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/><dns ip="&#x0;"/></ipv4></sslvpn-tunnel>`,
		strings.Repeat(" ", 1024) + base + strings.Repeat(" ", 1024),
	}
	for _, s := range cases {
		compareReaudit(t, s)
	}
	t.Logf("%d malformed, nested, trailing and namespace cases matched success/result", len(cases))
}
func TestIllustrativeFortinetShape(t *testing.T) {
	// Illustrative structure; source extraction does not establish a complete real 2019 XML response.
	s := `<?xml version='1.0' encoding='utf-8'?><sslvpn-tunnel ver='2' dtls='1' patch='1'><dtls-config heartbeat-interval='10' heartbeat-fail-count='10' heartbeat-idle-timeout='10' client-hello-timeout='10' /><tunnel-method value='ppp' /><tunnel-method value='tun' /><fos platform='FG100E' major='5' minor='06' patch='6' build='1630' branch='1630' /><client-config save-password='off' keep-alive='on' auto-connect='off' /><ipv4><assigned-addr ipv4='172.16.1.1' /><split-tunnel-info><addr ip='10.11.10.10' mask='255.255.255.255' /><addr ip='10.11.1.0' mask='255.255.255.0' /></split-tunnel-info></ipv4><idle-timeout val='3600' /><auth-timeout val='18000' /></sslvpn-tunnel>`
	now := time.Unix(1700000000, 0)
	a, ea := parseFortinetBaseline([]byte(s), now)
	b, eb := parseFortinetXMLConfiguration([]byte(s), now)
	if ea != nil || eb != nil || !reflect.DeepEqual(a, b) {
		t.Fatalf("baseline=%v streaming=%v equal=%t", ea, eb, reflect.DeepEqual(a, b))
	}
	t.Logf("illustrative Fortinet shape parsed equivalently, ipv4=%t routes=%d", b.wantIPv4, len(b.configuration.Routes))
}
func FuzzReauditDifferential(f *testing.F) {
	for _, s := range []string{`<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/></ipv4></sslvpn-tunnel>`, `<sslvpn-tunnel dtls="1"><ipv4><assigned-addr ipv4="192.0.2.1"/><dns ip="192.0.2.2"/><split-tunnel-info><addr ip="192.0.2.0" mask="255.255.255.0"/></split-tunnel-info></ipv4></sslvpn-tunnel>`, `<sslvpn-tunnel/>`, `<?xml version="1.0"?><x/>`, `<x></y>`, `<!DOCTYPE x><x a="&amp;"/>`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) { compareReaudit(t, s) })
}

func TestForumPostedXML(t *testing.T) {
	// Public Fortinet forum, 2024-08-26: author says working setup; DNS values are placeholders.
	// https://community.fortinet.com/t5/Support-Forum/JSON-for-deploying-a-Managed-configuration-Forticlient-VPN-to/td-p/335908
	s := `<?xml version='1.0' encoding='utf-8'?><sslvpn-tunnel ver='2' dtls='1' patch='1'><dtls-config ver='2' heartbeat-interval='10' heartbeat-fail-count='10' heartbeat-idle-timeout='10' client-hello-timeout='10' /><tunnel-method value='ppp' /><tunnel-method value='tun' /><tunnel-method value='websocket' /><auth-ses check-src-ip='1' tun-connect-without-reauth='0' tun-user-ses-timeout='30' /><client-config save-password='on' keep-alive='on' auto-connect='on' /><ipv4><split-dns domains='domain-one.com,domain-two.dk' dnsserver1='ip-of-dnsserver1' dnsserver2='ip-of-dnsserver2' /><dns domain='domain-one.com;domain-two.dk' /><dns ip='ip-of-dnsserver1' /><dns ip='ip-of-dnsserver2' /><assigned-addr ipv4='10.10.2.151' /><split-tunnel-info><addr ip='10.0.0.0' mask='255.0.0.0' /><addr ip='172.16.0.0' mask='255.240.0.0' /><addr ip='192.168.0.0' mask='255.255.0.0' /><addr ip='185.17.194.50' mask='255.255.255.255' /></split-tunnel-info></ipv4><idle-timeout val='0' /><auth-timeout val='28800' /></sslvpn-tunnel>`
	now := time.Unix(1700000000, 0)
	_, a := parseFortinetBaseline([]byte(s), now)
	_, b := parseFortinetXMLConfiguration([]byte(s), now)
	if a == nil || b == nil {
		t.Fatalf("expected placeholder DNS rejection: baseline=%v stream=%v", a, b)
	}
	s = strings.ReplaceAll(s, "ip-of-dnsserver1", "192.0.2.53")
	s = strings.ReplaceAll(s, "ip-of-dnsserver2", "192.0.2.54")
	old, ea := parseFortinetBaseline([]byte(s), now)
	cur, eb := parseFortinetXMLConfiguration([]byte(s), now)
	if ea != nil || eb != nil || !reflect.DeepEqual(old, cur) {
		t.Fatalf("adapted fixture: baseline=%v stream=%v equal=%t", ea, eb, reflect.DeepEqual(old, cur))
	}
	t.Logf("source fixture both reject placeholder IPs; replacing them with TEST-NET IPs yields matching results: routes=%d", len(cur.configuration.Routes))
}

func BenchmarkReauditResourceBoundaries(b *testing.B) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4="192.0.2.1"/>`
	suffix := `</ipv4></sslvpn-tunnel>`
	cases := map[string]string{
		"large_ignored_text_4MiB":   prefix + `<unknown>` + strings.Repeat("x", 4*1024*1024) + `</unknown>` + suffix,
		"many_output_domains_20000": prefix + strings.Repeat(`<dns domain="example.test"/>`, 20000) + suffix,
		"large_attribute_4MiB":      prefix + `<unknown a="` + strings.Repeat("x", 4*1024*1024) + `"/>` + suffix,
	}
	for name, s := range cases {
		for _, p := range []struct {
			name string
			fn   func([]byte, time.Time) (*fortinetTunnelConfiguration, error)
		}{{"old", parseFortinetBaseline}, {"stream", parseFortinetXMLConfiguration}} {
			b.Run(name+"/"+p.name, func(b *testing.B) {
				data := []byte(s)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_, err := p.fn(data, time.Unix(0, 0))
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

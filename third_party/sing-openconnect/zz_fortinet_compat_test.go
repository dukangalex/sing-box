package openconnect

import (
	"net/netip"
	"strings"
	"testing"
	"time"
)

// Response adapted from public Fortinet community examples; documentation IPs substituted.
// https://community.fortinet.com/t5/Support-Forum/Forticlient-Connected-but-no-bytes-sent-or-received/m-p/88779?m=170510
// https://community.fortinet.com/support-forum-92/json-for-deploying-a-managed-configuration-forticlient-vpn-to-chrome-devices-in-google-workspace-179725
func TestFortinetRealWorldShape(t *testing.T) {
	const fixture = `<?xml version='1.0' encoding='utf-8'?><sslvpn-tunnel ver='2' dtls='1' patch='1'><dtls-config heartbeat-interval='10' heartbeat-fail-count='10' heartbeat-idle-timeout='10' client-hello-timeout='10'/><tunnel-method value='ppp'/><tunnel-method value='tun'/><fos platform='FG100E' major='5' minor='06' patch='6' build='1630' branch='1630'/><client-config save-password='off' keep-alive='on' auto-connect='off'/><ipv4><assigned-addr ipv4='192.0.2.8'/><dns ip='192.0.2.53'/><split-tunnel-info><addr ip='198.51.100.0' mask='255.255.255.0'/><addr ip='203.0.113.10' mask='255.255.255.255'/></split-tunnel-info></ipv4><idle-timeout val='3600'/><auth-timeout val='18000'/></sslvpn-tunnel>`
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := parseFortinetXMLConfiguration([]byte(fixture), now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.wantIPv4 || got.proposedIPv4 != netip.MustParsePrefix("192.0.2.8/32") || !got.dtlsEnabled || got.echoInterval != 10*time.Second {
		t.Fatalf("unexpected settings: %+v", got)
	}
	if len(got.configuration.DNS) != 1 || got.configuration.DNS[0] != netip.MustParseAddr("192.0.2.53") {
		t.Fatalf("DNS: %v", got.configuration.DNS)
	}
	if len(got.configuration.Routes) != 2 || got.configuration.Routes[0].Prefix != netip.MustParsePrefix("198.51.100.0/24") || got.configuration.Routes[1].Prefix != netip.MustParsePrefix("203.0.113.10/32") {
		t.Fatalf("routes: %v", got.configuration.Routes)
	}
	if got.configuration.IdleTimeout != time.Hour || !got.configuration.AuthenticationExpiration.Equal(now.Add(5*time.Hour)) {
		t.Fatalf("timeouts: %+v", got.configuration)
	}
}
func TestFortinetFullDocumentDepth(t *testing.T) {
	prefix := `<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><extension>`
	suffix := `</extension></sslvpn-tunnel>`
	for _, tt := range []struct {
		nested int
		fail   bool
	}{{254, false}, {255, true}, {9998, true}} {
		// root + extension = 2; depth 256 passes, depth 257 fails.
		doc := prefix + strings.Repeat("<x>", tt.nested) + strings.Repeat("</x>", tt.nested) + suffix
		_, err := parseFortinetXMLConfiguration([]byte(doc), time.Now())
		if (err != nil) != tt.fail {
			t.Fatalf("nested %d: %v", tt.nested, err)
		}
		if tt.fail && !strings.Contains(err.Error(), "exceeds nesting limit") {
			t.Fatalf("wrong failure: %v", err)
		}
	}
}

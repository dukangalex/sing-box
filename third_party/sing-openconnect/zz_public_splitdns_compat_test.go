package openconnect

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

// Adapted from a Fortinet community member's posted working example, with
// placeholder DNS values replaced by documentation IPs and addresses anonymized.
// https://community.fortinet.com/t5/Support-Forum/JSON-for-deploying-a-Managed-configuration-Forticlient-VPN-to/m-p/390649
// This is one public example, not evidence of a live server's present behavior.
func TestPublicFortinetSplitDNSFixture(t *testing.T) {
	const document = `<?xml version='1.0' encoding='utf-8'?><sslvpn-tunnel ver='2' dtls='1' patch='1'><dtls-config ver='2' heartbeat-interval='10' heartbeat-fail-count='10' heartbeat-idle-timeout='10' client-hello-timeout='10' /><tunnel-method value='ppp' /><tunnel-method value='tun' /><tunnel-method value='websocket' /><auth-ses check-src-ip='1' tun-connect-without-reauth='0' tun-user-ses-timeout='30' /><client-config save-password='on' keep-alive='on' auto-connect='on' /><ipv4><split-dns domains='domain-one.com,domain-two.dk' dnsserver1='192.0.2.53' dnsserver2='192.0.2.54' /><dns domain='domain-one.com;domain-two.dk' /><dns ip='192.0.2.53' /><dns ip='192.0.2.54' /><assigned-addr ipv4='192.0.2.151' /><split-tunnel-info><addr ip='10.0.0.0' mask='255.0.0.0' /><addr ip='172.16.0.0' mask='255.240.0.0' /><addr ip='192.168.0.0' mask='255.255.0.0' /><addr ip='203.0.113.50' mask='255.255.255.255' /></split-tunnel-info></ipv4><idle-timeout val='0' /><auth-timeout val='28800' /></sslvpn-tunnel>`
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base, e1 := parseFortinetBaseline([]byte(document), now)
	got, e2 := parseFortinetXMLConfiguration([]byte(document), now)
	if e1 != nil || e2 != nil {
		t.Fatalf("baseline=%v candidate=%v", e1, e2)
	}
	if !reflect.DeepEqual(base, got) {
		t.Fatalf("candidate differs from baseline: baseline=%+v candidate=%+v", base, got)
	}
	t.Logf("public example adapted, bytes=%d exact baseline equivalence", len(document))
}

// Explicit regression demonstrating a syntactically legal input that used to
// parse and is rejected by experimental policy, not a claim Fortinet sends it.
func TestPreviouslyRejectedOneMiBAttributeAccepted(t *testing.T) {
	doc := []byte(`<sslvpn-tunnel><ipv4><assigned-addr ipv4='192.0.2.8'/></ipv4><ignored note='` + strings.Repeat("x", (1<<20)+1) + `'/></sslvpn-tunnel>`)
	now := time.Unix(0, 0)
	baseline, baseErr := parseFortinetBaseline(doc, now)
	candidate, candidateErr := parseFortinetXMLConfiguration(doc, now)
	if baseErr != nil || baseline == nil {
		t.Fatalf("baseline unexpectedly rejected: %v", baseErr)
	}
	if candidateErr != nil || candidate == nil || !reflect.DeepEqual(baseline, candidate) {
		t.Fatalf("candidate mismatch: %v", candidateErr)
	}
	t.Logf("bytes=%d baseline and candidate accepted with identical output", len(doc))
}

package openconnect

import (
	"bytes"
	"encoding/xml"
	"fmt"
	E "github.com/sagernet/sing/common/exceptions"
	"io"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

func parseFortinetOriginalForComparison(content []byte, now time.Time) (*fortinetTunnelConfiguration, error) {
	decoder := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace(content)))
	var root fortinetXMLNode
	decodeErr := decoder.Decode(&root)
	if decodeErr != nil {
		return nil, E.Cause(decodeErr, "decode Fortinet VPN configuration XML")
	}
	var trailing fortinetXMLNode
	trailingErr := decoder.Decode(&trailing)
	if trailingErr == nil {
		return nil, E.New("VPN configuration contains trailing XML data")
	}
	if trailingErr != io.EOF {
		return nil, E.Cause(trailingErr, "VPN configuration contains trailing XML data")
	}
	if root.name.Local != "sslvpn-tunnel" {
		return nil, E.New("VPN configuration has no sslvpn-tunnel root")
	}
	configuration := &fortinetTunnelConfiguration{
		configuration: TunnelConfiguration{MTU: pppDefaultTunnelMTU},
	}
	if dtlsValue, loaded := root.attribute("dtls"); loaded {
		dtlsEnabled, parseErr := parseFortinetBoolean(dtlsValue)
		if parseErr != nil {
			return nil, E.Cause(parseErr, "parse Fortinet DTLS enablement")
		}
		configuration.dtlsEnabled = dtlsEnabled
	}
	var ipv4IncludeRoutes int
	for childIndex := range root.children {
		child := &root.children[childIndex]
		switch child.name.Local {
		case "dtls-config":
			intervalValue, loaded := child.attribute("heartbeat-interval")
			if loaded {
				interval, parseErr := parseFortinetDurationSeconds(intervalValue)
				if parseErr != nil {
					return nil, E.Cause(parseErr, "parse Fortinet heartbeat interval")
				}
				configuration.echoInterval = interval
			}
		case "idle-timeout":
			value, loaded := child.attribute("val")
			if loaded {
				idleTimeout, parseErr := parseFortinetDurationSeconds(value)
				if parseErr != nil {
					return nil, E.Cause(parseErr, "parse Fortinet idle timeout")
				}
				configuration.configuration.IdleTimeout = idleTimeout
			}
		case "auth-timeout":
			value, loaded := child.attribute("val")
			if loaded {
				authenticationTimeout, parseErr := parseFortinetDurationSeconds(value)
				if parseErr != nil {
					return nil, E.Cause(parseErr, "parse Fortinet authentication timeout")
				}
				if authenticationTimeout > 0 {
					configuration.configuration.AuthenticationExpiration = now.Add(authenticationTimeout)
				}
			}
		case "auth-ses":
			parseErr := parseFortinetReconnectPolicy(child, configuration)
			if parseErr != nil {
				return nil, parseErr
			}
		case "fos":
			configuration.platform = formatFortinetPlatform(child)
		case "ipv4":
			includeCount, parseErr := parseFortinetIPSection(child, false, configuration)
			if parseErr != nil {
				return nil, parseErr
			}
			ipv4IncludeRoutes += includeCount
		case "ipv6":
			_, parseErr := parseFortinetIPSection(child, true, configuration)
			if parseErr != nil {
				return nil, parseErr
			}
		}
	}
	if configuration.wantIPv4 && ipv4IncludeRoutes == 0 {
		configuration.configuration.Routes = append(configuration.configuration.Routes, TunnelRoute{
			Prefix: netip.PrefixFrom(netip.IPv4Unspecified(), 0),
		})
	}
	if !configuration.wantIPv4 && !configuration.wantIPv6 {
		return nil, E.New("VPN configuration enables no usable network family")
	}
	return configuration, nil
}

func TestDepthGuardPreservesSupportedInputs(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, root := range []string{"sslvpn-tunnel", "f:sslvpn-tunnel"} {
		ns := ""
		if strings.HasPrefix(root, "f:") {
			ns = ` xmlns:f="urn:example:test"`
		}
		for _, children := range []string{
			`<ipv4><assigned-addr ipv4="192.0.2.8"/><dns ip="192.0.2.53"/></ipv4>`,
			`<ipv4><assigned-addr ipv4="192.0.2.8"/><split-tunnel-info><addr ip="198.51.100.0" mask="255.255.255.0"/></split-tunnel-info></ipv4><dtls-config heartbeat-interval="10"/>`,
		} {
			for _, extra := range []string{"", "<!--comment-->", "<unknown><![CDATA[text]]></unknown>", "<unknown>text&amp;more</unknown>"} {
				for _, depth := range []int{0, 1, 32, 254} {
					nested := strings.Repeat("<z>", depth) + strings.Repeat("</z>", depth)
					input := []byte(fmt.Sprintf("<?xml version=\"1.0\"?><%s%s>%s%s%s</%s>", root, ns, children, extra, nested, root))
					old, oldErr := parseFortinetOriginalForComparison(input, now)
					got, gotErr := parseFortinetXMLConfiguration(input, now)
					if (oldErr == nil) != (gotErr == nil) || (oldErr == nil && !reflect.DeepEqual(old, got)) {
						t.Fatalf("different behavior root=%s depth=%d old=%v new=%v", root, depth, oldErr, gotErr)
					}
				}
			}
		}
	}
}

package openconnect

import (
	"bytes"
	"encoding/xml"
	E "github.com/sagernet/sing/common/exceptions"
	"io"
	"net/netip"
	"time"
)

func parseFortinetBaseline(content []byte, now time.Time) (*fortinetTunnelConfiguration, error) {
	if depthErr := validateFortinetDepth(content); depthErr != nil {
		return nil, E.Cause(depthErr, "decode Fortinet VPN configuration XML")
	}
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

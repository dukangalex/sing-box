package naive

import (
	"net/http"
	"strings"
	"testing"
)

func TestBuildSafeExtraHeaders(t *testing.T) {
	for _, name := range []string{"-connect-authority", "-Connect-Authority", "-CONNECT-AUTHORITY", "-force-quic", "-Force-Quic", "-network-isolation-key", "-Network-Isolation-Key"} {
		t.Run(name, func(t *testing.T) {
			_, err := buildSafeExtraHeaders(http.Header{name: []string{"bad"}}, false)
			if err == nil {
				t.Fatalf("reserved %q accepted", name)
			}
		})
	}
	for _, name := range []string{"Proxy-Authorization", "proxy-authorization", "PROXY-AUTHORIZATION"} {
		_, err := buildSafeExtraHeaders(http.Header{name: []string{"custom"}}, true)
		if err == nil {
			t.Fatalf("conflicting credentials %q accepted", name)
		}
		got, err := buildSafeExtraHeaders(http.Header{name: []string{"custom"}}, false)
		if err != nil || got[name] != "custom" {
			t.Fatalf("custom authorization %q: %v, %v", name, got, err)
		}
	}
	got, err := buildSafeExtraHeaders(http.Header{"X-Example": []string{"first", "second"}, "Empty": nil}, true)
	if err != nil || got["X-Example"] != "first" || len(got) != 1 {
		t.Fatalf("ordinary headers: %#v %v", got, err)
	}
	_, err = buildSafeExtraHeaders(http.Header{"-Force-Quic": []string{"SECRET"}}, false)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error leaks value: %v", err)
	}
}

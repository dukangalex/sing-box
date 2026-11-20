package naive

import (
	"fmt"
	"net/http"
	"strings"
)

// buildSafeExtraHeaders keeps user headers away from Cronet's internal
// CONNECT controls. Proxy authorization is reserved only when credentials
// were supplied separately, so custom schemes remain available otherwise.
func buildSafeExtraHeaders(headers http.Header, hasCredentials bool) (map[string]string, error) {
	result := make(map[string]string, len(headers))
	for name, values := range headers {
		if len(values) == 0 {
			continue
		}
		switch strings.ToLower(name) {
		case "-connect-authority", "-force-quic", "-network-isolation-key":
			return nil, fmt.Errorf("naive extra_headers contains reserved header %q", name)
		case "proxy-authorization":
			if hasCredentials {
				return nil, fmt.Errorf("naive extra_headers conflicts with configured proxy credentials")
			}
		}
		result[name] = values[0]
	}
	return result, nil
}

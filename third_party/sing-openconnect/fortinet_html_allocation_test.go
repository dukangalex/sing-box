package openconnect

import (
	"runtime"
	"strings"
	"testing"
)

// GO-2026-6860: ignored end tags previously made html.Parse repeatedly copy a
// coalesced text node (quadratic allocation). Exercise our real auth entry point.
func TestFortinetHTMLChallengeCoalescingAllocation(t *testing.T) {
	doc := []byte(`<form method="POST" action="/remote/logincheck"><input name="username" type="hidden"><input name="credential" type="password">` + strings.Repeat("a</x>", 16384) + `</form>`)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	f, err := parseFortinetHTMLChallenge(doc)
	runtime.ReadMemStats(&after)
	if err != nil || f == nil || f.action != "/remote/logincheck" {
		t.Fatalf("parse form: %v", err)
	}
	alloc := after.TotalAlloc - before.TotalAlloc
	// Fixed implementation uses linear buffers. Old x/net v0.57 copies >128MiB
	// for this <100KiB input; keep ample headroom for platform/runtime variation.
	if alloc > 16<<20 {
		t.Fatalf("quadratic HTML coalescing regression: allocated %d bytes for %d input bytes", alloc, len(doc))
	}
	t.Logf("input=%d total_alloc=%d fields=%d", len(doc), alloc, len(f.fields))
}

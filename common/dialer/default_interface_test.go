package dialer

import (
	"testing"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/control"
)

func iface(name string, index int) adapter.NetworkInterface {
	return adapter.NetworkInterface{Interface: control.Interface{Name: name, Index: index}}
}

func TestMatchDefaultInterfaceUsesLiveListWhenIndexIsGone(t *testing.T) {
	live := []adapter.NetworkInterface{iface("wlan0", 24), iface("rmnet0", 30)}
	stale := &control.Interface{Name: "wlan0", Index: 12}
	got := matchDefaultInterface(live, stale)
	if len(got) != 2 {
		t.Fatalf("stale index must fall back to live interfaces, got %d", len(got))
	}
	current := &control.Interface{Name: "wlan0", Index: 24}
	got = matchDefaultInterface(live, current)
	if len(got) != 1 || got[0].Index != 24 {
		t.Fatalf("matching index must stay bound, got %d", len(got))
	}
	if got = matchDefaultInterface(live, nil); len(got) != 2 {
		t.Fatal("nil default keeps the list")
	}
}

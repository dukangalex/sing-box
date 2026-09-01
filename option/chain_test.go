package option

import "testing"

func TestChainOptionsDisabled(t *testing.T) {
	o := &ChainOptions{Mode: ChainModeDisabled, Landing: "us-vps"}
	if o.IsEnabled() {
		t.Fatal("disabled chain must not be enabled")
	}
	if o.AppliesToGroup("Al") {
		t.Fatal("disabled chain must not apply")
	}
}

func TestChainOptionsGlobal(t *testing.T) {
	o := &ChainOptions{Mode: ChainModeGlobal, Landing: "us-vps"}
	if !o.IsEnabled() {
		t.Fatal("global chain should be enabled")
	}
	if !o.AppliesToGroup("Al") || !o.AppliesToGroup("Apple") {
		t.Fatal("global chain should apply to every group")
	}
}

func TestChainOptionsLocal(t *testing.T) {
	o := &ChainOptions{Mode: ChainModeLocal, Landing: "us-vps", Groups: []string{"Al", "AI"}}
	if !o.AppliesToGroup("Al") || !o.AppliesToGroup("AI") {
		t.Fatal("local chain should apply to configured groups")
	}
	if o.AppliesToGroup("Apple") {
		t.Fatal("local chain must not apply to unconfigured groups")
	}
}

func TestChainOptionsRequiresLanding(t *testing.T) {
	o := &ChainOptions{Mode: ChainModeGlobal}
	if o.IsEnabled() {
		t.Fatal("chain without landing must not be enabled")
	}
}

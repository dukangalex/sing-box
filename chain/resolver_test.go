package chain

import "testing"

func TestResolvePlanPreservesDisabledPlan(t *testing.T) {
	p := Plan{Mode: ModeDisabled}
	got, err := ResolvePlan(p, "Proxy", nil)
	if err != nil {
		t.Fatalf("disabled plan rejected: %v", err)
	}
	if got.Mode != ModeDisabled {
		t.Fatalf("mode changed: %v", got.Mode)
	}
}

func TestResolvePlanUsesCurrentStrategyResult(t *testing.T) {
	p := Plan{Landing: "us-vps", Mode: ModeLocal}
	got, err := ResolvePlan(p, "Auto", StaticResolver{Entry: "jp"})
	if err != nil {
		t.Fatalf("plan rejected: %v", err)
	}
	if got.Entry != "jp" {
		t.Fatalf("entry = %q, want jp", got.Entry)
	}
	if got.Landing != "us-vps" {
		t.Fatalf("landing changed: %q", got.Landing)
	}
}

func TestResolvePlanRejectsNilResolver(t *testing.T) {
	p := Plan{Landing: "us-vps", Mode: ModeLocal}
	if _, err := ResolvePlan(p, "Auto", nil); err == nil {
		t.Fatal("expected nil resolver error")
	}
}

func TestResolvePlanRejectsRecursiveResult(t *testing.T) {
	p := Plan{Landing: "Auto", Mode: ModeLocal}
	if _, err := ResolvePlan(p, "Auto", StaticResolver{Entry: "Auto"}); err == nil {
		t.Fatal("expected entry/landing recursion error")
	}
}

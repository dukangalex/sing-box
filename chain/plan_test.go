package chain

import "testing"

func TestPlanValidateDisabled(t *testing.T) {
	if err := (Plan{Mode: ModeDisabled}).Validate(); err != nil {
		t.Fatalf("disabled plan must be valid: %v", err)
	}
}

func TestPlanValidateRequiresEntryAndLanding(t *testing.T) {
	cases := []Plan{
		{Mode: ModeLocal, Landing: "us-vps"},
		{Mode: ModeGlobal, Entry: "auto"},
	}
	for _, p := range cases {
		if err := p.Validate(); err == nil {
			t.Fatalf("expected validation error for %+v", p)
		}
	}
}

func TestPlanValidateRejectsSameEntryAndLanding(t *testing.T) {
	p := Plan{Entry: "auto", Landing: "auto", Mode: ModeLocal}
	if err := p.Validate(); err == nil {
		t.Fatal("expected recursive/self landing validation error")
	}
}

func TestPlanValidateAcceptsStrictPath(t *testing.T) {
	p := Plan{Entry: "auto", Landing: "us-vps", Mode: ModeLocal}
	if err := p.Validate(); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

package chain

import "testing"

func TestPolicyGlobal(t *testing.T) {
	p, err := NewPolicy(ModeGlobal, "us-vps", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Applies("Al") || !p.Applies("Apple") {
		t.Fatal("global policy should apply to every group")
	}
}

func TestPolicyLocal(t *testing.T) {
	p, err := NewPolicy(ModeLocal, "us-vps", []string{"Al", "AI"})
	if err != nil {
		t.Fatal(err)
	}
	if !p.Applies("Al") || p.Applies("Apple") {
		t.Fatal("local policy group matching is incorrect")
	}
}

func TestPolicyValidation(t *testing.T) {
	cases := []struct {
		name   string
		mode   Mode
		landing string
		groups []string
	}{
		{"global without landing", ModeGlobal, "", nil},
		{"local without groups", ModeLocal, "us-vps", nil},
		{"local empty group", ModeLocal, "us-vps", []string{""}},
		{"unknown mode", Mode("invalid"), "us-vps", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewPolicy(tc.mode, tc.landing, tc.groups); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

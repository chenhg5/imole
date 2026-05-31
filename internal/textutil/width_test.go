package textutil

import "testing"

func TestDisplayWidth(t *testing.T) {
	if got := DisplayWidth("微信"); got != 4 {
		t.Fatalf("DisplayWidth(微信) = %d, want 4", got)
	}
	if got := DisplayWidth("Lark"); got != 4 {
		t.Fatalf("DisplayWidth(Lark) = %d, want 4", got)
	}
}

func TestPadRight(t *testing.T) {
	got := PadRight("微信", 6)
	if DisplayWidth(got) != 6 {
		t.Fatalf("width = %d, want 6", DisplayWidth(got))
	}
}

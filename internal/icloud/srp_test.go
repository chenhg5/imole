package icloud

import (
	"testing"
)

func TestNewSRPClient(t *testing.T) {
	c, err := NewSRPClient()
	if err != nil {
		t.Fatal(err)
	}
	if c.A.Sign() <= 0 {
		t.Error("A should be positive")
	}
	// A must be < N
	if c.A.Cmp(srpN) >= 0 {
		t.Error("A must be < N")
	}
	pub := c.PublicB64()
	if pub == "" {
		t.Error("PublicB64 should not be empty")
	}
}

func TestSRPNAndGInitialised(t *testing.T) {
	if srpN == nil || srpN.Sign() == 0 {
		t.Error("srpN not initialised")
	}
	if srpK == nil || srpK.Sign() == 0 {
		t.Error("srpK not initialised")
	}
}

func TestPadToN(t *testing.T) {
	small := srpG // g = 2, single byte
	padded := padToN(small)
	if len(padded) != len(srpN.Bytes()) {
		t.Errorf("padToN: got len %d, want %d", len(padded), len(srpN.Bytes()))
	}
	if padded[len(padded)-1] != 2 {
		t.Error("last byte should be 2 (value of g)")
	}
}

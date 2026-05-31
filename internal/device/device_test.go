package device

import "testing"

func TestParseDiskUsage(t *testing.T) {
	storage, err := parseDiskUsage(`AmountDataAvailable: 250
TotalDataCapacity: 1000
`)
	if err != nil {
		t.Fatal(err)
	}
	if storage.TotalDataCapacity != 1000 {
		t.Fatalf("total = %d, want 1000", storage.TotalDataCapacity)
	}
	if storage.AmountDataAvailable != 250 {
		t.Fatalf("available = %d, want 250", storage.AmountDataAvailable)
	}
	if storage.UsedData != 750 {
		t.Fatalf("used = %d, want 750", storage.UsedData)
	}
	if storage.FreePercent != 25 || storage.UsedPercent != 75 {
		t.Fatalf("percents = %.1f/%.1f, want 25/75", storage.FreePercent, storage.UsedPercent)
	}
}

func TestParseDiskUsageClampsAvailable(t *testing.T) {
	storage, err := parseDiskUsage(`AmountDataAvailable: 2000
TotalDataCapacity: 1000
`)
	if err != nil {
		t.Fatal(err)
	}
	if storage.AmountDataAvailable != 1000 || storage.UsedData != 0 {
		t.Fatalf("storage = %+v", storage)
	}
}

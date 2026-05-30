package media

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanDCIM(t *testing.T) {
	root := t.TempDir()
	dcim := filepath.Join(root, "DCIM", "100APPLE")
	if err := os.MkdirAll(dcim, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dcim, "IMG_0001.HEIC"), []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dcim, "IMG_0002.MOV"), []byte("video-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dcim, "notes.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Scan(context.Background(), root, Options{LargeThreshold: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Summary.TotalFiles != 2 {
		t.Fatalf("TotalFiles = %d, want 2", result.Summary.TotalFiles)
	}
	if result.Summary.PhotoFiles != 1 || result.Summary.VideoFiles != 1 {
		t.Fatalf("summary = %+v", result.Summary)
	}
}

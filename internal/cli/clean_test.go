package cli

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chenhg5/imole/internal/backup"
)

func TestCleanFileFilterDryRun(t *testing.T) {
	t.Setenv(noDeleteEnv, "")
	manifestPath := filepath.Join(t.TempDir(), backup.ManifestName)
	manifest := backup.Manifest{
		Version:   1,
		CreatedAt: time.Now(),
		Root:      "imagecapture:test",
		Files: []backup.ManifestFile{
			{SourceRel: "DCIM/202507__/IMG_7523.MOV", DestRel: "2025/07/IMG_7523.MOV", Kind: "video", Size: 10, Verified: true},
			{SourceRel: "DCIM/202507__/IMG_7510.MOV", DestRel: "2025/07/IMG_7510.MOV", Kind: "video", Size: 20, Verified: true},
		},
	}
	if err := backup.WriteManifest(manifestPath, manifest); err != nil {
		t.Fatal(err)
	}

	var out, errOut bytes.Buffer
	app := New(&out, &errOut)
	code := app.Run(context.Background(), []string{
		"clean",
		"--manifest", manifestPath,
		"--file", "DCIM/202507__/IMG_7523.MOV",
		"--dry-run",
	})
	if code != ExitDryRun {
		t.Fatalf("clean --file --dry-run exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "IMG_7523.MOV") {
		t.Fatalf("output missing selected file:\n%s", out.String())
	}
	if strings.Contains(out.String(), "IMG_7510.MOV") {
		t.Fatalf("output included unselected file:\n%s", out.String())
	}
	if !strings.Contains(errOut.String(), "1 files") {
		t.Fatalf("dry-run stderr missing selected count:\n%s", errOut.String())
	}
}

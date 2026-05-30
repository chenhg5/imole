package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
)

func TestRunBackup(t *testing.T) {
	srcRoot := t.TempDir()
	src := filepath.Join(srcRoot, "IMG_0001.MOV")
	if err := os.WriteFile(src, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	item := media.NewItem(srcRoot, src, 5, time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC))
	scan := media.Result{
		Summary: media.Summary{Root: srcRoot},
		Items:   []media.Item{item},
	}

	manifest, err := Run(context.Background(), scan, Options{Destination: dest, Filter: filter.Default()})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Summary.CopiedFiles != 1 || manifest.Summary.VerifiedFiles != 1 {
		t.Fatalf("summary = %+v", manifest.Summary)
	}
	if _, err := os.Stat(filepath.Join(dest, ManifestName)); err != nil {
		t.Fatal(err)
	}
}

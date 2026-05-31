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

func TestRunBackupFileFilterDryRun(t *testing.T) {
	srcRoot := t.TempDir()
	video := filepath.Join(srcRoot, "DCIM", "100APPLE", "IMG_0001.MOV")
	photo := filepath.Join(srcRoot, "DCIM", "100APPLE", "IMG_0002.HEIC")
	if err := os.MkdirAll(filepath.Dir(video), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(video, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(photo, []byte("photo"), 0o644); err != nil {
		t.Fatal(err)
	}
	modTime := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	videoItem := media.NewItem(srcRoot, video, 5, modTime)
	photoItem := media.NewItem(srcRoot, photo, 5, modTime)
	scan := media.Result{
		Summary: media.Summary{Root: srcRoot},
		Items:   []media.Item{videoItem, photoItem},
	}

	manifest, err := Run(context.Background(), scan, Options{
		Destination: t.TempDir(),
		Filter:      filter.Filter{Only: filter.KindAll, Files: []string{videoItem.RelPath}, Now: modTime},
		DryRun:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Summary.SelectedFiles != 1 {
		t.Fatalf("selected files = %d, want 1", manifest.Summary.SelectedFiles)
	}
	if len(manifest.Files) != 1 || manifest.Files[0].SourceRel != videoItem.RelPath {
		t.Fatalf("files = %+v, want only %s", manifest.Files, videoItem.RelPath)
	}
}

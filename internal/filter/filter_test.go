package filter

import (
	"testing"
	"time"

	"github.com/chenhg5/imole/internal/media"
)

func TestParseSize(t *testing.T) {
	got, err := ParseSize("1.5GB")
	if err != nil {
		t.Fatal(err)
	}
	want := int64(1.5 * 1024 * 1024 * 1024)
	if got != want {
		t.Fatalf("ParseSize() = %d, want %d", got, want)
	}
}

func TestFilterMatch(t *testing.T) {
	now := time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)
	item := media.Item{Kind: "video", Size: 1024, ModTime: now.Add(-48 * time.Hour)}
	f := Filter{Only: KindVideos, OlderThan: 24 * time.Hour, LargeThan: 512, Now: now}
	if !f.Match(item) {
		t.Fatal("expected item to match")
	}
}

func TestFilterMatchFiles(t *testing.T) {
	item := media.Item{RelPath: "DCIM/202507__/IMG_7523.MOV", Name: "IMG_7523.MOV", Kind: "video"}
	if !(Filter{Only: KindAll, Files: []string{"imagecapture://DCIM/202507__/IMG_7523.MOV"}}).Match(item) {
		t.Fatal("expected imagecapture rel path to match")
	}
	if (Filter{Only: KindAll, Files: []string{"IMG_7523.MOV"}}).Match(item) {
		t.Fatal("expected basename-only value to not match")
	}
	if (Filter{Only: KindAll, Files: []string{"DCIM/other.mov"}}).Match(item) {
		t.Fatal("expected different file to not match")
	}
}

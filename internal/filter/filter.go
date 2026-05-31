// Package filter provides size, age, and kind filtering for media items.
package filter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/media"
)

type Kind string

const (
	KindAll    Kind = "all"
	KindPhotos Kind = "photos"
	KindVideos Kind = "videos"
)

type Filter struct {
	Only      Kind
	OlderThan time.Duration
	LargeThan int64
	Files     []string
	Now       time.Time
}

func Default() Filter {
	return Filter{Only: KindAll, Now: time.Now()}
}

func (f Filter) Match(item media.Item) bool {
	if len(f.Files) > 0 && !matchFile(f.Files, item) {
		return false
	}
	if f.Only == KindPhotos && !item.IsPhoto() {
		return false
	}
	if f.Only == KindVideos && !item.IsVideo() {
		return false
	}
	if f.OlderThan > 0 && f.Now.Sub(item.ModTime) < f.OlderThan {
		return false
	}
	if f.LargeThan > 0 && item.Size < f.LargeThan {
		return false
	}
	return true
}

func matchFile(files []string, item media.Item) bool {
	for _, file := range files {
		normalized := NormalizeFile(file)
		if normalized == item.RelPath {
			return true
		}
	}
	return false
}

func NormalizeFile(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "imagecapture://")
	path = strings.TrimPrefix(path, "./")
	path = strings.ReplaceAll(path, "\\", "/")
	return strings.TrimPrefix(path, "/")
}

func ParseKind(s string) (Kind, error) {
	switch Kind(strings.ToLower(s)) {
	case KindAll, KindPhotos, KindVideos:
		return Kind(strings.ToLower(s)), nil
	default:
		return "", fmt.Errorf("invalid --only value %q", s)
	}
}

func ParseSize(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	raw := strings.TrimSpace(strings.ToUpper(s))
	mult := int64(1)
	for _, suffix := range []struct {
		s string
		m int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"G", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"M", 1024 * 1024},
		{"KB", 1024},
		{"K", 1024},
		{"B", 1},
	} {
		if strings.HasSuffix(raw, suffix.s) {
			mult = suffix.m
			raw = strings.TrimSpace(strings.TrimSuffix(raw, suffix.s))
			break
		}
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v < 0 {
		return 0, fmt.Errorf("invalid size %q", s)
	}
	return int64(v * float64(mult)), nil
}

func ParseAge(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	raw := strings.TrimSpace(strings.ToLower(s))
	unit := 24 * time.Hour
	switch {
	case strings.HasSuffix(raw, "d"):
		raw = strings.TrimSuffix(raw, "d")
	case strings.HasSuffix(raw, "day"):
		raw = strings.TrimSuffix(raw, "day")
	case strings.HasSuffix(raw, "days"):
		raw = strings.TrimSuffix(raw, "days")
	case strings.HasSuffix(raw, "m"):
		raw = strings.TrimSuffix(raw, "m")
		unit = 30 * 24 * time.Hour
	case strings.HasSuffix(raw, "mo"):
		raw = strings.TrimSuffix(raw, "mo")
		unit = 30 * 24 * time.Hour
	case strings.HasSuffix(raw, "y"):
		raw = strings.TrimSuffix(raw, "y")
		unit = 365 * 24 * time.Hour
	default:
		return 0, fmt.Errorf("invalid age %q, use values like 90d, 6m, 1y", s)
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid age %q", s)
	}
	return time.Duration(n) * unit, nil
}

// Package filter provides size, age, and kind filtering for media items.
package filter

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/geo"
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

	// Extension filter (case-insensitive, without leading dot, e.g. "png").
	Ext string

	// Metadata filters — only effective when item metadata has been fetched.
	Country     string    // filter by country name / code / region (partial match)
	NoGPS       bool      // keep only items WITHOUT GPS data
	TakenAfter  time.Time // keep items taken at or after this time
	TakenBefore time.Time // keep items taken before this time
	DurationGt  float64   // keep videos with duration > N seconds
	// Dimension filters (require --with-meta).
	MinWidth  int
	MinHeight int
	MaxWidth  int
	MaxHeight int

	// iCloud placeholder filters (no metadata required; uses CheckCloudPlaceholder heuristic).
	SkipPlaceholders bool // exclude files flagged as iCloud thumbnails
	OnlyPlaceholders bool // include ONLY files flagged as iCloud thumbnails
}

func Default() Filter {
	return Filter{Only: KindAll, Now: time.Now()}
}

// NeedsMetadata returns true if any metadata-dependent filter is set.
func (f Filter) NeedsMetadata() bool {
	return f.Country != "" || f.NoGPS || !f.TakenAfter.IsZero() || !f.TakenBefore.IsZero() || f.DurationGt > 0 ||
		f.MinWidth > 0 || f.MinHeight > 0 || f.MaxWidth > 0 || f.MaxHeight > 0
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
	// Extension filter: compare without leading dot, case-insensitive.
	if f.Ext != "" {
		itemExt := strings.TrimPrefix(strings.ToLower(item.Ext), ".")
		filterExt := strings.TrimPrefix(strings.ToLower(f.Ext), ".")
		if itemExt != filterExt {
			return false
		}
	}
	if f.OlderThan > 0 && f.Now.Sub(item.ModTime) < f.OlderThan {
		return false
	}
	if f.LargeThan > 0 && item.Size < f.LargeThan {
		return false
	}

	// Metadata filters.
	if f.NoGPS && item.HasGPS {
		return false
	}
	// iCloud placeholder filters.
	if f.SkipPlaceholders && item.IsCloudPlaceholder {
		return false
	}
	if f.OnlyPlaceholders && !item.IsCloudPlaceholder {
		return false
	}

	if f.Country != "" {
		loc := geo.Location{
			Country:     item.Country,
			CountryCode: item.CountryCode,
			Continent:   item.Continent,
			Region:      item.Region,
		}
		if !geo.MatchCountry(loc, f.Country) {
			return false
		}
	}
	if !f.TakenAfter.IsZero() && !item.TakenAt.IsZero() && item.TakenAt.Before(f.TakenAfter) {
		return false
	}
	if !f.TakenBefore.IsZero() && !item.TakenAt.IsZero() && !item.TakenAt.Before(f.TakenBefore) {
		return false
	}
	if f.DurationGt > 0 && item.DurationSec <= f.DurationGt {
		return false
	}
	// Dimension filters (only applied when dimensions are populated via --with-meta).
	if item.Width > 0 || item.Height > 0 {
		if f.MinWidth > 0 && item.Width < f.MinWidth {
			return false
		}
		if f.MinHeight > 0 && item.Height < f.MinHeight {
			return false
		}
		if f.MaxWidth > 0 && item.Width > f.MaxWidth {
			return false
		}
		if f.MaxHeight > 0 && item.Height > f.MaxHeight {
			return false
		}
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
		return "", fmt.Errorf("invalid --only value %q: must be all, photos, or videos", s)
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

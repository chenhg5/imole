// Package media provides DCIM media file scanning and classification.
package media

import (
	"path/filepath"
	"strings"
	"time"
)

type Item struct {
	SourcePath string    `json:"source_path"`
	RelPath    string    `json:"rel_path"`
	Name       string    `json:"name"`
	Ext        string    `json:"ext"`
	Kind       string    `json:"kind"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`

	// Metadata fields — populated only when --with-meta is used.
	// GPS coordinates from EXIF data (zero value means absent).
	GPSLat float64 `json:"gps_lat,omitempty"`
	GPSLon float64 `json:"gps_lon,omitempty"`
	HasGPS bool    `json:"has_gps,omitempty"`
	// TakenAt is the DateTimeOriginal from EXIF (zero if absent).
	TakenAt time.Time `json:"taken_at,omitempty"`
	// Image/video dimensions.
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
	// Duration in seconds (video only).
	DurationSec float64 `json:"duration_sec,omitempty"`
	// Human-readable location resolved from GPS coordinates (offline).
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
	Region      string `json:"region,omitempty"`
	Continent   string `json:"continent,omitempty"`

	// IsCloudPlaceholder is true when the file is likely an iCloud "optimized"
	// thumbnail rather than the full-resolution original.  It is set by
	// heuristic (size vs. pixel-count ratio) and does not require --with-meta,
	// though having dimensions makes the signal much stronger.
	IsCloudPlaceholder bool `json:"is_cloud_placeholder,omitempty"`
}

func NewItem(root, path string, size int64, modTime time.Time) Item {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	ext := strings.ToLower(filepath.Ext(path))
	return Item{
		SourcePath: path,
		RelPath:    filepath.ToSlash(rel),
		Name:       filepath.Base(path),
		Ext:        ext,
		Kind:       kindForExt(ext),
		Size:       size,
		ModTime:    modTime,
	}
}

func (i Item) IsVideo() bool      { return i.Kind == "video" }
func (i Item) IsPhoto() bool      { return i.Kind == "photo" }
func (i Item) IsScreenshot() bool { return i.Ext == ".png" }

// CheckCloudPlaceholder sets IsCloudPlaceholder based on a heuristic:
// when dimensions are known, flag photos whose bytes-per-megapixel is far below
// what a normal compressed image produces.  When dimensions are unknown, flag
// photos/videos whose on-device size is suspiciously small for a modern iPhone camera.
func (i *Item) CheckCloudPlaceholder() {
	if i.Kind != "photo" && i.Kind != "video" {
		return
	}
	// Only applies to common iPhone photo/video formats.
	switch i.Ext {
	case ".heic", ".heif", ".jpg", ".jpeg", ".mov", ".mp4":
	default:
		return
	}

	if i.Width > 0 && i.Height > 0 {
		// With dimensions: flag if bytes-per-megapixel < 100 KB/MP.
		// A properly-encoded 12MP HEIC is typically 400–800 KB/MP.
		mp := float64(i.Width) * float64(i.Height) / 1_000_000.0
		if mp > 2 {
			bytesPerMP := float64(i.Size) / mp
			i.IsCloudPlaceholder = bytesPerMP < 100_000 // < 100 KB/MP
		}
		return
	}

	// Without dimensions: use absolute size thresholds.
	switch i.Kind {
	case "photo":
		// Modern iPhone photos (12MP+) are 3–12 MB. Under 500 KB is suspicious.
		i.IsCloudPlaceholder = i.Size < 500_000
	case "video":
		// Even a 5-second 4K clip is several MB. Under 1 MB is suspicious.
		i.IsCloudPlaceholder = i.Size < 1_000_000
	}
}

func kindForExt(ext string) string {
	switch ext {
	case ".mov", ".mp4", ".m4v", ".avi":
		return "video"
	case ".jpg", ".jpeg", ".heic", ".heif", ".png", ".gif", ".tif", ".tiff", ".dng":
		return "photo"
	default:
		return "other"
	}
}

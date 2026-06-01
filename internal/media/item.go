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

func (i Item) IsVideo() bool { return i.Kind == "video" }
func (i Item) IsPhoto() bool { return i.Kind == "photo" }

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

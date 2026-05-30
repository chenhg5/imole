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

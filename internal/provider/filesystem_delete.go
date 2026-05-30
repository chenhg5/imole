package provider

import (
	"fmt"
	"os"
	"path/filepath"
)

// DeleteFilesystem deletes files from a mounted filesystem path using os.Remove.
// source is the root mount point (e.g. an ifuse mount on Linux, or an iTunes
// DCIM mount on Windows). Each request.Path is a relative path from the
// manifest (e.g. "DCIM/100APPLE/IMG_1234.MOV") and is joined with source to
// form the absolute path.
//
// Space is reclaimed immediately — there is no "Recently Deleted" buffer when
// deleting directly from the mounted filesystem.
func DeleteFilesystem(source string, requests []DeleteRequest) ([]DeleteResult, error) {
	if source == "" {
		return nil, fmt.Errorf("source mount path is required for filesystem deletion")
	}
	results := make([]DeleteResult, len(requests))
	for i, req := range requests {
		fullPath := filepath.Join(source, req.Path)
		err := os.Remove(fullPath)
		if err != nil {
			results[i] = DeleteResult{
				Path:  req.Path,
				Error: err.Error(),
			}
		} else {
			results[i] = DeleteResult{
				Path:    req.Path,
				Deleted: true,
			}
		}
	}
	return results, nil
}

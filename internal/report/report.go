// Package report generates summaries from backup manifests.
package report

import (
	"os"
	"path/filepath"

	"github.com/chenhg5/imole/internal/backup"
)

type Summary struct {
	Files         int   `json:"files"`
	Verified      int   `json:"verified"`
	Failed        int   `json:"failed"`
	TotalSize     int64 `json:"total_size"`
	VerifiedSize  int64 `json:"verified_size"`
	Cleanable     int   `json:"cleanable"`
	CleanableSize int64 `json:"cleanable_size"`
}

func FromManifest(manifest backup.Manifest) Summary {
	var out Summary
	for _, file := range manifest.Files {
		out.Files++
		out.TotalSize += file.Size
		if file.Verified {
			out.Verified++
			out.VerifiedSize += file.Size
			out.Cleanable++
			out.CleanableSize += file.Size
		}
		if file.Error != "" {
			out.Failed++
		}
	}
	return out
}

// VerifyResult holds the result of re-checking backed-up files on disk.
type VerifyResult struct {
	Total     int      `json:"total"`
	OnDisk    int      `json:"on_disk"`
	Missing   int      `json:"missing"`
	Corrupted int      `json:"corrupted"`
	HealthPct float64  `json:"health_pct"`
	Issues    []string `json:"issues,omitempty"`
}

// VerifyManifest checks every verified file in the manifest still exists on disk
// at manifestDir and has the expected size.
func VerifyManifest(manifest backup.Manifest, manifestDir string) VerifyResult {
	var result VerifyResult
	for _, file := range manifest.Files {
		if !file.Verified {
			continue
		}
		result.Total++
		localPath := filepath.Join(manifestDir, filepath.FromSlash(file.DestRel))
		info, err := os.Stat(localPath)
		if err != nil {
			result.Missing++
			result.Issues = append(result.Issues, "missing: "+file.DestRel)
			continue
		}
		if info.Size() != file.Size {
			result.Corrupted++
			result.Issues = append(result.Issues, "size mismatch: "+file.DestRel)
			continue
		}
		result.OnDisk++
	}
	if result.Total > 0 {
		result.HealthPct = float64(result.OnDisk) / float64(result.Total) * 100
	} else {
		result.HealthPct = 100
	}
	// Cap issues list at 20 for display purposes
	if len(result.Issues) > 20 {
		result.Issues = result.Issues[:20]
	}
	return result
}

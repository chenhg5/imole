// Package report generates summaries from backup manifests.
package report

import "github.com/chenhg5/imole/internal/backup"

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

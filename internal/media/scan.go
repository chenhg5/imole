package media

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
)

type Summary struct {
	Root        string `json:"root"`
	TotalFiles  int64  `json:"total_files"`
	TotalSize   int64  `json:"total_size"`
	PhotoFiles  int64  `json:"photo_files"`
	PhotoSize   int64  `json:"photo_size"`
	VideoFiles  int64  `json:"video_files"`
	VideoSize   int64  `json:"video_size"`
	OtherFiles  int64  `json:"other_files"`
	OtherSize   int64  `json:"other_size"`
	LargeFiles  int64  `json:"large_files"`
	LargeSize   int64  `json:"large_size"`
	OldFiles    int64  `json:"old_files"`
	OldSize     int64  `json:"old_size"`
	ScanSkipped int64  `json:"scan_skipped"`
}

type Result struct {
	Summary Summary `json:"summary"`
	Items   []Item  `json:"items"`
}

type Options struct {
	LargeThreshold int64
	OldBeforeUnix  int64
	WithMeta       bool // fetch EXIF metadata (GPS, date, dimensions)
}

func Scan(ctx context.Context, root string, opts Options) (Result, error) {
	if root == "" {
		return Result{}, errors.New("scan root is empty")
	}
	dcimRoot := resolveMediaRoot(root)
	var skipped atomic.Int64
	var items []Item

	err := filepath.WalkDir(dcimRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			skipped.Add(1)
			return nil
		}
		if cerr := ctx.Err(); cerr != nil {
			return cerr
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") && path != dcimRoot {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			skipped.Add(1)
			return nil
		}
		item := NewItem(dcimRoot, path, info.Size(), info.ModTime())
		if item.Kind == "other" {
			return nil
		}
		items = append(items, item)
		return nil
	})
	if err != nil {
		return Result{}, err
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})

	summary := Summary{Root: dcimRoot, ScanSkipped: skipped.Load()}
	for _, item := range items {
		summary.TotalFiles++
		summary.TotalSize += item.Size
		switch item.Kind {
		case "photo":
			summary.PhotoFiles++
			summary.PhotoSize += item.Size
		case "video":
			summary.VideoFiles++
			summary.VideoSize += item.Size
		default:
			summary.OtherFiles++
			summary.OtherSize += item.Size
		}
		if opts.LargeThreshold > 0 && item.Size >= opts.LargeThreshold {
			summary.LargeFiles++
			summary.LargeSize += item.Size
		}
		if opts.OldBeforeUnix > 0 && item.ModTime.Unix() < opts.OldBeforeUnix {
			summary.OldFiles++
			summary.OldSize += item.Size
		}
	}

	return Result{Summary: summary, Items: items}, nil
}

func resolveMediaRoot(root string) string {
	candidates := []string{
		filepath.Join(root, "DCIM"),
		filepath.Join(root, "Media", "DCIM"),
		root,
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return root
}

// Package backup copies media files and writes verification manifests.
package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
)

type Options struct {
	Destination string
	Filter      filter.Filter
	Layout      string // e.g. "{year}/{month}/{type}/{filename}", empty = default
	DryRun      bool
}

func Run(ctx context.Context, scan media.Result, opts Options) (Manifest, error) {
	if opts.Destination == "" {
		return Manifest{}, fmt.Errorf("backup destination is required")
	}
	destRoot, err := filepath.Abs(opts.Destination)
	if err != nil {
		return Manifest{}, err
	}
	if !opts.DryRun {
		if err := os.MkdirAll(destRoot, 0o755); err != nil {
			return Manifest{}, err
		}
	}

	manifest := Manifest{
		Version:   1,
		CreatedAt: time.Now(),
		Root:      scan.Summary.Root,
	}

	for _, item := range scan.Items {
		if !opts.Filter.Match(item) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return manifest, err
		}
		destRel := DestinationRel(item, opts.Layout)
		entry := ManifestFile{
			SourceRel: item.RelPath,
			DestRel:   destRel,
			Kind:      item.Kind,
			Size:      item.Size,
			ModTime:   item.ModTime,
		}
		manifest.Summary.SelectedFiles++
		manifest.Summary.SelectedSize += item.Size

		if opts.DryRun {
			manifest.Files = append(manifest.Files, entry)
			continue
		}

		destPath := filepath.Join(destRoot, filepath.FromSlash(destRel))
		if err := copyFile(item.SourcePath, destPath, item.Size); err != nil {
			entry.Error = err.Error()
			manifest.Summary.FailedFiles++
			manifest.Files = append(manifest.Files, entry)
			continue
		}
		entry.Verified = verifyFast(destPath, item.Size)
		manifest.Summary.CopiedFiles++
		manifest.Summary.CopiedSize += item.Size
		if entry.Verified {
			manifest.Summary.VerifiedFiles++
			manifest.Summary.VerifiedSize += item.Size
		}
		manifest.Files = append(manifest.Files, entry)
	}

	if !opts.DryRun {
		err = WriteManifest(filepath.Join(destRoot, ManifestName), manifest)
	}
	return manifest, err
}

// DestinationRel returns the relative destination path for an item.
// layout supports tokens: {year} {month} {day} {type} {filename} {ext} {date}
// Empty layout uses the default: {year}/{month}/{filename}
func DestinationRel(item media.Item, layout string) string {
	if layout == "" {
		year, month, _ := item.ModTime.Date()
		return filepath.ToSlash(filepath.Join(fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", int(month)), item.Name))
	}
	return applyLayout(layout, item)
}

func applyLayout(layout string, item media.Item) string {
	year, month, day := item.ModTime.Date()
	ext := item.Ext
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	typeLabel := item.Kind // "photo", "video", "other"
	if typeLabel == "photo" {
		typeLabel = "photos"
	} else if typeLabel == "video" {
		typeLabel = "videos"
	}
	r := layout
	r = strings.ReplaceAll(r, "{year}", fmt.Sprintf("%04d", year))
	r = strings.ReplaceAll(r, "{month}", fmt.Sprintf("%02d", int(month)))
	r = strings.ReplaceAll(r, "{day}", fmt.Sprintf("%02d", day))
	r = strings.ReplaceAll(r, "{type}", typeLabel)
	r = strings.ReplaceAll(r, "{filename}", item.Name)
	r = strings.ReplaceAll(r, "{ext}", ext)
	r = strings.ReplaceAll(r, "{date}", fmt.Sprintf("%04d-%02d-%02d", year, int(month), day))
	return filepath.ToSlash(r)
}

func copyFile(src, dst string, expectedSize int64) error {
	if info, err := os.Stat(dst); err == nil && info.Size() == expectedSize {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	tmp := dst + ".imole-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, dst)
}

func verifyFast(path string, size int64) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() == size
}

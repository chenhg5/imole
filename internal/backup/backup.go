package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
)

type Options struct {
	Destination string
	Filter      filter.Filter
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
		destRel := DestinationRel(item)
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

func DestinationRel(item media.Item) string {
	year, month, _ := item.ModTime.Date()
	return filepath.ToSlash(filepath.Join(fmt.Sprintf("%04d", year), fmt.Sprintf("%02d", int(month)), item.Name))
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

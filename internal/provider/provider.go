package provider

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
)

type Name string

const (
	Auto         Name = "auto"
	Filesystem   Name = "filesystem"
	GPhoto       Name = "gphoto"
	ImageCapture Name = "imagecapture"
)

type Options struct {
	Name           Name
	Source         string
	LargeThreshold int64
	OldAge         time.Duration
}

func Scan(ctx context.Context, opts Options) (media.Result, error) {
	var oldBefore int64
	if opts.OldAge > 0 {
		oldBefore = time.Now().Add(-opts.OldAge).Unix()
	}
	scanOpts := media.Options{LargeThreshold: opts.LargeThreshold, OldBeforeUnix: oldBefore}

	if opts.Source != "" {
		return ScanFilesystem(ctx, opts.Source, scanOpts)
	}

	switch opts.Name {
	case "", Auto:
		if runtime.GOOS == "darwin" {
			var failures []string
			if result, err := ScanImageCapture(ctx, scanOpts); err == nil {
				return result, nil
			} else {
				failures = append(failures, "imagecapture: "+err.Error())
			}
			if result, err := ScanGPhoto(ctx, scanOpts); err == nil {
				return result, nil
			} else {
				failures = append(failures, "gphoto: "+err.Error())
			}
			return media.Result{}, fmt.Errorf("no macOS media provider is ready; use --source PATH for now (%s)", strings.Join(failures, "; "))
		}
		return media.Result{}, fmt.Errorf("auto provider requires --source PATH on %s", runtime.GOOS)
	case Filesystem:
		if opts.Source == "" {
			return media.Result{}, fmt.Errorf("filesystem provider requires --source PATH")
		}
		return ScanFilesystem(ctx, opts.Source, scanOpts)
	case GPhoto:
		return ScanGPhoto(ctx, scanOpts)
	case ImageCapture:
		return ScanImageCapture(ctx, scanOpts)
	default:
		return media.Result{}, fmt.Errorf("unknown provider %q", opts.Name)
	}
}

func FilteredItems(result media.Result, f filter.Filter) []media.Item {
	items := make([]media.Item, 0, len(result.Items))
	for _, item := range result.Items {
		if f.Match(item) {
			items = append(items, item)
		}
	}
	return items
}

// Package provider provides media backend implementations: ImageCaptureCore (macOS USB), filesystem, and gphoto2 (Linux USB).
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
	WithMeta       bool // fetch EXIF metadata (GPS, date, dimensions)
}

func Scan(ctx context.Context, opts Options) (media.Result, error) {
	var oldBefore int64
	if opts.OldAge > 0 {
		oldBefore = time.Now().Add(-opts.OldAge).Unix()
	}
	scanOpts := media.Options{LargeThreshold: opts.LargeThreshold, OldBeforeUnix: oldBefore, WithMeta: opts.WithMeta}

	if opts.Source != "" {
		return ScanFilesystem(ctx, opts.Source, scanOpts)
	}

	scanIC := func(ctx context.Context, scanOpts media.Options) (media.Result, error) {
		if scanOpts.WithMeta {
			return ScanImageCaptureWithMeta(ctx, scanOpts)
		}
		return ScanImageCapture(ctx, scanOpts)
	}

	switch opts.Name {
	case "", Auto:
		if runtime.GOOS == "darwin" {
			var failures []string
			if result, err := scanIC(ctx, scanOpts); err == nil {
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
		if runtime.GOOS == "linux" {
			// On Linux: try gphoto2 (works over USB with libgphoto2).
			// For ifuse-mounted paths use --source instead.
			if result, err := ScanGPhoto(ctx, scanOpts); err == nil {
				return result, nil
			}
			return media.Result{}, fmt.Errorf(
				"no USB media provider found on Linux; options:\n" +
					"  1. install gphoto2:  sudo apt install gphoto2\n" +
					"  2. mount via ifuse:  ifuse ~/iphone && imole scan --source ~/iphone/DCIM",
			)
		}
		// Windows and other platforms: require an explicit mounted path.
		return media.Result{}, fmt.Errorf(
			"auto provider is not supported on %s; use --source PATH\n"+
				"  On Windows: connect iPhone, open iTunes/Finder, then use the DCIM path exposed by Windows Explorer",
			runtime.GOOS,
		)
	case Filesystem:
		if opts.Source == "" {
			return media.Result{}, fmt.Errorf("filesystem provider requires --source PATH")
		}
		return ScanFilesystem(ctx, opts.Source, scanOpts)
	case GPhoto:
		return ScanGPhoto(ctx, scanOpts)
	case ImageCapture:
		return scanIC(ctx, scanOpts)
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

package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

func parseFilter(only, olderThan, largeThan string) (filter.Filter, error) {
	f := filter.Default()
	kind, err := filter.ParseKind(only)
	if err != nil {
		return f, err
	}
	f.Only = kind
	if olderThan != "" {
		age, err := filter.ParseAge(olderThan)
		if err != nil {
			return f, err
		}
		f.OlderThan = age
	}
	if largeThan != "" {
		size, err := filter.ParseSize(largeThan)
		if err != nil {
			return f, err
		}
		f.LargeThan = size
	}
	return f, nil
}

func scanFromFlags(ctx context.Context, providerName, source string, largeThreshold int64, oldAge time.Duration) (media.Result, error) {
	return provider.Scan(ctx, provider.Options{
		Name:           provider.Name(providerName),
		Source:         source,
		LargeThreshold: largeThreshold,
		OldAge:         oldAge,
	})
}

func addFilterFlags(fs *flag.FlagSet, only, olderThan, largeThan *string) {
	fs.StringVar(only, "only", "all", "media filter: all, photos, videos")
	fs.StringVar(olderThan, "older-than", "", "include media older than an age, e.g. 90d, 6m, 1y")
	fs.StringVar(largeThan, "large-than", "", "include media larger than a size, e.g. 500MB, 1GB")
}

func addProviderFlags(fs *flag.FlagSet, providerName, source *string) {
	fs.StringVar(providerName, "provider", "auto", "media provider: auto, filesystem, imagecapture, gphoto")
	fs.StringVar(source, "source", "", "scan an existing mounted media path; implies filesystem provider")
}

func absPath(path string) string {
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}

func flagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
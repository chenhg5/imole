package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("--file cannot be empty")
	}
	*s = append(*s, value)
	return nil
}

// metaFlags holds metadata-based filter flag values.
type metaFlags struct {
	country          string
	noGPS            bool
	takenAfter       string
	takenBefore      string
	durationGt       float64
	minWidth         int
	minHeight        int
	maxWidth         int
	maxHeight        int
	skipPlaceholders bool
	onlyPlaceholders bool
}

func parseFilterMeta(only, olderThan, largeThan, ext string, files []string, meta metaFlags) (filter.Filter, error) {
	f := filter.Default()
	kind, err := filter.ParseKind(only)
	if err != nil {
		return f, err
	}
	f.Only = kind
	f.Files = normalizedFiles(files)
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
	f.Ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))

	// Metadata filters.
	f.Country = strings.TrimSpace(meta.country)
	f.NoGPS = meta.noGPS
	f.DurationGt = meta.durationGt
	f.MinWidth = meta.minWidth
	f.MinHeight = meta.minHeight
	f.MaxWidth = meta.maxWidth
	f.MaxHeight = meta.maxHeight
	f.SkipPlaceholders = meta.skipPlaceholders
	f.OnlyPlaceholders = meta.onlyPlaceholders
	if meta.takenAfter != "" {
		t, err := parseDate(meta.takenAfter)
		if err != nil {
			return f, fmt.Errorf("--taken-after: %w", err)
		}
		f.TakenAfter = t
	}
	if meta.takenBefore != "" {
		t, err := parseDate(meta.takenBefore)
		if err != nil {
			return f, fmt.Errorf("--taken-before: %w", err)
		}
		f.TakenBefore = t
	}
	return f, nil
}

// parseDate accepts YYYY-MM-DD or YYYY/MM/DD.
func parseDate(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, strings.TrimSpace(s), time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q, use YYYY-MM-DD", s)
}

func normalizedFiles(files []string) []string {
	if len(files) == 0 {
		return nil
	}
	out := make([]string, 0, len(files))
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		normalized := filter.NormalizeFile(file)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func scanFromFlags(ctx context.Context, providerName, source string, largeThreshold int64, oldAge time.Duration, withMeta bool) (media.Result, error) {
	return provider.Scan(ctx, provider.Options{
		Name:           provider.Name(providerName),
		Source:         source,
		LargeThreshold: largeThreshold,
		OldAge:         oldAge,
		WithMeta:       withMeta,
	})
}

func addFilterFlags(fs *flag.FlagSet, only, olderThan, largeThan, ext *string) {
	fs.StringVar(only, "only", "all", "media filter: all, photos, videos")
	fs.StringVar(olderThan, "older-than", "", "include media older than an age, e.g. 90d, 6m, 1y")
	fs.StringVar(largeThan, "large-than", "", "include media larger than a size, e.g. 500MB, 1GB")
	fs.StringVar(ext, "ext", "", "filter by file extension, e.g. png (≈screenshots), heic, mov")
}

func addMetaFilterFlags(fs *flag.FlagSet, m *metaFlags) {
	fs.StringVar(&m.country, "country", "", "keep items whose GPS resolves to this country/region (requires --with-meta)")
	fs.BoolVar(&m.noGPS, "no-gps", false, "keep items without GPS data (requires --with-meta)")
	fs.StringVar(&m.takenAfter, "taken-after", "", "keep items taken on or after date, e.g. 2023-01-01 (requires --with-meta)")
	fs.StringVar(&m.takenBefore, "taken-before", "", "keep items taken before date, e.g. 2024-01-01 (requires --with-meta)")
	fs.Float64Var(&m.durationGt, "duration-gt", 0, "keep videos longer than N seconds (requires --with-meta)")
	fs.IntVar(&m.minWidth, "min-width", 0, "keep items with width >= N pixels (requires --with-meta)")
	fs.IntVar(&m.minHeight, "min-height", 0, "keep items with height >= N pixels (requires --with-meta)")
	fs.IntVar(&m.maxWidth, "max-width", 0, "keep items with width <= N pixels (requires --with-meta)")
	fs.IntVar(&m.maxHeight, "max-height", 0, "keep items with height <= N pixels (requires --with-meta)")
	fs.BoolVar(&m.skipPlaceholders, "skip-placeholders", false, "exclude iCloud-optimized thumbnails (keep only confirmed full-res files)")
	fs.BoolVar(&m.onlyPlaceholders, "only-placeholders", false, "include ONLY iCloud-optimized thumbnails (to pipe into `imole icloud`)")
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

// scanHint returns a contextually appropriate hint for scan failures.
// When using imagecapture (USB on macOS), --source is not the right fix.
func scanHint(providerName, source string) string {
	if providerName == string(provider.ImageCapture) ||
		(providerName == string(provider.Auto) && source == "") {
		return "Make sure iPhone is unlocked, trusted, and the cable supports data transfer"
	}
	return "Try specifying the DCIM path: imole scan --source /path/to/DCIM"
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

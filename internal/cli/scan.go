package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/human"
)

func (a *App) runScan(ctx context.Context, args []string) int {
	var providerName, source, largeThan, oldAgeRaw string
	var jsonMode bool
	fs := flagSet("scan")
	addProviderFlags(fs, &providerName, &source)
	fs.StringVar(&largeThan, "large-than", "500MB", "large file threshold")
	fs.StringVar(&oldAgeRaw, "older-than", "1y", "old media threshold")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	large, err := filter.ParseSize(largeThan)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    fmt.Sprintf("invalid --large-than value: %s", err.Error()),
			Suggestion: "Use format like: 500MB, 1GB, 2GB",
			Retryable:  false,
		})
		return ExitUsage
	}
	oldAge, err := filter.ParseAge(oldAgeRaw)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    fmt.Sprintf("invalid --older-than value: %s", err.Error()),
			Suggestion: "Use format like: 90d, 6m, 1y",
			Retryable:  false,
		})
		return ExitUsage
	}
	result, err := scanFromFlags(ctx, providerName, source, large, oldAge)
	if err != nil {
		a.printError(runtimeError("scan_failed", err.Error(), "", true))
		return ExitError
	}
	if a.shouldJSON() || jsonMode {
		return a.writeJSON(result)
	}

	s := result.Summary
	fmt.Fprintln(a.out, "iMole Scan Report")
	fmt.Fprintf(a.out, "Source: %s\n\n", s.Root)
	fmt.Fprintf(a.out, "Media files: %d · %s\n", s.TotalFiles, human.Bytes(s.TotalSize))
	fmt.Fprintf(a.out, "Photos:      %d · %s\n", s.PhotoFiles, human.Bytes(s.PhotoSize))
	fmt.Fprintf(a.out, "Videos:      %d · %s\n", s.VideoFiles, human.Bytes(s.VideoSize))
	fmt.Fprintf(a.out, "Large media: %d · %s (>%s)\n", s.LargeFiles, human.Bytes(s.LargeSize), largeThan)
	fmt.Fprintf(a.out, "Old media:   %d · %s (>%s)\n", s.OldFiles, human.Bytes(s.OldSize), oldAgeRaw)
	if s.ScanSkipped > 0 {
		fmt.Fprintf(a.out, "Skipped:     %d unreadable entries\n", s.ScanSkipped)
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Recommended next steps:")
	fmt.Fprintln(a.out, "  imole videos --top 30")
	if source != "" {
		fmt.Fprintf(a.out, "  imole backup --source %s --to /path/to/backup --only videos --older-than 90d\n", source)
	} else {
		fmt.Fprintln(a.out, "  imole backup --source /path/to/DCIM --to /path/to/backup --only videos --older-than 90d")
	}
	return ExitSuccess
}
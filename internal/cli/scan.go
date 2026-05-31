package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/human"
)

func (a *App) runScan(ctx context.Context, args []string) int {
	var providerName, source, only, largeThan, oldAgeRaw, fields string
	var jsonMode bool
	fs := flagSet("scan")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &oldAgeRaw, &largeThan)
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output, e.g. summary.total_files,summary.photo_files")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	f, err := parseFilter(only, oldAgeRaw, largeThan)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    err.Error(),
			Suggestion: "Use --only photos|videos, --older-than 90d|6m|1y, --large-than 500MB|1GB",
			Retryable:  false,
		})
		return ExitUsage
	}
	a.status("Scanning device… (may take ~15 s for USB)")
	result, err := scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan)
	if err != nil {
		hint := scanHint(providerName, source)
		a.printError(runtimeError("scan_failed", err.Error(), hint, true))
		return ExitError
	}
	if result.Summary.Root != "" {
		a.status("Device ready: " + result.Summary.Root)
	}
	if a.shouldJSON() || jsonMode {
		return a.outputJSON(result, fields)
	}

	s := result.Summary
	fmt.Fprintln(a.out, a.bold("iMole Scan Report"))
	fmt.Fprintf(a.out, "Source: %s\n\n", a.cyan(s.Root))
	fmt.Fprintf(a.out, "Media files: %d · %s\n", s.TotalFiles, a.cyan(human.Bytes(s.TotalSize)))
	fmt.Fprintf(a.out, "Photos:      %d · %s\n", s.PhotoFiles, a.cyan(human.Bytes(s.PhotoSize)))
	fmt.Fprintf(a.out, "Videos:      %d · %s\n", s.VideoFiles, a.cyan(human.Bytes(s.VideoSize)))
	fmt.Fprintf(a.out, "Large media: %d · %s (>%s)\n", s.LargeFiles, a.cyan(human.Bytes(s.LargeSize)), largeThan)
	fmt.Fprintf(a.out, "Old media:   %d · %s (>%s)\n", s.OldFiles, a.cyan(human.Bytes(s.OldSize)), oldAgeRaw)
	if s.ScanSkipped > 0 {
		fmt.Fprintf(a.out, "Skipped:     %d unreadable entries\n", s.ScanSkipped)
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("Recommended next steps:"))
	fmt.Fprintln(a.out, a.dim("  imole videos --top 30"))
	if source != "" {
		fmt.Fprintf(a.out, a.dim("  imole backup --source %s --to /path/to/backup --only videos --older-than 90d\n"), source)
	} else {
		fmt.Fprintln(a.out, a.dim("  imole backup --source /path/to/DCIM --to /path/to/backup --only videos --older-than 90d"))
	}
	return ExitSuccess
}
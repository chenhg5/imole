package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

func (a *App) runScan(ctx context.Context, args []string) int {
	var providerName, source, only, largeThan, oldAgeRaw, fields string
	var top int
	var jsonMode, summary bool
	fs := flagSet("scan")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &oldAgeRaw, &largeThan)
	fs.IntVar(&top, "top", 0, "show top N largest files sorted by size; use with --only videos|photos|all")
	fs.BoolVar(&summary, "summary", false, "show compact stats table only (equivalent to old `stats` command)")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
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
		if top > 0 {
			items := media.TopItems(provider.FilteredItems(result, f), only, top)
			return a.outputJSON(items, fields)
		}
		return a.outputJSON(result, fields)
	}

	if top > 0 {
		return a.printTopItems(provider.FilteredItems(result, f), only, top)
	}
	if summary {
		return a.printScanSummary(result, oldAgeRaw, largeThan)
	}
	return a.printScanReport(result, source, largeThan, oldAgeRaw)
}

func (a *App) printScanReport(result media.Result, source, largeThan, oldAgeRaw string) int {
	s := result.Summary
	fmt.Fprintln(a.out, a.bold("iMole Scan Report"))
	fmt.Fprintf(a.out, "Source: %s\n\n", a.cyan(s.Root))
	fmt.Fprintf(a.out, "Media files: %d · %s\n", s.TotalFiles, a.cyan(human.Bytes(s.TotalSize)))
	fmt.Fprintf(a.out, "Photos:      %d · %s\n", s.PhotoFiles, a.cyan(human.Bytes(s.PhotoSize)))
	fmt.Fprintf(a.out, "Videos:      %d · %s\n", s.VideoFiles, a.cyan(human.Bytes(s.VideoSize)))
	if largeThan != "" {
		fmt.Fprintf(a.out, "Large media: %d · %s (>%s)\n", s.LargeFiles, a.cyan(human.Bytes(s.LargeSize)), largeThan)
	}
	if oldAgeRaw != "" {
		fmt.Fprintf(a.out, "Old media:   %d · %s (>%s)\n", s.OldFiles, a.cyan(human.Bytes(s.OldSize)), oldAgeRaw)
	}
	if s.ScanSkipped > 0 {
		fmt.Fprintf(a.out, "Skipped:     %d unreadable entries\n", s.ScanSkipped)
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("Recommended next steps:"))
	fmt.Fprintln(a.out, a.dim("  imole scan --top 30 --only videos"))
	if source != "" {
		fmt.Fprintf(a.out, a.dim("  imole backup --source %s --to /path/to/backup --only videos --older-than 90d\n"), source)
	} else {
		fmt.Fprintln(a.out, a.dim("  imole backup --to /path/to/backup --only videos --older-than 90d"))
	}
	return ExitSuccess
}

func (a *App) printScanSummary(result media.Result, oldAgeRaw, largeThan string) int {
	s := result.Summary
	fmt.Fprintln(a.out, a.bold("iMole Stats"))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Total:   %d files · %s\n", s.TotalFiles, a.cyan(human.Bytes(s.TotalSize)))
	fmt.Fprintf(a.out, "Photos:  %d files · %s\n", s.PhotoFiles, a.cyan(human.Bytes(s.PhotoSize)))
	fmt.Fprintf(a.out, "Videos:  %d files · %s\n", s.VideoFiles, a.cyan(human.Bytes(s.VideoSize)))
	if oldAgeRaw != "" {
		fmt.Fprintf(a.out, "Old:     %d files · %s (>%s)\n", s.OldFiles, a.cyan(human.Bytes(s.OldSize)), oldAgeRaw)
	}
	if largeThan != "" {
		fmt.Fprintf(a.out, "Large:   %d files · %s (>%s)\n", s.LargeFiles, a.cyan(human.Bytes(s.LargeSize)), largeThan)
	}
	return ExitSuccess
}

func (a *App) printTopItems(items []media.Item, only string, top int) int {
	topItems := media.TopItems(items, only, top)

	kindLabel := "Files"
	switch only {
	case "videos", "video":
		kindLabel = "Videos"
	case "photos", "photo":
		kindLabel = "Photos"
	}

	fmt.Fprintf(a.out, "%s\n\n", a.bold(fmt.Sprintf("Top %d %s", len(topItems), kindLabel)))
	for i, item := range topItems {
		fmt.Fprintf(a.out, "%2d. %-28s %s  %s\n",
			i+1,
			item.Name,
			a.cyan(fmt.Sprintf("%8s", human.Bytes(item.Size))),
			a.dim(item.ModTime.Format("2006-01-02")),
		)
	}
	return ExitSuccess
}

// runStats is a backward-compatible alias for `scan --summary`.
func (a *App) runStats(ctx context.Context, args []string) int {
	return a.runScan(ctx, append([]string{"--summary"}, args...))
}

// runVideos is a backward-compatible alias for `scan --only videos --top N`.
func (a *App) runVideos(ctx context.Context, args []string) int {
	// Translate --top flag; if not present, default to 20.
	// Pass through all args but inject --only videos if --only is not set.
	hasOnly := false
	hasSummary := false
	for _, arg := range args {
		if arg == "--only" || len(arg) > 7 && arg[:7] == "--only=" {
			hasOnly = true
		}
		if arg == "--summary" {
			hasSummary = true
		}
	}
	_ = hasSummary
	newArgs := args
	if !hasOnly {
		newArgs = append([]string{"--only", "videos"}, args...)
	}
	// If no --top is in args, inject --top 20
	hasTop := false
	for _, arg := range args {
		if arg == "--top" || len(arg) > 5 && arg[:5] == "--top" {
			hasTop = true
		}
	}
	if !hasTop {
		newArgs = append([]string{"--top", "20"}, newArgs...)
	}
	return a.runScan(ctx, newArgs)
}

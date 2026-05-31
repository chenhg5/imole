package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/human"
)

// StatsResult is the agent-friendly stats output with pre-computed human-readable sizes.
type StatsResult struct {
	TotalFiles int64       `json:"total_files"`
	TotalSize  int64       `json:"total_size"`
	TotalHuman string      `json:"total_size_human"`
	PhotoFiles int64       `json:"photo_files"`
	PhotoSize  int64       `json:"photo_size"`
	PhotoHuman string      `json:"photo_size_human"`
	VideoFiles int64       `json:"video_files"`
	VideoSize  int64       `json:"video_size"`
	VideoHuman string      `json:"video_size_human"`
	OldFiles   int64       `json:"old_files"`
	OldSize    int64       `json:"old_size"`
	OldHuman   string      `json:"old_size_human"`
	LargeFiles int64       `json:"large_files"`
	LargeSize  int64       `json:"large_size"`
	LargeHuman string      `json:"large_size_human"`
	Filter     StatsFilter `json:"filter"`
}

type StatsFilter struct {
	Only      string `json:"only"`
	OlderThan string `json:"older_than,omitempty"`
	LargeThan string `json:"large_than,omitempty"`
	Source    string `json:"source,omitempty"`
}

func (a *App) runStats(ctx context.Context, args []string) int {
	var providerName, source, only, oldAgeRaw, largeThan, fields string
	var jsonMode bool
	fs := flagSet("stats")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &oldAgeRaw, &largeThan)
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

	s := result.Summary
	stats := StatsResult{
		TotalFiles: s.TotalFiles,
		TotalSize:  s.TotalSize,
		TotalHuman: human.Bytes(s.TotalSize),
		PhotoFiles: s.PhotoFiles,
		PhotoSize:  s.PhotoSize,
		PhotoHuman: human.Bytes(s.PhotoSize),
		VideoFiles: s.VideoFiles,
		VideoSize:  s.VideoSize,
		VideoHuman: human.Bytes(s.VideoSize),
		OldFiles:   s.OldFiles,
		OldSize:    s.OldSize,
		OldHuman:   human.Bytes(s.OldSize),
		LargeFiles: s.LargeFiles,
		LargeSize:  s.LargeSize,
		LargeHuman: human.Bytes(s.LargeSize),
		Filter: StatsFilter{
			Only:      only,
			OlderThan: oldAgeRaw,
			LargeThan: largeThan,
			Source:    source,
		},
	}

	if a.shouldJSON() || jsonMode {
		return a.outputJSON(stats, fields)
	}

	// Human-readable output
	fmt.Fprintln(a.out, a.bold("iMole Stats"))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Total:   %d files · %s\n", stats.TotalFiles, a.cyan(stats.TotalHuman))
	fmt.Fprintf(a.out, "Photos:  %d files · %s\n", stats.PhotoFiles, a.cyan(stats.PhotoHuman))
	fmt.Fprintf(a.out, "Videos:  %d files · %s\n", stats.VideoFiles, a.cyan(stats.VideoHuman))
	if oldAgeRaw != "" {
		fmt.Fprintf(a.out, "Old:     %d files · %s (>%s)\n", stats.OldFiles, a.cyan(stats.OldHuman), oldAgeRaw)
	}
	if largeThan != "" {
		fmt.Fprintf(a.out, "Large:   %d files · %s (>%s)\n", stats.LargeFiles, a.cyan(stats.LargeHuman), largeThan)
	}
	return ExitSuccess
}

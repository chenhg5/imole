package cli

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/apps"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
	"github.com/chenhg5/imole/internal/scancache"
)

// shortErr returns the first line of err.Error(), truncated to 80 chars.
func shortErr(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if idx := strings.Index(msg, "\n"); idx > 0 {
		msg = msg[:idx]
	}
	if len(msg) > 80 {
		return msg[:77] + "…"
	}
	return msg
}

func (a *App) runScan(ctx context.Context, args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "apps", "app":
			return a.runScanApps(ctx, args[1:])
		case "media":
			args = args[1:]
		default:
			// Keep ordinary flag parsing for "scan --summary", etc.
		}
	}

	var providerName, source, only, largeThan, oldAgeRaw, fields string
	var top int
	var jsonMode, summary, useCache bool
	fs := flagSet("scan")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &oldAgeRaw, &largeThan)
	fs.IntVar(&top, "top", 0, "show top N largest files sorted by size; use with --only videos|photos|all")
	fs.BoolVar(&summary, "summary", false, "show compact stats table only")
	fs.BoolVar(&useCache, "cache", false, "use cached scan result if available and less than 1 hour old")
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

	if summary && source == "" && providerName == "auto" && only == "all" && largeThan == "" && oldAgeRaw == "" && top == 0 {
		return a.runScanSummaryAll(ctx, jsonMode, fields)
	}

	var result media.Result
	var fromCache bool

	if useCache {
		if entry, ok := scancache.Read(providerName, source, scancache.DefaultTTL); ok {
			result = entry.Result
			fromCache = true
			age := time.Since(entry.ScannedAt)
			mins := int(math.Round(age.Minutes()))
			a.status(fmt.Sprintf("Using cached scan from %d min ago (run without --cache for a fresh scan)", mins))
		}
	}

	if !fromCache {
		a.status("Scanning device… (may take ~15 s for USB)")
		result, err = scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan)
		if err != nil {
			// Before reporting failure, check whether a recent cache can save the day.
			// Use a generous 24-hour fallback window so short-lived USB/ImageCaptureCore
			// glitches don't break a user's workflow.
			if fallback, ok := scancache.Read(providerName, source, 24*time.Hour); ok {
				result = fallback.Result
				age := time.Since(fallback.ScannedAt)
				mins := int(math.Round(age.Minutes()))
				a.status(fmt.Sprintf(
					"⚠ Live scan failed (%s). Using cached data from %d min ago — re-run when iPhone is accessible.",
					shortErr(err), mins,
				))
			} else {
				hint := scanHint(providerName, source)
				a.printError(runtimeError("scan_failed", err.Error(), hint, true))
				return ExitError
			}
		} else {
			if result.Summary.Root != "" {
				a.status("Device ready: " + result.Summary.Root)
			}
			_ = scancache.Write(providerName, source, result)
		}
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

func (a *App) runScanSummaryAll(ctx context.Context, jsonMode bool, fields string) int {
	type appSummary struct {
		Count       int       `json:"count"`
		TotalSize   int64     `json:"total_size"`
		StaticSize  int64     `json:"static_size"`
		DynamicSize int64     `json:"dynamic_size"`
		Top         *apps.App `json:"top,omitempty"`
	}
	type mediaTop struct {
		Name    string `json:"name"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
	}
	type combined struct {
		Media     media.Summary `json:"media"`
		Apps      appSummary    `json:"apps"`
		TopVideo  *mediaTop     `json:"top_video,omitempty"`
		Generated string        `json:"generated"`
	}

	a.status("Scanning media…")
	mediaResult, mediaErr := scanFromFlags(ctx, "auto", "", 0, 0)
	if mediaErr != nil {
		if fallback, ok := scancache.Read("auto", "", 24*time.Hour); ok {
			mediaResult = fallback.Result
			a.status(fmt.Sprintf("⚠ Media scan failed (%s). Using cached media data.", shortErr(mediaErr)))
		} else {
			a.printError(runtimeError("scan_failed", mediaErr.Error(), scanHint("auto", ""), true))
			return ExitError
		}
	} else {
		_ = scancache.Write("auto", "", mediaResult)
	}

	a.status("Querying app storage…")
	appResult, appErr := apps.List(ctx, apps.ScopeUser)
	if appErr != nil {
		a.status("⚠ App storage unavailable: " + shortErr(appErr))
	}

	out := combined{
		Media:     mediaResult.Summary,
		Generated: time.Now().Format(time.RFC3339),
	}
	for _, app := range appResult.Apps {
		out.Apps.Count++
		out.Apps.TotalSize += app.TotalSize
		out.Apps.StaticSize += app.StaticSize
		out.Apps.DynamicSize += app.DynamicSize
	}
	if len(appResult.Apps) > 0 {
		top := appResult.Apps[0]
		out.Apps.Top = &top
	}
	if topVideos := media.TopItems(mediaResult.Items, "videos", 1); len(topVideos) > 0 {
		item := topVideos[0]
		out.TopVideo = &mediaTop{Name: item.Name, Size: item.Size, ModTime: item.ModTime.Format("2006-01-02")}
	}

	if a.shouldJSON() || jsonMode {
		return a.outputJSON(out, fields)
	}

	fmt.Fprintln(a.out, a.bold("iMole Storage Summary"))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Media:     %s · %d files\n", a.cyan(human.Bytes(out.Media.TotalSize)), out.Media.TotalFiles)
	fmt.Fprintf(a.out, "  Photos:  %s · %d files\n", human.Bytes(out.Media.PhotoSize), out.Media.PhotoFiles)
	fmt.Fprintf(a.out, "  Videos:  %s · %d files\n", human.Bytes(out.Media.VideoSize), out.Media.VideoFiles)
	if out.TopVideo != nil {
		fmt.Fprintf(a.out, "  Top video: %s · %s\n", out.TopVideo.Name, human.Bytes(out.TopVideo.Size))
	}
	fmt.Fprintln(a.out)
	if appErr == nil {
		fmt.Fprintf(a.out, "Apps:      %s · %d apps\n", a.cyan(human.Bytes(out.Apps.TotalSize)), out.Apps.Count)
		fmt.Fprintf(a.out, "  App code: %s\n", human.Bytes(out.Apps.StaticSize))
		fmt.Fprintf(a.out, "  App data: %s\n", human.Bytes(out.Apps.DynamicSize))
		if out.Apps.Top != nil {
			fmt.Fprintf(a.out, "  Top app:  %s · %s\n", out.Apps.Top.Name, human.Bytes(out.Apps.Top.TotalSize))
		}
	} else {
		fmt.Fprintln(a.out, "Apps:      unavailable")
		fmt.Fprintf(a.out, "  %s\n", appErr.Error())
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("Recommended next steps:"))
	fmt.Fprintln(a.out, a.dim("  imole scan --top 30 --only videos"))
	fmt.Fprintln(a.out, a.dim("  imole scan apps --top 20"))
	return ExitSuccess
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

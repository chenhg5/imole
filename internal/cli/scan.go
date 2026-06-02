package cli

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/apps"
	"github.com/chenhg5/imole/internal/device"
	"github.com/chenhg5/imole/internal/filter"
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
		case "help", "--help", "-h":
			a.runScanHelp()
			return ExitSuccess
		default:
			// Keep ordinary flag parsing for "scan --summary", etc.
		}
	}

	// Also handle: imole scan --help
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			a.runScanHelp()
			return ExitSuccess
		}
	}

	var providerName, source, only, largeThan, oldAgeRaw, fields, ext string
	var top, limit int
	var jsonMode, summary, useCache, withMeta bool
	var mf metaFlags
	fs := flagSet("scan")
	addProviderFlags(fs, &providerName, &source)
	addFilterFlags(fs, &only, &oldAgeRaw, &largeThan, &ext)
	addMetaFilterFlags(fs, &mf)
	fs.BoolVar(&withMeta, "with-meta", false, "fetch EXIF metadata (GPS, date, dimensions) — slower first time, cached 7 days")
	fs.IntVar(&top, "top", 0, "show top N largest files sorted by size; use with --only videos|photos|all")
	fs.IntVar(&limit, "limit", 0, "cap result to N items after filtering (sorted by size desc); 0 = no limit")
	fs.BoolVar(&summary, "summary", false, "show compact stats table only")
	fs.BoolVar(&useCache, "cache", false, "use cached scan result if available and less than 1 hour old")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	if err := parseFlags(fs, args); err != nil {
		if strings.Contains(err.Error(), "flag provided but not defined: -dry-run") {
			a.printError(&Error{
				Code:       "usage_error",
				Message:    "scan is read-only and does not accept --dry-run",
				Suggestion: "Remove --dry-run, or use: imole backup --dry-run / imole clean --dry-run for previews",
				Retryable:  false,
			})
			return ExitUsage
		}
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	// Auto-enable --with-meta if any metadata filter is specified.
	if mf.country != "" || mf.noGPS || mf.takenAfter != "" || mf.takenBefore != "" || mf.durationGt > 0 ||
		mf.minWidth > 0 || mf.minHeight > 0 || mf.maxWidth > 0 || mf.maxHeight > 0 {
		withMeta = true
	}

	f, err := parseFilterMeta(only, oldAgeRaw, largeThan, ext, nil, mf)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    err.Error(),
			Suggestion: "Use --only photos|videos, --older-than 90d|6m|1y, --large-than 500MB|1GB, --country CN, --taken-after 2023-01-01",
			Retryable:  false,
		})
		return ExitUsage
	}

	if summary && source == "" && providerName == "auto" && only == "all" && largeThan == "" && oldAgeRaw == "" && top == 0 && !withMeta {
		return a.runScanSummaryAll(ctx, jsonMode, fields)
	}

	var result media.Result
	var fromCache bool

	if useCache && !withMeta {
		if entry, ok := scancache.Read(providerName, source, scancache.DefaultTTL); ok {
			result = entry.Result
			fromCache = true
			age := time.Since(entry.ScannedAt)
			mins := int(math.Round(age.Minutes()))
			a.status(fmt.Sprintf("Using cached scan from %d min ago (run without --cache for a fresh scan)", mins))
		}
	}

	if !fromCache {
		spinMsg := "Scanning device… (may take ~15 s for USB)"
		if withMeta {
			spinMsg = "Scanning with metadata (GPS, date)… ~60 s first run, cached 7 days"
		}
		stopSpinner := a.startSpinner(spinMsg)
		result, err = scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan, withMeta)
		if err != nil {
			stopSpinner("")
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
			root := result.Summary.Root
			if root == "" {
				root = "device"
			}
			extraInfo := ""
			if withMeta {
				gpsCount := 0
				for _, it := range result.Items {
					if it.HasGPS {
						gpsCount++
					}
				}
				if gpsCount > 0 {
					extraInfo = fmt.Sprintf(" · %d with GPS", gpsCount)
				}
			}
			stopSpinner(fmt.Sprintf("Device ready: %s  (%d files · %s%s)", root, result.Summary.TotalFiles, human.Bytes(result.Summary.TotalSize), extraInfo))
			a.debug("Scan complete: %d items from %s", len(result.Items), root)
			if !withMeta {
				_ = scancache.Write(providerName, source, result)
			}
		}
	}

	// Apply filter then optional limit (sorted by size desc by default).
	filtered := provider.FilteredItems(result, f)
	hasMetaFilter := f.NeedsMetadata()
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	if a.shouldJSON() || jsonMode {
		if top > 0 {
			items := media.TopItems(filtered, only, top)
			return a.outputJSON(items, fields)
		}
		if hasMetaFilter || f.Only != "all" || f.OlderThan > 0 || f.LargeThan > 0 || len(f.Files) > 0 || f.Ext != "" || limit > 0 {
			// Return filtered items + mini summary when filters are active.
			type filteredResult struct {
				FilteredCount int          `json:"filtered_count"`
				FilteredSize  int64        `json:"filtered_size"`
				Items         []media.Item `json:"items"`
			}
			var sz int64
			for _, it := range filtered {
				sz += it.Size
			}
			return a.outputJSON(filteredResult{FilteredCount: len(filtered), FilteredSize: sz, Items: filtered}, fields)
		}
		return a.outputJSON(result, fields)
	}

	if top > 0 {
		return a.printTopItems(filtered, only, top)
	}
	if summary {
		return a.printScanSummary(result, oldAgeRaw, largeThan)
	}
	if hasMetaFilter {
		return a.printMetaFilterReport(filtered, f)
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
		Device    device.Info   `json:"device"`
		TopVideo  *mediaTop     `json:"top_video,omitempty"`
		Generated string        `json:"generated"`
	}

	deviceInfo := device.Check(ctx).Device

	stopSpinner := a.startSpinner("Scanning media…")
	mediaResult, mediaErr := scanFromFlags(ctx, "auto", "", 0, 0, false)
	if mediaErr != nil {
		stopSpinner("")
		if fallback, ok := scancache.Read("auto", "", 24*time.Hour); ok {
			mediaResult = fallback.Result
			a.status(fmt.Sprintf("⚠ Media scan failed (%s). Using cached media data.", shortErr(mediaErr)))
		} else {
			a.printError(runtimeError("scan_failed", mediaErr.Error(), scanHint("auto", ""), true))
			return ExitError
		}
	} else {
		stopSpinner(fmt.Sprintf("Media scan complete: %d files · %s", mediaResult.Summary.TotalFiles, human.Bytes(mediaResult.Summary.TotalSize)))
		_ = scancache.Write("auto", "", mediaResult)
	}

	stopAppSpinner := a.startSpinner("Querying app storage…")
	appResult, appErr := apps.List(ctx, apps.ScopeUser)
	if appErr != nil {
		stopAppSpinner("")
		a.status("⚠ App storage unavailable: " + shortErr(appErr))
	} else {
		stopAppSpinner(fmt.Sprintf("App storage ready: %d apps", len(appResult.Apps)))
	}

	out := combined{
		Media:     mediaResult.Summary,
		Device:    deviceInfo,
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
	if out.Device.Storage != nil {
		pct := out.Device.Storage.UsedPercent
		fmt.Fprintf(a.out, "Device:   %s %s free\n", a.progressBar(pct), human.Bytes(out.Device.Storage.AmountDataAvailable))
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

	// Type breakdown with visual bars
	if s.TotalSize > 0 {
		photoPct := float64(s.PhotoSize) * 100 / float64(s.TotalSize)
		videoPct := float64(s.VideoSize) * 100 / float64(s.TotalSize)
		fmt.Fprintf(a.out, "Photos:      %d files · %s  %s\n",
			s.PhotoFiles, a.cyan(fmt.Sprintf("%-8s", human.Bytes(s.PhotoSize))), a.green(miniSizeBar(photoPct, 20)))
		fmt.Fprintf(a.out, "Videos:      %d files · %s  %s\n",
			s.VideoFiles, a.cyan(fmt.Sprintf("%-8s", human.Bytes(s.VideoSize))), a.yellow(miniSizeBar(videoPct, 20)))
	} else {
		fmt.Fprintf(a.out, "Photos:      %d · %s\n", s.PhotoFiles, a.cyan(human.Bytes(s.PhotoSize)))
		fmt.Fprintf(a.out, "Videos:      %d · %s\n", s.VideoFiles, a.cyan(human.Bytes(s.VideoSize)))
	}
	fmt.Fprintf(a.out, "Total:       %d files · %s\n", s.TotalFiles, a.cyan(human.Bytes(s.TotalSize)))
	if largeThan != "" {
		fmt.Fprintf(a.out, "Large:       %d files · %s (>%s)\n", s.LargeFiles, a.cyan(human.Bytes(s.LargeSize)), largeThan)
	}
	if oldAgeRaw != "" {
		fmt.Fprintf(a.out, "Old:         %d files · %s (>%s)\n", s.OldFiles, a.cyan(human.Bytes(s.OldSize)), oldAgeRaw)
	}
	if s.ScanSkipped > 0 {
		fmt.Fprintf(a.out, "Skipped:     %d unreadable entries\n", s.ScanSkipped)
	}

	// iCloud placeholder warning
	printCloudPlaceholderWarning(a.out, result.Items, a.yellow, a.dim)

	// Folder breakdown
	if len(result.Items) > 0 {
		a.printScanByFolder(result.Items, s.TotalSize)
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
	printCloudPlaceholderWarning(a.out, result.Items, a.yellow, a.dim)
	return ExitSuccess
}

// printCloudPlaceholderWarning prints a one-line iCloud placeholder warning when
// the scan result contains files that look like iCloud-optimized thumbnails.
func printCloudPlaceholderWarning(w io.Writer, items []media.Item, yellow, dim func(string) string) {
	var count int
	var size int64
	for _, item := range items {
		if item.IsCloudPlaceholder {
			count++
			size += item.Size
		}
	}
	if count == 0 {
		return
	}
	fmt.Fprintf(w, "%s %d files (%s) look like iCloud-optimized thumbnails, not originals.\n",
		yellow("⚠ iCloud:"), count, human.Bytes(size))
	fmt.Fprintf(w, "  %s\n", dim("Use `imole icloud --to <dir>` to download full-resolution originals from iCloud."))
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

// printScanByFolder groups items by DCIM sub-folder and prints a size breakdown.
func (a *App) printScanByFolder(items []media.Item, totalSize int64) {
	type folderStat struct {
		name  string
		size  int64
		count int
	}

	folderMap := make(map[string]*folderStat)
	for _, item := range items {
		folder := item.RelPath
		if idx := strings.Index(folder, "/"); idx >= 0 {
			folder = folder[:idx]
		}
		if folder == "" {
			folder = "."
		}
		if _, ok := folderMap[folder]; !ok {
			folderMap[folder] = &folderStat{name: folder}
		}
		folderMap[folder].size += item.Size
		folderMap[folder].count++
	}

	// Sort folders by size descending
	folders := make([]*folderStat, 0, len(folderMap))
	for _, f := range folderMap {
		folders = append(folders, f)
	}
	sort.Slice(folders, func(i, j int) bool {
		return folders[i].size > folders[j].size
	})

	const maxFolders = 12
	if len(folders) <= 1 {
		return
	}

	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("By folder:"))
	shown := folders
	if len(shown) > maxFolders {
		shown = shown[:maxFolders]
	}
	for _, f := range shown {
		pct := 0.0
		if totalSize > 0 {
			pct = float64(f.size) * 100 / float64(totalSize)
		}
		bar := miniSizeBar(pct, 18)
		color := a.cyan
		if f.size > totalSize/4 {
			color = a.yellow
		}
		fmt.Fprintf(a.out, "  %-14s %s  %s  (%d files)\n",
			f.name,
			color(fmt.Sprintf("%-8s", human.Bytes(f.size))),
			a.dim(bar),
			f.count,
		)
	}
	if len(folders) > maxFolders {
		fmt.Fprintf(a.out, a.dim("  … %d more folders\n"), len(folders)-maxFolders)
	}
}

// printMetaFilterReport shows a concise result for metadata-filtered scans.
func (a *App) printMetaFilterReport(items []media.Item, f filter.Filter) int {
	var totalSize int64
	for _, it := range items {
		totalSize += it.Size
	}

	fmt.Fprintln(a.out, a.bold("iMole Metadata Filter Results"))
	fmt.Fprintln(a.out)

	if len(items) == 0 {
		if f.Country != "" {
			fmt.Fprintf(a.out, "  No items matched country filter %q\n", f.Country)
			fmt.Fprintln(a.out)
			fmt.Fprintln(a.out, a.dim("  Note: GPS metadata is only available when scanning a USB-connected iPhone."))
			fmt.Fprintln(a.out, a.dim("  Use: imole scan --with-meta --country "+f.Country+" (without --source)"))
		} else {
			fmt.Fprintln(a.out, "  No items matched the specified metadata filters.")
		}
		return ExitSuccess
	}

	fmt.Fprintf(a.out, "Matched:  %d files · %s\n", len(items), a.cyan(human.Bytes(totalSize)))
	if f.Country != "" {
		fmt.Fprintf(a.out, "Country:  %s\n", a.cyan(f.Country))
	}
	if f.NoGPS {
		fmt.Fprintln(a.out, "GPS:      no GPS only")
	}
	if !f.TakenAfter.IsZero() || !f.TakenBefore.IsZero() {
		after := "any"
		before := "any"
		if !f.TakenAfter.IsZero() {
			after = f.TakenAfter.Format("2006-01-02")
		}
		if !f.TakenBefore.IsZero() {
			before = f.TakenBefore.Format("2006-01-02")
		}
		fmt.Fprintf(a.out, "Taken:    %s → %s\n", after, before)
	}
	fmt.Fprintln(a.out)
	if len(items) > 0 {
		fmt.Fprintln(a.out, a.bold("Top files:"))
		limit := 20
		if len(items) < limit {
			limit = len(items)
		}
		for i, it := range items[:limit] {
			loc := ""
			if it.Country != "" {
				loc = "  " + a.dim(it.Country)
			}
			takenStr := ""
			if !it.TakenAt.IsZero() {
				takenStr = "  " + a.dim(it.TakenAt.Format("2006-01-02"))
			}
			fmt.Fprintf(a.out, "  %3d. %-30s %s%s%s\n",
				i+1, it.Name, a.cyan(human.Bytes(it.Size)), takenStr, loc)
		}
		if len(items) > 20 {
			fmt.Fprintf(a.out, a.dim("  … %d more\n"), len(items)-20)
		}
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("Next step:"))
	fmt.Fprintln(a.out, a.dim("  imole backup --to ~/iphone-backup "+buildMetaFlagStr(f)))
	return ExitSuccess
}

func buildMetaFlagStr(f filter.Filter) string {
	var parts []string
	if f.Country != "" {
		parts = append(parts, "--country "+f.Country)
	}
	if f.NoGPS {
		parts = append(parts, "--no-gps")
	}
	if !f.TakenAfter.IsZero() {
		parts = append(parts, "--taken-after "+f.TakenAfter.Format("2006-01-02"))
	}
	if !f.TakenBefore.IsZero() {
		parts = append(parts, "--taken-before "+f.TakenBefore.Format("2006-01-02"))
	}
	if f.DurationGt > 0 {
		parts = append(parts, fmt.Sprintf("--duration-gt %.0f", f.DurationGt))
	}
	if f.Only != "all" {
		parts = append(parts, "--only "+string(f.Only))
	}
	return strings.Join(parts, " ")
}

// miniSizeBar returns a compact ASCII bar of the given percentage (0–100).
func miniSizeBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int(pct / 100 * float64(width))
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

package cli

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/apps"
	"github.com/chenhg5/imole/internal/device"
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
	f, err := parseFilter(only, oldAgeRaw, largeThan, nil)
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
		stopSpinner := a.startSpinner("Scanning device… (may take ~15 s for USB)")
		result, err = scanFromFlags(ctx, providerName, source, f.LargeThan, f.OlderThan)
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
			stopSpinner(fmt.Sprintf("Device ready: %s  (%d files · %s)", root, result.Summary.TotalFiles, human.Bytes(result.Summary.TotalSize)))
			a.debug("Scan complete: %d items from %s", len(result.Items), root)
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
		Device    device.Info   `json:"device"`
		TopVideo  *mediaTop     `json:"top_video,omitempty"`
		Generated string        `json:"generated"`
	}

	deviceInfo := device.Check(ctx).Device

	stopSpinner := a.startSpinner("Scanning media…")
	mediaResult, mediaErr := scanFromFlags(ctx, "auto", "", 0, 0)
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

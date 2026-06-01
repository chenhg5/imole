package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenhg5/imole/internal/backup"
	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
	"github.com/chenhg5/imole/internal/provider"
)

func (a *App) runBackup(ctx context.Context, args []string) int {
	// Handle --help before flag parsing to avoid "flag: help requested" error
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			a.runBackupHelp()
			return ExitSuccess
		}
	}

	var providerName, source, to, only, olderThan, largeThan, fields, layout string
	var dryRun, jsonMode, yes, withMeta bool
	var limit int
	var files stringList
	var mf metaFlags
	fs := flagSet("backup")
	addProviderFlags(fs, &providerName, &source)
	fs.StringVar(&to, "to", "", "backup destination (local path or rclone:remote:path)")
	fs.StringVar(&layout, "layout", "", `destination layout template, e.g. "{year}/{month}/{type}/{filename}"`)
	fs.Var(&files, "file", "back up a specific rel_path from scan output; repeat for multiple files")
	fs.IntVar(&limit, "limit", 0, "back up at most N files (largest first); 0 = no limit")
	fs.BoolVar(&dryRun, "dry-run", false, "preview backup without copying")
	fs.BoolVar(&yes, "yes", false, "skip interactive confirmation prompt")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	addFilterFlags(fs, &only, &olderThan, &largeThan)
	addMetaFilterFlags(fs, &mf)
	fs.BoolVar(&withMeta, "with-meta", false, "fetch EXIF metadata to enable GPS/date/country filtering")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	if to == "" {
		a.printError(usageError("backup requires --to PATH"))
		return ExitUsage
	}
	// Auto-enable --with-meta if any metadata filter is specified.
	if mf.country != "" || mf.noGPS || mf.takenAfter != "" || mf.takenBefore != "" || mf.durationGt > 0 {
		withMeta = true
	}
	f, err := parseFilterMeta(only, olderThan, largeThan, files, mf)
	if err != nil {
		a.printError(&Error{
			Code:       "usage_error",
			Message:    err.Error(),
			Suggestion: "Use --only photos|videos, --older-than 90d|6m|1y, --large-than 500MB|1GB",
			Retryable:  false,
		})
		return ExitUsage
	}
	largeThreshold := f.LargeThan
	spinMsg := "Scanning device…"
	if withMeta {
		spinMsg = "Scanning with metadata (GPS, date)… ~60 s first run, cached 7 days"
	}
	stopSpinner := a.startSpinner(spinMsg)
	result, err := scanFromFlags(ctx, providerName, source, largeThreshold, f.OlderThan, withMeta)
	if err != nil {
		stopSpinner("")
		a.printError(runtimeError("scan_failed", err.Error(), "", true))
		return ExitError
	}
	stopSpinner(fmt.Sprintf("Scan complete: %d files · %s", result.Summary.TotalFiles, human.Bytes(result.Summary.TotalSize)))

	// Apply filter then optional limit (sorted by size desc).
	selectedItems := provider.FilteredItems(result, f)
	if limit > 0 && len(selectedItems) > limit {
		selectedItems = selectedItems[:limit]
	}
	selectedCount := len(selectedItems)
	var selectedSize int64
	for _, item := range selectedItems {
		selectedSize += item.Size
	}

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run: preview backup to %s\n", to)
	} else if !yes && a.isTTY && selectedCount > 0 {
		// Interactive confirmation before starting the actual copy
		fmt.Fprintln(a.out)
		fmt.Fprintf(a.out, "Ready to back up %s to %s\n",
			a.cyan(fmt.Sprintf("%d files · %s", selectedCount, human.Bytes(selectedSize))),
			a.cyan(to))
		fmt.Fprintf(a.out, "Proceed? [y/N] ")
		reader := bufio.NewReader(a.in)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if answer != "y" && answer != "Y" {
			fmt.Fprintln(a.out, "Cancelled.")
			return ExitSuccess
		}
		fmt.Fprintln(a.out)
	}

	if selectedCount == 0 && !dryRun {
		fmt.Fprintln(a.out, "No files match the filter — nothing to back up.")
		if len(result.Items) > 0 {
			printLargestHint(a.out, result.Items, f)
		}
		return ExitSuccess
	}

	a.status(fmt.Sprintf("Backing up %d files to %s…", selectedCount, to))
	var manifest backup.Manifest

	// Detect rclone destination: --to rclone:remote:path
	localDest := to
	rcloneDest := ""
	if isRcloneDest(to) {
		rcloneDest = strings.TrimPrefix(to, "rclone:")
		localDest = rcloneLocalCache(rcloneDest)
		a.debug("rclone mode: local staging at %s → %s", localDest, rcloneDest)
	}

	if source == "" && (providerName == string(provider.ImageCapture) || providerName == string(provider.Auto)) {
		manifest, err = a.runProviderBackup(ctx, result, localDest, f, layout, provider.Name(providerName), dryRun)
	} else {
		manifest, err = backup.Run(ctx, result, backup.Options{Destination: localDest, Filter: f, Layout: layout, DryRun: dryRun})
	}
	if err != nil {
		a.printError(runtimeError("backup_failed", err.Error(), "", false))
		return ExitError
	}

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run complete: %d files would be copied (exit 10)\n", manifest.Summary.SelectedFiles)
		return ExitDryRun
	}

	// Sync to rclone if requested
	if rcloneDest != "" && !dryRun {
		if code := a.runRcloneSync(ctx, localDest, rcloneDest); code != ExitSuccess {
			return code
		}
	}

	if a.shouldJSON() || jsonMode {
		return a.outputJSON(manifest, fields)
	}
	// Record the operation before printing so the log is written even if output fails.
	history.Append(history.Entry{
		Kind:        history.KindBackup,
		Files:       manifest.Summary.CopiedFiles,
		Size:        manifest.Summary.CopiedSize,
		Destination: absPath(to),
		Failed:      manifest.Summary.FailedFiles,
	})

	fmt.Fprintln(a.out, a.bold("Backup complete"))
	fmt.Fprintf(a.out, "Destination: %s\n", a.cyan(absPath(to)))
	fmt.Fprintf(a.out, "Selected:    %d files · %s\n", manifest.Summary.SelectedFiles, a.cyan(human.Bytes(manifest.Summary.SelectedSize)))
	if manifest.Summary.SelectedFiles == 0 && len(result.Items) > 0 {
		printLargestHint(a.out, result.Items, f)
	}
	if manifest.Summary.SelectedFiles > 0 {
		printBackupCandidates(a.out, manifest, 20)
	}
	fmt.Fprintf(a.out, "%s %d files · %s\n", a.green("Copied:     "), manifest.Summary.CopiedFiles, a.cyan(human.Bytes(manifest.Summary.CopiedSize)))
	fmt.Fprintf(a.out, "%s %d files · %s\n", a.green("Verified:   "), manifest.Summary.VerifiedFiles, a.cyan(human.Bytes(manifest.Summary.VerifiedSize)))
	if manifest.Summary.FailedFiles > 0 {
		fmt.Fprintf(a.out, "%s %d files\n", a.red("Failed:     "), manifest.Summary.FailedFiles)
		printFirstErrors(a.out, manifest)
	}
	fmt.Fprintf(a.out, "Manifest:    %s\n", a.dim(absPath(to)+"/"+backup.ManifestName))
	return ExitSuccess
}

func printBackupCandidates(w io.Writer, manifest backup.Manifest, limit int) {
	if limit <= 0 {
		limit = 20
	}
	fmt.Fprintln(w, "Candidates:")
	for i, file := range manifest.Files {
		if i >= limit {
			fmt.Fprintf(w, "  ... %d more\n", len(manifest.Files)-limit)
			return
		}
		fmt.Fprintf(w, "  %2d. %-24s %8s  %s -> %s\n",
			i+1,
			fileName(file.SourceRel),
			human.Bytes(file.Size),
			file.ModTime.Format("2006-01-02"),
			file.DestRel,
		)
	}
}

func fileName(path string) string {
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

func printFirstErrors(w io.Writer, manifest backup.Manifest) {
	printed := 0
	for _, file := range manifest.Files {
		if file.Error == "" {
			continue
		}
		fmt.Fprintf(w, "  - %s: %s\n", file.SourceRel, file.Error)
		printed++
		if printed >= 3 {
			return
		}
	}
}

func printLargestHint(w io.Writer, items []media.Item, f filter.Filter) {
	for _, item := range items {
		if f.Only == filter.KindVideos && !item.IsVideo() {
			continue
		}
		if f.Only == filter.KindPhotos && !item.IsPhoto() {
			continue
		}
		fmt.Fprintf(w, "Largest candidate: %s · %s\n", item.Name, human.Bytes(item.Size))
		if f.LargeThan > 0 {
			fmt.Fprintf(w, "Tip: lower --large-than below %s, for example --large-than 300MB.\n", human.Bytes(item.Size))
		}
		return
	}
}

func (a *App) runProviderBackup(ctx context.Context, result media.Result, to string, f filter.Filter, layout string, providerName provider.Name, dryRun bool) (backup.Manifest, error) {
	manifest := backup.Manifest{
		Version:   1,
		CreatedAt: f.Now,
		Root:      result.Summary.Root,
	}
	var requests []provider.DownloadRequest
	for _, item := range result.Items {
		if !f.Match(item) {
			continue
		}
		destRel := backup.DestinationRel(item, layout)
		manifest.Summary.SelectedFiles++
		manifest.Summary.SelectedSize += item.Size
		manifest.Files = append(manifest.Files, backup.ManifestFile{
			SourceRel: item.RelPath,
			DestRel:   destRel,
			Kind:      item.Kind,
			Size:      item.Size,
			ModTime:   item.ModTime,
		})
		requests = append(requests, provider.DownloadRequest{Item: item, DestRel: destRel})
	}
	if dryRun {
		return manifest, nil
	}
	if err := os.MkdirAll(to, 0o755); err != nil {
		return manifest, err
	}
	fmt.Fprintf(a.err, "Downloading %d files with %s provider...\n", len(requests), providerName)
	results, err := provider.Download(ctx, providerName, requests, to)
	if err != nil {
		return manifest, err
	}
	resultBySource := make(map[string]provider.DownloadResult, len(results))
	for _, result := range results {
		resultBySource[result.SourceRel] = result
	}
	for i := range manifest.Files {
		file := &manifest.Files[i]
		if dl, ok := resultBySource[file.SourceRel]; ok {
			file.Verified = dl.Verified
			file.Skipped = dl.Skipped
			file.Error = dl.Error
			if dl.Error != "" {
				manifest.Summary.FailedFiles++
				continue
			}
			if dl.Skipped {
				manifest.Summary.SkippedFiles++
			} else {
				manifest.Summary.CopiedFiles++
				manifest.Summary.CopiedSize += file.Size
			}
			if dl.Verified {
				manifest.Summary.VerifiedFiles++
				manifest.Summary.VerifiedSize += file.Size
			}
		}
	}
	if err := backup.WriteManifest(filepath.Join(to, backup.ManifestName), manifest); err != nil {
		return manifest, err
	}
	return manifest, nil
}

// isRcloneDest reports whether to is a rclone remote path (rclone:remote:path).
func isRcloneDest(to string) bool {
	return strings.HasPrefix(to, "rclone:")
}

// rcloneLocalCache returns a local staging directory for a rclone destination.
func rcloneLocalCache(rcloneDest string) string {
	home, _ := os.UserHomeDir()
	safe := strings.NewReplacer(":", "_", "/", "_", "\\", "_").Replace(rcloneDest)
	return filepath.Join(home, ".imole", "rclone-cache", safe)
}

// runRcloneSync runs `rclone copy src rcloneDest` after a local backup.
func (a *App) runRcloneSync(ctx context.Context, localDir, rcloneDest string) int {
	rclone, err := exec.LookPath("rclone")
	if err != nil {
		a.printError(runtimeError("rclone_not_found",
			"rclone not found — cannot sync to remote",
			"Install rclone: https://rclone.org/install/ then run: imole backup again",
			false))
		return ExitError
	}
	a.status(fmt.Sprintf("Syncing to %s via rclone…", rcloneDest))
	cmd := exec.CommandContext(ctx, rclone, "copy", "--progress", localDir, rcloneDest)
	cmd.Stdout = a.err
	cmd.Stderr = a.err
	if err := cmd.Run(); err != nil {
		a.printError(runtimeError("rclone_sync_failed", err.Error(),
			fmt.Sprintf("Run manually: rclone copy %q %q", localDir, rcloneDest), false))
		return ExitError
	}
	fmt.Fprintf(a.out, "%s Synced to %s\n", a.green("✓"), a.cyan(rcloneDest))
	fmt.Fprintf(a.out, "  Local staging: %s\n", a.dim(localDir))
	return ExitSuccess
}

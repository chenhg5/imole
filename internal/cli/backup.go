package cli

import (
	"context"
	"fmt"
	"io"
	"os"
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

	var providerName, source, to, only, olderThan, largeThan, fields string
	var dryRun, jsonMode bool
	var files stringList
	fs := flagSet("backup")
	addProviderFlags(fs, &providerName, &source)
	fs.StringVar(&to, "to", "", "backup destination")
	fs.Var(&files, "file", "back up a specific rel_path from scan output; repeat for multiple files")
	fs.BoolVar(&dryRun, "dry-run", false, "preview backup without copying")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	addFilterFlags(fs, &only, &olderThan, &largeThan)
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	if to == "" {
		a.printError(usageError("backup requires --to PATH"))
		return ExitUsage
	}
	f, err := parseFilter(only, olderThan, largeThan, files)
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
	a.status("Scanning device…")
	result, err := scanFromFlags(ctx, providerName, source, largeThreshold, f.OlderThan)
	if err != nil {
		a.printError(runtimeError("scan_failed", err.Error(), "", true))
		return ExitError
	}

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run: preview backup to %s\n", to)
	}

	a.status(fmt.Sprintf("Backing up %d selected files to %s…", result.Summary.TotalFiles, to))
	var manifest backup.Manifest
	if source == "" && (providerName == string(provider.ImageCapture) || providerName == string(provider.Auto)) {
		manifest, err = a.runProviderBackup(ctx, result, to, f, provider.Name(providerName), dryRun)
	} else {
		manifest, err = backup.Run(ctx, result, backup.Options{Destination: to, Filter: f, DryRun: dryRun})
	}
	if err != nil {
		a.printError(runtimeError("backup_failed", err.Error(), "", false))
		return ExitError
	}

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run complete: %d files would be copied (exit 10)\n", manifest.Summary.SelectedFiles)
		return ExitDryRun
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

func (a *App) runProviderBackup(ctx context.Context, result media.Result, to string, f filter.Filter, providerName provider.Name, dryRun bool) (backup.Manifest, error) {
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
		destRel := backup.DestinationRel(item)
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

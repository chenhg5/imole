package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/chenhg5/imole/internal/backup"
	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/provider"
)

func (a *App) runClean(ctx context.Context, args []string) int {
	var manifestPath, providerName, sourcePath string
	var dryRun, yes bool
	fs := flagSet("clean")
	fs.StringVar(&manifestPath, "manifest", "", "path to manifest.json from a previous backup")
	fs.StringVar(&providerName, "provider", "auto", "provider to use for deletion (auto|imagecapture|filesystem)")
	fs.StringVar(&sourcePath, "source", "", "mount point of the iPhone DCIM directory (Linux/Windows: ifuse or iTunes mount)")
	fs.BoolVar(&dryRun, "dry-run", false, "show what would be deleted without deleting")
	fs.BoolVar(&yes, "yes", false, "skip confirmation prompt")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	if manifestPath == "" {
		return a.runCleanGuide()
	}
	return a.runCleanFromManifest(ctx, manifestPath, provider.Name(providerName), sourcePath, dryRun, yes)
}

// runCleanGuide prints the safe cleanup flow when no manifest is provided.
func (a *App) runCleanGuide() int {
	fmt.Fprintln(a.out, "Safe cleanup mode")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Recommended flow (macOS, USB):")
	fmt.Fprintln(a.out, "  1. imole scan")
	fmt.Fprintln(a.out, "  2. imole backup --to /path/to/backup --only videos --older-than 90d")
	fmt.Fprintln(a.out, "  3. imole report --manifest /path/to/backup/manifest.json")
	fmt.Fprintln(a.out, "  4. imole clean --manifest /path/to/backup/manifest.json")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Recommended flow (Linux, ifuse mount):")
	fmt.Fprintln(a.out, "  1. sudo apt install ifuse libimobiledevice-utils")
	fmt.Fprintln(a.out, "  2. idevicepair pair && mkdir -p ~/iphone && ifuse ~/iphone")
	fmt.Fprintln(a.out, "  3. imole backup --source ~/iphone/DCIM --to /path/to/backup --only videos")
	fmt.Fprintln(a.out, "  4. imole clean --manifest /path/to/backup/manifest.json --source ~/iphone/DCIM")
	fmt.Fprintln(a.out, "  5. fusermount -u ~/iphone")
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, `Recommended flow (Windows, iTunes mount):`)
	fmt.Fprintln(a.out, "  1. Install iTunes and connect iPhone; unlock and tap Trust")
	fmt.Fprintln(a.out, `  2. Open Windows Explorer → This PC → [iPhone] → Internal Storage → DCIM`)
	fmt.Fprintln(a.out, `  3. imole backup --source "\\Apple\iPhone\Internal Storage\DCIM" --to C:\backup`)
	fmt.Fprintln(a.out, `  4. imole clean  --manifest C:\backup\manifest.json --source "\\Apple\iPhone\Internal Storage\DCIM"`)
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Tip: use --dry-run to preview what would be deleted before committing.")
	return ExitSuccess
}

// runCleanFromManifest reads a backup manifest, finds verified files, and
// deletes them from the iPhone. On macOS it uses ImageCaptureCore (USB/PTP);
// on Linux/Windows pass --source with the ifuse/iTunes mount point to use
// filesystem deletion (os.Remove), which frees space immediately.
func (a *App) runCleanFromManifest(ctx context.Context, manifestPath string, providerName provider.Name, sourcePath string, dryRun, yes bool) int {
	manifest, err := backup.ReadManifest(manifestPath)
	if err != nil {
		a.printError(runtimeError("manifest_read_failed", err.Error(),
			"Run: imole backup first, then point --manifest to the generated manifest.json", false))
		return ExitError
	}

	// Collect only verified files — we never touch unverified backups.
	var requests []provider.DeleteRequest
	var totalSize int64
	for _, f := range manifest.Files {
		if f.Verified && f.Error == "" {
			requests = append(requests, provider.DeleteRequest{Path: f.SourceRel})
			totalSize += f.Size
		}
	}

	if len(requests) == 0 {
		fmt.Fprintln(a.out, "No verified files found in manifest — nothing to delete.")
		fmt.Fprintln(a.out, "Run: imole backup to create a verified backup first.")
		return ExitSuccess
	}

	// Determine whether we are using filesystem (mount) or USB provider.
	usingFilesystem := sourcePath != "" || providerName == provider.Filesystem

	// Show deletion plan.
	fmt.Fprintln(a.out, "Clean plan")
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Manifest:       %s\n", absPath(manifestPath))
	fmt.Fprintf(a.out, "Verified files: %d (%s)\n", len(requests), human.Bytes(totalSize))
	if usingFilesystem {
		fmt.Fprintf(a.out, "Source mount:   %s\n", sourcePath)
	}
	fmt.Fprintln(a.out)

	printCleanCandidates(a.out, manifest, 15)

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run: %d files (%s) would be deleted from iPhone.\n", len(requests), human.Bytes(totalSize))
		return ExitDryRun
	}

	fmt.Fprintln(a.out, "Warning: This will delete the files listed above from your iPhone.")
	fmt.Fprintln(a.out, "         iMole only deletes files verified in the manifest.")
	if usingFilesystem {
		fmt.Fprintln(a.out, "         Deletion is from the mounted filesystem — space is freed immediately.")
	} else {
		fmt.Fprintln(a.out, "         Files will remain in Recently Deleted for 30 days (no space freed until cleared).")
	}
	fmt.Fprintln(a.out)

	if !yes && !a.confirm("Proceed with deletion?") {
		fmt.Fprintln(a.out, "Aborted.")
		return ExitSuccess
	}

	fmt.Fprintf(a.err, "Deleting %d files via %s provider...\n", len(requests), providerName)
	if !usingFilesystem {
		fmt.Fprintln(a.err, "Note: your iPhone may show a confirmation prompt — accept it to allow deletion.")
	}
	fmt.Fprintln(a.err)

	results, err := provider.Delete(ctx, providerName, sourcePath, requests)
	if err != nil {
		hint := "Make sure iPhone is connected, unlocked, and trusted."
		if usingFilesystem {
			hint = fmt.Sprintf("Check that the mount point exists and is readable: %s", sourcePath)
		}
		a.printError(runtimeError("delete_failed", err.Error(), hint, false))
		return ExitError
	}

	var deleted, failed int
	var deletedSize int64
	sizeByPath := make(map[string]int64, len(manifest.Files))
	for _, f := range manifest.Files {
		sizeByPath[f.SourceRel] = f.Size
	}
	for _, r := range results {
		if r.Deleted {
			deleted++
			deletedSize += sizeByPath[r.Path]
		} else {
			failed++
		}
	}

	history.Append(history.Entry{
		Kind:         history.KindClean,
		Files:        deleted,
		Size:         deletedSize,
		ManifestPath: absPath(manifestPath),
		Failed:       failed,
	})

	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Delete complete")
	fmt.Fprintf(a.out, "  Deleted: %d files · %s\n", deleted, human.Bytes(deletedSize))
	if failed > 0 {
		fmt.Fprintf(a.out, "  Failed:  %d files\n", failed)
		printDeleteErrors(a.out, results, 5)
	}
	fmt.Fprintln(a.out)
	if usingFilesystem {
		fmt.Fprintf(a.out, "Space freed: ~%s (filesystem deletion, immediate)\n", human.Bytes(deletedSize))
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, "Next step:")
		if runtime.GOOS == "windows" {
			fmt.Fprintln(a.out, "  Safely eject the iPhone from Windows Explorer.")
		} else {
			fmt.Fprintf(a.out, "  Unmount the iPhone: fusermount -u %s\n", sourcePath)
		}
	} else {
		fmt.Fprintln(a.out, "Space freed so far: 0  (files are in Recently Deleted)")
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, "Final step to reclaim space:")
		fmt.Fprintln(a.out, "  On iPhone → Photos → Albums → Recently Deleted → Delete All")
		fmt.Fprintf(a.out, "  Estimated space freed after that step: ~%s\n", human.Bytes(deletedSize))
	}

	if failed > 0 {
		return ExitError
	}
	return ExitSuccess
}

func (a *App) confirm(prompt string) bool {
	fmt.Fprintf(a.out, "%s [y/N] ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes"
	}
	return false
}

func printCleanCandidates(w io.Writer, manifest backup.Manifest, limit int) {
	type candidate struct {
		path string
		size int64
	}
	var candidates []candidate
	for _, f := range manifest.Files {
		if f.Verified && f.Error == "" {
			candidates = append(candidates, candidate{path: f.SourceRel, size: f.Size})
		}
	}
	if len(candidates) == 0 {
		return
	}
	n := limit
	if n > len(candidates) {
		n = len(candidates)
	}
	fmt.Fprintf(w, "Files to delete (showing %d of %d):\n", n, len(candidates))
	for i := 0; i < n; i++ {
		c := candidates[i]
		fmt.Fprintf(w, "  %3d. %-40s %s\n", i+1, fileName(c.path), human.Bytes(c.size))
	}
	if len(candidates) > n {
		fmt.Fprintf(w, "  ... %d more\n", len(candidates)-n)
	}
	fmt.Fprintln(w)
}

func printDeleteErrors(w io.Writer, results []provider.DeleteResult, limit int) {
	printed := 0
	for _, r := range results {
		if r.Deleted || r.Error == "" {
			continue
		}
		fmt.Fprintf(w, "  - %s: %s\n", fileName(r.Path), r.Error)
		printed++
		if printed >= limit {
			return
		}
	}
}

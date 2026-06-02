package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/media"
)

// runICloud wraps icloudpd to download full-resolution originals from iCloud
// Photos, bypassing the device entirely.  This is the correct solution when
// the iPhone has "Optimize iPhone Storage" enabled and stores only low-res
// thumbnails locally.
//
// Usage:
//
//	imole icloud --to <dir> [--username email] [--album "All Photos"]
//	             [--recent N] [--from-scan FILE|-] [--dry-run] [--no-progress]
func (a *App) runICloud(ctx context.Context, args []string) int {
	fs := flagSet("icloud")
	var to, username, album, password, fromScan string
	var recent int
	var dryRun, noProgress, forceSize bool
	fs.StringVar(&to, "to", "", "destination directory for downloaded photos/videos")
	fs.StringVar(&username, "username", "", "Apple ID email address")
	fs.StringVar(&username, "u", "", "Apple ID (short)")
	fs.StringVar(&album, "album", "All Photos", "iCloud album to download")
	fs.StringVar(&password, "password", "", "App-specific password (or set ICLOUD_PASSWORD env var)")
	fs.IntVar(&recent, "recent", 0, "download only the N most recently added items (0 = all)")
	fs.StringVar(&fromScan, "from-scan", "", "path to JSON from `imole scan --only-placeholders --json`, or - for stdin; narrows download to the date range of those files")
	fs.BoolVar(&dryRun, "dry-run", false, "list what would be downloaded without downloading")
	fs.BoolVar(&noProgress, "no-progress", false, "disable icloudpd progress output")
	fs.BoolVar(&forceSize, "force-size", false, "skip size check and always re-download")

	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	// Validate required flags.
	if to == "" {
		a.printError(usageError("icloud: --to is required\n\nUsage:\n  imole icloud --to <dir> [--username email] [--from-scan FILE|-]"))
		return ExitUsage
	}

	// Load placeholder scan results if --from-scan was provided.
	var placeholders []media.Item
	var fromDate, toDate time.Time
	if fromScan != "" {
		var err error
		placeholders, err = loadScanJSON(fromScan, a.in)
		if err != nil {
			a.printError(runtimeError("icloud_scan_read", err.Error(),
				"Make sure to pipe: imole scan --only-placeholders --json | imole icloud --to <dir> --from-scan -", false))
			return ExitError
		}
		if len(placeholders) == 0 {
			fmt.Fprintln(a.out, a.green("✓")+" No placeholder files in scan result — nothing to fetch from iCloud.")
			return ExitSuccess
		}
		fromDate, toDate = dateRangeFromItems(placeholders)
		fmt.Fprintf(a.out, "Loaded %s placeholder files from scan (%s … %s)\n",
			a.cyan(fmt.Sprintf("%d", len(placeholders))),
			a.dim(fromDate.Format("2006-01-02")),
			a.dim(toDate.Format("2006-01-02")))
	}

	// Check that icloudpd is available.
	icloudpd, err := exec.LookPath("icloudpd")
	if err != nil {
		a.printError(runtimeError(
			"icloudpd_not_found",
			"icloudpd is not installed",
			"Install with: pip3 install icloudpd\n"+
				"  or: brew install icloudpd\n"+
				"  Docs: https://github.com/icloud-photos-downloader/icloud_photos_downloader",
			false,
		))
		return ExitError
	}

	// Resolve destination.
	destAbs, err := filepath.Abs(to)
	if err != nil {
		a.printError(runtimeError("icloud_dest_error", err.Error(), "", false))
		return ExitError
	}
	if !dryRun {
		if err := os.MkdirAll(destAbs, 0o755); err != nil {
			a.printError(runtimeError("icloud_mkdir", err.Error(), "", false))
			return ExitError
		}
	}

	// Build icloudpd arguments.
	icloudArgs := []string{
		"--directory", destAbs,
		"--album", album,
	}
	if username != "" {
		icloudArgs = append(icloudArgs, "--username", username)
	}

	pw := password
	if pw == "" {
		pw = os.Getenv("ICLOUD_PASSWORD")
	}
	if pw != "" {
		icloudArgs = append(icloudArgs, "--password", pw)
	}

	// Apply date range from --from-scan if available; otherwise use --recent.
	if !fromDate.IsZero() {
		// Add 1-day buffer on each side for safety.
		icloudArgs = append(icloudArgs,
			"--from-date", fromDate.AddDate(0, 0, -1).Format("2006-01-02"),
			"--to-date", toDate.AddDate(0, 0, 1).Format("2006-01-02"),
		)
		fmt.Fprintf(a.out, "Date range: %s → %s (±1 day buffer)\n",
			a.cyan(fromDate.Format("2006-01-02")), a.cyan(toDate.Format("2006-01-02")))
	} else if recent > 0 {
		icloudArgs = append(icloudArgs, "--recent", fmt.Sprintf("%d", recent))
	}

	if dryRun {
		icloudArgs = append(icloudArgs, "--dry-run")
	}
	if noProgress {
		icloudArgs = append(icloudArgs, "--no-progress")
	}
	if forceSize {
		icloudArgs = append(icloudArgs, "--force-size")
	}

	// Always use the auto-delete=no policy — deletions must go through imole clean.
	icloudArgs = append(icloudArgs, "--auto-delete=no")

	a.debug("icloudpd command: %s %s", icloudpd, strings.Join(icloudArgs, " "))

	fmt.Fprintf(a.out, "%s Downloading from iCloud (album: %s)…\n",
		a.bold("iCloud"), a.cyan(album))
	fmt.Fprintf(a.out, "Destination: %s\n\n", a.cyan(destAbs))

	cmd := exec.CommandContext(ctx, icloudpd, icloudArgs...)
	cmd.Stdout = a.out
	cmd.Stderr = a.err
	cmd.Stdin = a.in

	if err := cmd.Run(); err != nil {
		exitErr, ok := err.(*exec.ExitError)
		if ok {
			// icloudpd exits 2 when 2FA is needed — surface a helpful message.
			if exitErr.ExitCode() == 2 {
				fmt.Fprintf(a.err, "\n%s\n", a.yellow("Two-factor authentication required."))
				fmt.Fprintf(a.err, "icloudpd will prompt for a 2FA code above.\n")
				fmt.Fprintf(a.err, "If non-interactive, generate an App-Specific Password at\n")
				fmt.Fprintf(a.err, "  https://appleid.apple.com/account/manage\n")
				fmt.Fprintf(a.err, "and pass it via --password or ICLOUD_PASSWORD env var.\n")
			}
		}
		a.printError(runtimeError("icloudpd_failed", err.Error(),
			fmt.Sprintf("Run manually: icloudpd --directory %q --album %q", destAbs, album), false))
		return ExitError
	}

	if !dryRun {
		history.Append(history.Entry{
			Kind:        history.KindBackup,
			Destination: destAbs,
		})
	}

	fmt.Fprintf(a.out, "\n%s iCloud download complete → %s\n", a.green("✓"), a.cyan(destAbs))

	// If we had a scan result, do a filename cross-reference to report matches.
	if len(placeholders) > 0 && !dryRun {
		printICloudMatchReport(a.out, placeholders, destAbs, a.green, a.yellow, a.dim)
	} else {
		fmt.Fprintf(a.out, "  %s\n", a.dim("Run `imole scan --source "+destAbs+" --json` to verify downloaded originals."))
	}
	return ExitSuccess
}

// loadScanJSON reads a JSON file or stdin (-) produced by
// `imole scan --only-placeholders --json` and returns the media items.
func loadScanJSON(path string, stdin io.Reader) ([]media.Item, error) {
	var r io.Reader
	if path == "-" {
		r = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("cannot open scan file %q: %w", path, err)
		}
		defer f.Close()
		r = f
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading scan JSON: %w", err)
	}

	// The JSON may be a filtered result wrapper {"filtered_count":N,"items":[...]}
	// or a plain array [...].
	type filteredWrapper struct {
		Items []media.Item `json:"items"`
	}
	var wrapper filteredWrapper
	if err := json.Unmarshal(raw, &wrapper); err == nil && wrapper.Items != nil {
		return wrapper.Items, nil
	}
	var items []media.Item
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("invalid scan JSON (expected array or {items:[...]}): %w", err)
	}
	return items, nil
}

// dateRangeFromItems returns the earliest and latest dates across items,
// using TakenAt when available, falling back to ModTime.
func dateRangeFromItems(items []media.Item) (from, to time.Time) {
	for _, item := range items {
		t := item.TakenAt
		if t.IsZero() {
			t = item.ModTime
		}
		if t.IsZero() {
			continue
		}
		if from.IsZero() || t.Before(from) {
			from = t
		}
		if to.IsZero() || t.After(to) {
			to = t
		}
	}
	return
}

// printICloudMatchReport walks destDir and checks which placeholder filenames
// are now present as full-size files, then prints a summary.
func printICloudMatchReport(w io.Writer, placeholders []media.Item, destDir string, green, yellow, dim func(string) string) {
	// Build a set of basenames present in destDir (recursively).
	downloaded := make(map[string]int64) // basename → size
	_ = filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, e := os.Stat(path)
		if e == nil {
			downloaded[filepath.Base(path)] = info.Size()
		}
		return nil
	})

	var matched, missing int
	var matchedSize, missingSize int64
	var missingNames []string

	for _, item := range placeholders {
		if dlSize, ok := downloaded[item.Name]; ok && dlSize > item.Size {
			// Found in destDir AND larger than the thumbnail — it's the original.
			matched++
			matchedSize += dlSize
		} else {
			missing++
			missingSize += item.Size
			if len(missingNames) < 5 {
				missingNames = append(missingNames, item.Name)
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "iCloud match report (%d placeholder files checked):\n", len(placeholders))
	fmt.Fprintf(w, "  %s %d originals downloaded (%s)\n",
		green("✓"), matched, human.Bytes(matchedSize))
	if missing > 0 {
		fmt.Fprintf(w, "  %s %d not found in download — may be in a different album or date range\n",
			yellow("⚠"), missing)
		for _, name := range missingNames {
			fmt.Fprintf(w, "      %s\n", dim(name))
		}
		if missing > 5 {
			fmt.Fprintf(w, "      %s\n", dim(fmt.Sprintf("… and %d more", missing-5)))
		}
		fmt.Fprintf(w, "  Tip: try a wider %s or %s to catch the missing files.\n",
			dim("--album"), dim("--from-date / --to-date"))
	}
	_ = missingSize
}

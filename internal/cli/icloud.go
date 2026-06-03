package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/icloud"
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
	var to, username, album, password, fromScan, domain string
	var recent int
	var dryRun, noProgress, forceSize bool
	fs.StringVar(&to, "to", "", "destination directory for downloaded photos/videos")
	fs.StringVar(&username, "username", "", "Apple ID email address")
	fs.StringVar(&username, "u", "", "Apple ID (short)")
	fs.StringVar(&album, "album", "", "iCloud album to download; leave empty to download all photos (default)")
	fs.StringVar(&password, "password", "", "App-specific password (or set ICLOUD_PASSWORD env var)")
	fs.StringVar(&domain, "domain", "", "iCloud domain: leave empty for international, use 'icloud.com.cn' for China-registered Apple IDs")
	fs.IntVar(&recent, "recent", 0, "download only the N most recently added items (0 = all)")
	fs.StringVar(&fromScan, "from-scan", "", "path to JSON from `imole scan --only-placeholders --json`, or - for stdin; downloads the date range into a staging dir and extracts only the matching filenames")
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

	// Resolve destination first (needed for both native and icloudpd paths).
	destAbs, err := filepath.Abs(to)
	if err != nil {
		a.printError(runtimeError("icloud_dest_error", err.Error(), "", false))
		return ExitError
	}

	// When --from-scan is provided, use the native Go implementation which
	// queries CloudKit directly by filename — no icloudpd dependency needed.
	if len(placeholders) > 0 {
		if username == "" {
			a.printError(usageError("icloud: --username is required with --from-scan"))
			return ExitUsage
		}
		// Normalise domain.
		domainNorm := "com"
		if strings.Contains(domain, "cn") || domain == "cn" {
			domainNorm = "cn"
		}
		return a.runICloudNative(ctx, placeholders, destAbs, username, password, domainNorm, dryRun)
	}

	// Check that icloudpd is available.  Also probe common pip --user install
	// locations that may not be on $PATH (e.g. ~/Library/Python/3.x/bin on macOS).
	icloudpd, err := findIcloudpd()
	if err != nil {
		a.printError(runtimeError(
			"icloudpd_not_found",
			"icloudpd is not installed",
			"Install with: pip3 install --user icloudpd\n"+
				"  or: brew install icloudpd\n"+
				"  Docs: https://github.com/icloud-photos-downloader/icloud_photos_downloader",
			false,
		))
		return ExitError
	}

	if !dryRun {
		if err := os.MkdirAll(destAbs, 0o755); err != nil {
			a.printError(runtimeError("icloud_mkdir", err.Error(), "", false))
			return ExitError
		}
	}

	// Normalize domain before building any args so all paths get it.
	if domain != "" {
		switch strings.ToLower(domain) {
		case "icloud.com.cn", "icloudchina", "china", "cn":
			domain = "cn"
		case "icloud.com", "com", "international":
			domain = "com"
		}
	}

	// Build shared icloudpd arguments (used by both the placeholder and normal paths).
	icloudArgs := []string{}
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
	if domain != "" {
		icloudArgs = append(icloudArgs, "--domain", domain)
	}
	if album != "" {
		icloudArgs = append(icloudArgs, "--album", album)
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

	if recent > 0 {
		icloudArgs = append(icloudArgs, "--recent", fmt.Sprintf("%d", recent))
	}

	// Do NOT pass --auto-delete; deletions must go through imole clean.
	// (icloudpd >=1.x treats --auto-delete as a boolean flag with no argument)

	// Prepend --directory (must come first so username sub-command works correctly).
	icloudArgs = append([]string{"--directory", destAbs}, icloudArgs...)

	a.debug("icloudpd command: %s %s", icloudpd, strings.Join(icloudArgs, " "))

	albumLabel := album
	if albumLabel == "" {
		albumLabel = "All Photos"
	}
	fmt.Fprintf(a.out, "%s Downloading from iCloud (album: %s)…\n",
		a.bold("iCloud"), a.cyan(albumLabel))
	fmt.Fprintf(a.out, "Destination: %s\n\n", a.cyan(destAbs))

	// Capture stderr to detect China-domain error and surface a clear hint.
	var stderrBuf strings.Builder
	cmd := exec.CommandContext(ctx, icloudpd, icloudArgs...) //nolint:gosec // G702: args from trusted flags
	cmd.Stdout = a.out
	cmd.Stderr = io.MultiWriter(a.err, &stderrBuf)
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
			fmt.Sprintf("Run manually: icloudpd --directory %q", destAbs), false))
		// Detect China-region accounts and surface actionable hint.
		if strings.Contains(stderrBuf.String(), "iCloud.com.cn") || strings.Contains(stderrBuf.String(), "icloud.com.cn") {
			fmt.Fprintf(a.err, "\n%s Your Apple ID is registered with China iCloud (iCloud.com.cn).\n", a.yellow("Tip:"))
			fmt.Fprintf(a.err, "  Re-run with:  %s\n",
				a.cyan(fmt.Sprintf("imole icloud --domain cn --to %q --username %s --recent 5", to, username)))
		}
		return ExitError
	}

	if !dryRun {
		history.Append(history.Entry{
			Kind:        history.KindBackup,
			Destination: destAbs,
		})
	}

	fmt.Fprintf(a.out, "\n%s iCloud download complete → %s\n", a.green("✓"), a.cyan(destAbs))

	fmt.Fprintf(a.out, "  %s\n", a.dim("Run `imole scan --source "+destAbs+" --json` to verify downloaded originals."))
	return ExitSuccess
}

// runICloudNative downloads exactly the target placeholder files from iCloud
// using our own Go implementation — no icloudpd required.
//
// Flow:
//  1. Login (reuse session cookie if valid, otherwise SRP + 2FA)
//  2. CloudKit query by filename for each placeholder
//  3. Download matched assets directly to destDir
func (a *App) runICloudNative(ctx context.Context, placeholders []media.Item, destDir, username, password, domain string, dryRun bool) int {
	if err := os.MkdirAll(destDir, 0o755); err != nil && !dryRun {
		a.printError(runtimeError("icloud_mkdir", err.Error(), "", false))
		return ExitError
	}

	// Collect target filenames.
	targets := make([]string, len(placeholders))
	for i, p := range placeholders {
		targets[i] = p.Name
	}
	fmt.Fprintf(a.out, "%s Connecting to iCloud (native Go)…\n", a.bold("iCloud"))
	fmt.Fprintf(a.out, "Searching for %s files by filename\n\n", a.cyan(fmt.Sprintf("%d", len(targets))))

	// Authenticate.
	client, err := icloud.Login(username, password, domain, true, icloud.StdinTwoFA)
	if err != nil {
		a.printError(runtimeError("icloud_auth", err.Error(),
			"Check your Apple ID, password, and --domain flag", false))
		return ExitError
	}
	fmt.Fprintf(a.out, "%s Authenticated\n", a.green("✓"))

	if dryRun {
		fmt.Fprintf(a.out, "%s Dry run: would search iCloud for %d files\n",
			a.cyan("→"), len(targets))
		return ExitSuccess
	}

	// Query CloudKit for each target filename.
	fmt.Fprintf(a.out, "Querying iCloud Photos library…\n")
	assets, err := client.QueryByFilenames(targets)
	if err != nil {
		a.printError(runtimeError("icloud_query", err.Error(), "", true))
		return ExitError
	}

	// Match query results back to placeholders.
	found := map[string]icloud.AssetRecord{}
	for _, asset := range assets {
		found[asset.Filename] = asset
	}
	var downloaded, missing int
	for _, name := range targets {
		asset, ok := found[name]
		if !ok {
			fmt.Fprintf(a.out, "  %s %s — not found in iCloud Photos\n", a.yellow("⚠"), name)
			missing++
			continue
		}
		fmt.Fprintf(a.out, "  Downloading %s (%s)…\n",
			a.cyan(name), human.Bytes(asset.FileSize))
		destPath, err := client.DownloadAsset(asset, destDir)
		if err != nil {
			fmt.Fprintf(a.out, "  %s Download failed for %s: %v\n", a.yellow("⚠"), name, err)
			missing++
			continue
		}
		fmt.Fprintf(a.out, "  %s → %s\n", a.green("✓"), destPath)
		downloaded++
	}

	history.Append(history.Entry{Kind: history.KindBackup, Destination: destDir})
	fmt.Fprintf(a.out, "\n%s %d/%d files downloaded → %s\n",
		a.green("✓"), downloaded, len(targets), a.cyan(destDir))
	if missing > 0 {
		fmt.Fprintf(a.out, "  %s %d not found — check if they exist in your iCloud Photo Library\n",
			a.yellow("⚠"), missing)
	}
	return ExitSuccess
}

// runICloudByFileWindows runs one icloudpd invocation per unique date found in
// placeholders, using a ±1 day window.  Each run downloads only that day's
// photos (~5-30 files) to a temp staging dir, then copies matching filenames to
// destAbs.  This is far more efficient than a single wide date-range run.

// extractMatchedFiles walks stagingDir looking for files whose basename appears
// in the placeholders list AND whose size is larger than the on-device thumbnail.
// Matched files are moved (or copied if cross-device) to destDir.
// Returns the count of successfully extracted files.

// moveOrCopy tries os.Rename first (fast, same filesystem); falls back to a
// copy+delete if the source and destination are on different filesystems.

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

// findIcloudpd looks for the icloudpd binary in $PATH and in common pip --user
// install locations that macOS pip does not add to $PATH by default.
func findIcloudpd() (string, error) {
	if p, err := exec.LookPath("icloudpd"); err == nil {
		return p, nil
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Library", "Python", "3.13", "bin", "icloudpd"),
		filepath.Join(home, "Library", "Python", "3.12", "bin", "icloudpd"),
		filepath.Join(home, "Library", "Python", "3.11", "bin", "icloudpd"),
		filepath.Join(home, "Library", "Python", "3.10", "bin", "icloudpd"),
		filepath.Join(home, "Library", "Python", "3.9", "bin", "icloudpd"),
		filepath.Join(home, ".local", "bin", "icloudpd"),
		"/opt/homebrew/bin/icloudpd",
		"/usr/local/bin/icloudpd",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, nil
		}
	}
	return "", fmt.Errorf("icloudpd not found")
}

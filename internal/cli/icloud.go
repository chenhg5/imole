package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chenhg5/imole/internal/history"
)

// runICloud wraps icloudpd to download full-resolution originals from iCloud
// Photos, bypassing the device entirely.  This is the correct solution when
// the iPhone has "Optimize iPhone Storage" enabled and stores only low-res
// thumbnails locally.
//
// Usage:
//
//	imole icloud --to <dir> [--username email] [--album "All Photos"]
//	             [--recent N] [--dry-run] [--no-progress]
func (a *App) runICloud(ctx context.Context, args []string) int {
	fs := flagSet("icloud")
	var to, username, album, password string
	var recent int
	var dryRun, noProgress, forceSize bool
	fs.StringVar(&to, "to", "", "destination directory for downloaded photos/videos")
	fs.StringVar(&username, "username", "", "Apple ID email address")
	fs.StringVar(&username, "u", "", "Apple ID (short)")
	fs.StringVar(&album, "album", "All Photos", "iCloud album to download")
	fs.StringVar(&password, "password", "", "App-specific password (or set ICLOUD_PASSWORD env var)")
	fs.IntVar(&recent, "recent", 0, "download only the N most recently added items (0 = all)")
	fs.BoolVar(&dryRun, "dry-run", false, "list what would be downloaded without downloading")
	fs.BoolVar(&noProgress, "no-progress", false, "disable icloudpd progress output")
	fs.BoolVar(&forceSize, "force-size", false, "skip size check and always re-download")

	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}

	// Validate required flags.
	if to == "" {
		a.printError(usageError("icloud: --to is required\n\nUsage:\n  imole icloud --to <dir> [--username email]"))
		return ExitUsage
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

	if recent > 0 {
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
	fmt.Fprintf(a.out, "  %s\n", a.dim("Run `imole report --manifest "+filepath.Join(destAbs, "manifest.json")+"` if you also ran imole backup here."))
	return ExitSuccess
}

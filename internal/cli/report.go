package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/backup"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/report"
)

func (a *App) runReport(_ context.Context, args []string) int {
	var manifestPath string
	var jsonMode bool
	fs := flagSet("report")
	fs.StringVar(&manifestPath, "manifest", backup.ManifestName, "manifest path")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}
	manifest, err := backup.ReadManifest(manifestPath)
	if err != nil {
		a.printError(runtimeError("manifest_read_failed", err.Error(), "Check that --manifest points to a valid manifest.json file", false))
		return ExitError
	}
	summary := report.FromManifest(manifest)
	if a.shouldJSON() || jsonMode {
		return a.writeJSON(summary)
	}
	fmt.Fprintln(a.out, "iMole Backup Report")
	fmt.Fprintf(a.out, "Manifest: %s\n", absPath(manifestPath))
	fmt.Fprintf(a.out, "Created:  %s\n\n", manifest.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(a.out, "Files:    %d\n", summary.Files)
	fmt.Fprintf(a.out, "Verified: %d · %s\n", summary.Verified, human.Bytes(summary.VerifiedSize))
	fmt.Fprintf(a.out, "Failed:   %d\n", summary.Failed)
	fmt.Fprintf(a.out, "Total:    %s\n", human.Bytes(summary.TotalSize))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Potential iPhone saving: %s\n", human.Bytes(summary.CleanableSize))
	fmt.Fprintf(a.out, "Safe to review/delete:   %d verified files\n", summary.Cleanable)
	return ExitSuccess
}
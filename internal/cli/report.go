package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/chenhg5/imole/internal/backup"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/report"
)

func (a *App) runReport(_ context.Context, args []string) int {
	// Handle --help
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			a.runReportHelp()
			return ExitSuccess
		}
	}

	var manifestPath, fields, htmlOut string
	var jsonMode, verify bool
	fs := flagSet("report")
	fs.StringVar(&manifestPath, "manifest", backup.ManifestName, "manifest path")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	fs.BoolVar(&verify, "verify", false, "re-check each backed-up file still exists on disk")
	fs.StringVar(&htmlOut, "html", "", "generate HTML report; value is output file (default: report.html)")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	manifest, err := backup.ReadManifest(manifestPath)
	if err != nil {
		a.printError(runtimeError("manifest_read_failed", err.Error(), "Check that --manifest points to a valid manifest.json file", false))
		return ExitError
	}

	// --html mode: generate report and open in browser
	if htmlOut != "" || flagPresent(args, "--html") {
		outFile := htmlOut
		if outFile == "" || outFile == "true" {
			outFile = "report.html"
		}
		return a.generateHTMLReport(manifest, manifestPath, outFile, verify)
	}

	summary := report.FromManifest(manifest)

	// --verify: re-check files on disk
	var vr *report.VerifyResult
	if verify {
		manifestDir := filepath.Dir(absPath(manifestPath))
		stopSpinner := a.startSpinner("Verifying files on disk…")
		result := report.VerifyManifest(manifest, manifestDir)
		vr = &result
		stopSpinner(fmt.Sprintf("Verify complete: %.1f%% healthy (%d/%d files on disk)", result.HealthPct, result.OnDisk, result.Total))
	}

	if a.shouldJSON() || jsonMode {
		type jsonOut struct {
			Summary report.Summary       `json:"summary"`
			Verify  *report.VerifyResult `json:"verify,omitempty"`
		}
		return a.outputJSON(jsonOut{Summary: summary, Verify: vr}, fields)
	}

	fmt.Fprintln(a.out, a.bold("iMole Backup Report"))
	fmt.Fprintf(a.out, "Manifest: %s\n", absPath(manifestPath))
	fmt.Fprintf(a.out, "Created:  %s\n\n", manifest.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(a.out, "Files:    %d\n", summary.Files)
	fmt.Fprintf(a.out, "Verified: %d · %s\n", summary.Verified, a.cyan(human.Bytes(summary.VerifiedSize)))
	if summary.Failed > 0 {
		fmt.Fprintf(a.out, "Failed:   %d\n", summary.Failed)
	}
	fmt.Fprintf(a.out, "Total:    %s\n", a.cyan(human.Bytes(summary.TotalSize)))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Potential iPhone saving: %s\n", a.green(human.Bytes(summary.CleanableSize)))
	fmt.Fprintf(a.out, "Safe to review/delete:   %d verified files\n", summary.Cleanable)

	// Verify section
	if vr != nil {
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, a.bold("Backup Health"))
		pctColor := a.green
		if vr.HealthPct < 95 {
			pctColor = a.yellow
		}
		if vr.HealthPct < 80 {
			pctColor = a.red
		}
		bar := a.progressBar(vr.HealthPct)
		fmt.Fprintf(a.out, "  %s  %s\n", bar, pctColor(fmt.Sprintf("%.1f%%", vr.HealthPct)))
		fmt.Fprintf(a.out, "  On disk:   %d / %d files\n", vr.OnDisk, vr.Total)
		if vr.Missing > 0 {
			fmt.Fprintf(a.out, "  %s Missing:  %d files\n", a.red("✗"), vr.Missing)
		}
		if vr.Corrupted > 0 {
			fmt.Fprintf(a.out, "  %s Corrupted: %d files\n", a.red("✗"), vr.Corrupted)
		}
		if len(vr.Issues) > 0 {
			fmt.Fprintln(a.out)
			fmt.Fprintln(a.out, a.dim("  Issues (first 20):"))
			for _, issue := range vr.Issues {
				fmt.Fprintf(a.out, "    %s %s\n", a.red("→"), a.dim(issue))
			}
		}
	}

	return ExitSuccess
}

func (a *App) generateHTMLReport(manifest backup.Manifest, manifestPath, outFile string, verify bool) int {
	var vr *report.VerifyResult
	if verify {
		manifestDir := filepath.Dir(absPath(manifestPath))
		stopSpinner := a.startSpinner("Verifying files on disk…")
		result := report.VerifyManifest(manifest, manifestDir)
		vr = &result
		stopSpinner(fmt.Sprintf("Verify complete: %.1f%% healthy", result.HealthPct))
	}

	data := report.BuildHTMLData(manifest, vr)
	html, err := report.GenerateHTML(data)
	if err != nil {
		a.printError(runtimeError("html_render_failed", err.Error(), "", false))
		return ExitError
	}

	if err := os.WriteFile(outFile, []byte(html), 0o644); err != nil {
		a.printError(runtimeError("html_write_failed", err.Error(), "", false))
		return ExitError
	}

	abs, _ := filepath.Abs(outFile)
	fmt.Fprintf(a.out, "%s HTML report generated: %s\n", a.green("✓"), a.cyan(abs))
	fmt.Fprintf(a.out, "  %d files · %s freed\n", data.TotalFiles, a.cyan(data.TotalFreed))

	// Try to open in browser
	if a.isTTY {
		openInBrowser(abs)
	}
	return ExitSuccess
}

func openInBrowser(path string) {
	url := "file://" + path
	var cmd string
	var cmdArgs []string
	switch runtime.GOOS {
	case "darwin":
		cmd, cmdArgs = "open", []string{url}
	case "linux":
		cmd, cmdArgs = "xdg-open", []string{url}
	case "windows":
		cmd, cmdArgs = "cmd", []string{"/c", "start", url}
	default:
		return
	}
	_ = exec.Command(cmd, cmdArgs...).Start()
}

// flagPresent checks if a flag name was passed (handles --flag and --flag=value).
func flagPresent(args []string, name string) bool {
	for _, arg := range args {
		if arg == name || len(arg) > len(name) && arg[:len(name)+1] == name+"=" {
			return true
		}
	}
	return false
}

func (a *App) runReportHelp() {
	header := func(s string) string { return a.bold(s) + "\n" }
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-38s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}
	fmt.Fprint(a.out,
		header("imole report — summarise a backup manifest")+
			"\n"+
			header("Flags")+
			flag("--manifest PATH", "path to manifest.json (default: ./manifest.json)")+
			flag("--verify", "re-check each backed-up file still exists on disk")+
			flag("--html [FILE]", "generate HTML report (default output: report.html)")+
			flag("--json", "output JSON")+
			flag("--fields a,b.c", "select specific JSON fields")+
			"\n"+
			a.dim("Examples:\n")+
			a.dim("  imole report --manifest ~/backup/manifest.json\n")+
			a.dim("  imole report --manifest ~/backup/manifest.json --verify\n")+
			a.dim("  imole report --manifest ~/backup/manifest.json --html\n")+
			a.dim("  imole report --manifest ~/backup/manifest.json --html my-report.html\n")+
			"\n",
	)
}

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	githubRepo       = "chenhg5/imole"
	installScriptURL = "https://raw.githubusercontent.com/" + githubRepo + "/main/install.sh"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

func (a *App) runUpdate(_ context.Context, args []string) int {
	var check, nightly bool
	fs := flagSet("update")
	fs.BoolVar(&check, "check", false, "only check for updates, do not install")
	fs.BoolVar(&nightly, "nightly", false, "install the latest unreleased build from main branch (requires go)")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	if nightly {
		return a.doNightly()
	}

	fmt.Fprintf(a.err, "Current version: %s\n", Version)
	fmt.Fprintf(a.err, "Checking latest release from github.com/%s...\n", githubRepo)

	release, err := fetchLatestRelease()
	if err != nil {
		a.printError(runtimeError("update_check_failed", err.Error(),
			"Check your internet connection or visit https://github.com/"+githubRepo+"/releases", false))
		return ExitError
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(Version, "v")

	if latest == current {
		fmt.Fprintf(a.out, "Already up to date (%s).\n", Version)
		return ExitSuccess
	}

	fmt.Fprintf(a.out, "New version available: %s → %s\n", Version, release.TagName)
	if release.HTMLURL != "" {
		fmt.Fprintf(a.out, "Release notes: %s\n", release.HTMLURL)
	}

	if check {
		fmt.Fprintln(a.out, "Run: imole update  to install the new version.")
		return ExitSuccess
	}

	fmt.Fprintln(a.err)
	return a.doUpdate(release.TagName)
}

// doNightly installs the latest unreleased build from the main branch via go install.
// This mirrors mole's `mo update --nightly` behavior.
// Note: nightly builds are not available as pre-built binaries; go is required.
func (a *App) doNightly() int {
	goPath, err := exec.LookPath("go")
	if err != nil {
		a.printError(runtimeError("nightly_requires_go",
			"--nightly requires the Go toolchain to build from source",
			"Install Go from https://go.dev/dl/ then run: imole update --nightly",
			false))
		return ExitError
	}

	pkg := fmt.Sprintf("github.com/%s/cmd/imole@main", githubRepo)
	fmt.Fprintf(a.err, "Installing latest unreleased build from main branch...\n")
	fmt.Fprintf(a.err, "  go install %s\n", pkg)
	fmt.Fprintln(a.err)

	cmd := exec.Command(goPath, "install", pkg)
	cmd.Stdout = a.out
	cmd.Stderr = a.err
	if err := cmd.Run(); err != nil {
		a.printError(runtimeError("nightly_install_failed", err.Error(),
			"Try running manually: go install "+pkg, false))
		return ExitError
	}

	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Nightly build installed from main branch.")
	fmt.Fprintln(a.out, "Note: this is an unreleased build. Run 'imole update' to switch back to stable.")
	return ExitSuccess
}

// doUpdate installs the latest version using the best available method.
func (a *App) doUpdate(tag string) int {
	// Method 1: re-run install.sh (preferred — same flow as initial install).
	if canRunScript() {
		fmt.Fprintf(a.err, "Downloading and running install script...\n")
		return a.updateViaScript()
	}

	// Method 2: go install (fallback — works when GOPATH is set up).
	if goPath, err := exec.LookPath("go"); err == nil {
		fmt.Fprintf(a.err, "Updating via go install...\n")
		cmd := exec.Command(goPath, "install",
			fmt.Sprintf("github.com/%s/cmd/imole@%s", githubRepo, tag))
		cmd.Stdout = a.err
		cmd.Stderr = a.err
		if err := cmd.Run(); err != nil {
			a.printError(runtimeError("update_go_install_failed", err.Error(),
				"Try: go install github.com/"+githubRepo+"/cmd/imole@latest", false))
			return ExitError
		}
		fmt.Fprintln(a.out)
		fmt.Fprintf(a.out, "Updated to %s via go install.\n", tag)
		return ExitSuccess
	}

	// No method available.
	a.printError(runtimeError("update_no_method",
		"cannot update automatically: curl and go are both unavailable",
		"Download the latest binary from https://github.com/"+githubRepo+"/releases",
		false))
	return ExitError
}

// updateViaScript re-runs install.sh with --update flag.
func (a *App) updateViaScript() int {
	// Download install.sh to a temp file and execute it.
	sh, err := exec.LookPath("bash")
	if err != nil {
		sh = "/bin/bash"
	}
	curl, _ := exec.LookPath("curl")

	// Pipe curl into bash directly — same as the initial install UX.
	script := fmt.Sprintf(`%s -fsSL --connect-timeout 10 --max-time 60 %q | %s -s -- --update`,
		curl, installScriptURL, sh)

	cmd := exec.Command(sh, "-c", script)
	cmd.Stdout = a.out
	cmd.Stderr = a.err
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		a.printError(runtimeError("update_script_failed", err.Error(),
			"Try running manually: curl -fsSL "+installScriptURL+" | bash -s -- --update",
			false))
		return ExitError
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, "Update complete. Restart your terminal or run: hash -r")
	return ExitSuccess
}

// canRunScript returns true when curl and bash are available (macOS / Linux).
func canRunScript() bool {
	if runtime.GOOS == "windows" {
		return false
	}
	_, curlErr := exec.LookPath("curl")
	_, bashErr := exec.LookPath("bash")
	return curlErr == nil && bashErr == nil
}

// fetchLatestRelease queries the GitHub releases API.
func fetchLatestRelease() (*githubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "imole/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found at github.com/%s — check back later", githubRepo)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("GitHub response did not include a tag_name")
	}
	return &release, nil
}

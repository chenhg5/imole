package apps

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/syscmd"
)

// protectedPrefixes lists bundle ID prefixes that must never be uninstalled.
// These are Apple system apps, daemons, and frameworks that are part of iOS.
var protectedPrefixes = []string{
	"com.apple.",
}

// UninstallResult is the outcome of an uninstall operation.
type UninstallResult struct {
	BundleID string
	Name     string
	Size     int64
	Err      error
}

// CheckProtected returns an error if the bundle ID is protected and must not
// be uninstalled. Only user-installed third-party apps are allowed.
func CheckProtected(bundleID string) error {
	if bundleID == "" {
		return fmt.Errorf("bundle ID must not be empty")
	}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(bundleID, prefix) {
			return fmt.Errorf(
				"bundle ID %q starts with %q — Apple system apps cannot be uninstalled via imole, only user-installed third-party apps are supported (use: imole scan apps --top 20)",
				bundleID, prefix,
			)
		}
	}
	return nil
}

// FindApp looks up a user app by bundle ID in the installed app list.
// Returns the matching App or an error if not found.
func FindApp(ctx context.Context, bundleID string) (App, error) {
	result, err := List(ctx, ScopeUser)
	if err != nil {
		return App{}, fmt.Errorf("could not list installed apps: %w", err)
	}
	for _, app := range result.Apps {
		if app.BundleID == bundleID {
			return app, nil
		}
	}
	return App{}, fmt.Errorf(
		"app %q not found in user-installed apps\n"+
			"  Run: imole scan apps --top 20   to list installed apps and their bundle IDs",
		bundleID,
	)
}

// Uninstall removes a user-installed app from the connected iPhone.
// It calls `ideviceinstaller uninstall <bundleID>`.
func Uninstall(ctx context.Context, bundleID string) error {
	installer, err := syscmd.LookPath("ideviceinstaller")
	if err != nil {
		return fmt.Errorf("ideviceinstaller not found; install with: brew install ideviceinstaller")
	}

	runCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	out, err := exec.CommandContext(runCtx, installer, "uninstall", bundleID).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("uninstall failed: %s", msg)
	}
	return nil
}

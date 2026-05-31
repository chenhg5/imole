package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/chenhg5/imole/internal/apps"
	"github.com/chenhg5/imole/internal/history"
	"github.com/chenhg5/imole/internal/human"
)

func (a *App) runUninstall(ctx context.Context, args []string) int {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			a.runUninstallHelp()
			return ExitSuccess
		}
	}

	// IMOLE_NO_DELETE guard — same env var as `clean`.
	// Set IMOLE_NO_DELETE=1 to lock out all destructive operations.
	if v := os.Getenv(noDeleteEnv); v != "" {
		a.printError(&Error{
			Code:       "delete_disabled",
			Message:    noDeleteEnv + " is set — destructive operations are disabled in this environment",
			Suggestion: "Unset " + noDeleteEnv + " if you want to allow uninstall: unset " + noDeleteEnv,
			Retryable:  false,
		})
		return ExitError
	}

	var bundleID string
	var dryRun, yes bool
	fs := flagSet("uninstall")
	fs.StringVar(&bundleID, "bundle-id", "", "bundle ID of the app to uninstall (e.g. com.spotify.client)")
	fs.BoolVar(&dryRun, "dry-run", false, "show what would be uninstalled without removing anything")
	fs.BoolVar(&yes, "yes", false, "skip confirmation prompt")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	if bundleID == "" {
		a.printError(usageError("--bundle-id is required\n  Run: imole scan apps --top 20   to find bundle IDs"))
		return ExitUsage
	}

	// Block protected (system) apps before doing anything else.
	if err := apps.CheckProtected(bundleID); err != nil {
		a.printError(runtimeError("protected_app", err.Error(), "", false))
		return ExitError
	}

	// Look up the app to get its display name and size.
	stopSpinner := a.startSpinner("Looking up app…")
	app, err := apps.FindApp(ctx, bundleID)
	stopSpinner("")
	if err != nil {
		a.printError(runtimeError("app_not_found", err.Error(),
			"Make sure the iPhone is connected and trusted", false))
		return ExitError
	}

	// Show what will be uninstalled.
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("App to uninstall"))
	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "  Name:      %s\n", a.cyan(app.Name))
	fmt.Fprintf(a.out, "  Bundle ID: %s\n", a.dim(app.BundleID))
	fmt.Fprintf(a.out, "  Size:      %s  (app %s · data %s)\n",
		a.cyan(human.Bytes(app.TotalSize)),
		human.Bytes(app.StaticSize),
		human.Bytes(app.DynamicSize),
	)
	fmt.Fprintln(a.out)

	if dryRun {
		fmt.Fprintf(a.err, "Dry-run: %s (%s) would be uninstalled from iPhone (exit 10 = safe).\n",
			app.Name, app.BundleID)
		return ExitDryRun
	}

	// Warn clearly: this is irreversible.
	fmt.Fprintln(a.out, a.yellow("Warning: Uninstalling an app removes it and all its data from your iPhone."))
	fmt.Fprintln(a.out, a.yellow("         This cannot be undone. The app can be reinstalled from the App Store,"))
	fmt.Fprintln(a.out, a.yellow("         but its private data (documents, settings, cache) will be gone."))
	fmt.Fprintln(a.out)

	if !yes && !a.confirm(fmt.Sprintf("Uninstall %s from iPhone?", app.Name)) {
		fmt.Fprintln(a.out, "Aborted.")
		return ExitSuccess
	}

	stopSpinner2 := a.startSpinner(fmt.Sprintf("Uninstalling %s…", app.Name))
	uninstallErr := apps.Uninstall(ctx, bundleID)
	stopSpinner2("")

	if uninstallErr != nil {
		a.printError(runtimeError("uninstall_failed", uninstallErr.Error(),
			"Make sure the iPhone is connected, unlocked, and trusted", false))
		history.Append(history.Entry{
			Kind:     history.KindUninstall,
			BundleID: bundleID,
			AppName:  app.Name,
			Size:     app.TotalSize,
			Failed:   1,
		})
		return ExitError
	}

	history.Append(history.Entry{
		Kind:     history.KindUninstall,
		BundleID: bundleID,
		AppName:  app.Name,
		Size:     app.TotalSize,
	})

	fmt.Fprintln(a.out, a.bold("Uninstall complete"))
	fmt.Fprintf(a.out, "  %s %s  (%s freed)\n",
		a.green("Removed:"), a.cyan(app.Name), human.Bytes(app.TotalSize))
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.dim("  Note: app data removal may take a moment to reflect in iPhone storage."))

	return ExitSuccess
}

func (a *App) runUninstallHelp() {
	fmt.Fprint(a.out, a.renderUninstallHelp())
}

func (a *App) renderUninstallHelp() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-40s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}

	return header("imole uninstall — remove a user-installed app from iPhone") +
		"\n" +
		a.yellow("  Safety: blocked by IMOLE_NO_DELETE. Only user apps (non-Apple) can be removed.\n") +
		"\n" +
		header("Flags") +
		flag("--bundle-id ID", "Required. Bundle ID of the app (e.g. com.spotify.client)") +
		flag("--dry-run", "Show what would be uninstalled without removing (exit 10 = safe)") +
		flag("--yes", "Skip confirmation prompt") +
		"\n" +
		a.dim("  Run: imole scan apps --top 20   to find app names and bundle IDs.\n") +
		a.dim("  com.apple.* apps are blocked — Apple system apps cannot be removed.\n") +
		"\n" +
		a.dim("Examples:\n") +
		a.dim("  imole scan apps --top 20\n") +
		a.dim("  imole uninstall --bundle-id com.spotify.client --dry-run\n") +
		a.dim("  imole uninstall --bundle-id com.spotify.client\n") +
		"\n"
}

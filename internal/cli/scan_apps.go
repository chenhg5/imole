package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/apps"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/textutil"
)

func (a *App) runScanApps(ctx context.Context, args []string) int {
	var scope, fields string
	var top int
	var jsonMode bool
	fs := flagSet("apps")
	fs.StringVar(&scope, "scope", "user", "app scope: user, system, all")
	fs.IntVar(&top, "top", 30, "number of apps to show")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	a.status("Querying app storage…")
	result, err := apps.List(ctx, apps.Scope(scope))
	if err != nil {
		a.printError(runtimeError("apps_failed", err.Error(), "Install with: brew install ideviceinstaller", true))
		return ExitError
	}
	if top > 0 && top < len(result.Apps) {
		result.Apps = result.Apps[:top]
	}
	if a.shouldJSON() || jsonMode {
		return a.outputJSON(result, fields)
	}

	fmt.Fprintln(a.out, a.bold("iMole App Storage"))
	fmt.Fprintf(a.out, "Scope: %s · Apps: %d\n\n", a.cyan(result.Scope), len(result.Apps))
	fmt.Fprintf(a.out, "%-3s %s %10s %10s %10s  %s\n", "#", textutil.PadRight("App", 24), "Total", "Data", "App", "Bundle")
	for i := range result.Apps {
		app := result.Apps[i]
		name := textutil.PadRight(textutil.Truncate(app.Name, 24), 24)
		fmt.Fprintf(a.out, "%-3d %s %10s %10s %10s  %s\n",
			i+1,
			name,
			a.cyan(fmt.Sprintf("%10s", human.Bytes(app.TotalSize))),
			human.Bytes(app.DynamicSize),
			human.Bytes(app.StaticSize),
			app.BundleID,
		)
	}
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.dim("Source: iOS installation_proxy StaticDiskUsage/DynamicDiskUsage."))
	fmt.Fprintln(a.out, a.dim("Some apps, especially chat apps using shared App Group containers, may be underreported."))
	fmt.Fprintln(a.out, a.dim("iMole can rank apps, but it cannot safely clear private app caches directly."))
	return ExitSuccess
}

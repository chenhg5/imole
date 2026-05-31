package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/device"
)

func (a *App) runDoctor(ctx context.Context, args []string) int {
	var jsonMode bool
	var fields string
	fs := flagSet("doctor")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	fs.StringVar(&fields, "fields", "", "comma-separated dot-paths to include in JSON output")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	a.status("Checking device and dependencies…")
	report := device.Check(ctx)
	if a.shouldJSON() || jsonMode {
		return a.outputJSON(report, fields)
	}

	fmt.Fprintln(a.out, a.bold("iMole Doctor"))
	fmt.Fprintln(a.out)
	for _, dep := range report.Dependencies {
		var status string
		if dep.Found {
			status = a.green("found")
		} else {
			status = a.red("missing")
		}
		fmt.Fprintf(a.out, "  %-14s %s", dep.Name, status)
		if dep.Path != "" {
			fmt.Fprintf(a.out, " · %s", a.dim(dep.Path))
		}
		if !dep.Found {
			fmt.Fprintf(a.out, " · install: %s", a.dim(dep.Install))
		}
		fmt.Fprintln(a.out)
	}
	fmt.Fprintln(a.out)
	if report.Device.UDID == "" {
		fmt.Fprintln(a.out, a.yellow("Device: not detected"))
		fmt.Fprintln(a.out, a.dim("Tip: connect iPhone by USB, unlock it, and tap Trust This Computer."))
		return ExitSuccess
	}
	fmt.Fprintf(a.out, "%s %s\n", a.bold("Device:"), a.green(firstNonEmpty(report.Device.Name, "iPhone")))
	fmt.Fprintf(a.out, "  UDID: %s\n", a.dim(report.Device.UDID))
	if report.Device.ProductType != "" {
		fmt.Fprintf(a.out, "  Model: %s\n", report.Device.ProductType)
	}
	if report.Device.IOSVersion != "" {
		fmt.Fprintf(a.out, "  iOS: %s\n", report.Device.IOSVersion)
	}
	return ExitSuccess
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
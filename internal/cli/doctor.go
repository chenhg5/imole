package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/device"
)

func (a *App) runDoctor(ctx context.Context, args []string) int {
	var jsonMode bool
	fs := flagSet("doctor")
	fs.BoolVar(&jsonMode, "json", false, "output JSON")
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	report := device.Check(ctx)
	if a.shouldJSON() || jsonMode {
		return a.writeJSON(report)
	}

	fmt.Fprintln(a.out, "iMole Doctor")
	fmt.Fprintln(a.out)
	for _, dep := range report.Dependencies {
		status := "missing"
		if dep.Found {
			status = "found"
		}
		fmt.Fprintf(a.out, "  %-14s %s", dep.Name, status)
		if dep.Path != "" {
			fmt.Fprintf(a.out, " · %s", dep.Path)
		}
		if !dep.Found {
			fmt.Fprintf(a.out, " · install: %s", dep.Install)
		}
		fmt.Fprintln(a.out)
	}
	fmt.Fprintln(a.out)
	if report.Device.UDID == "" {
		fmt.Fprintln(a.out, "Device: not detected")
		fmt.Fprintln(a.out, "Tip: connect iPhone by USB, unlock it, and tap Trust This Computer.")
		return ExitSuccess
	}
	fmt.Fprintf(a.out, "Device: %s\n", firstNonEmpty(report.Device.Name, "iPhone"))
	fmt.Fprintf(a.out, "  UDID: %s\n", report.Device.UDID)
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
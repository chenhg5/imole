package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/imole/internal/device"
	"github.com/chenhg5/imole/internal/human"
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

	stopSpinner := a.startSpinner("Checking device and dependencies…")
	report := device.Check(ctx)
	stopSpinner("")
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
		fmt.Fprintln(a.out, a.dim("Tip: connect iPhone by USB, unlock it, and run: idevicepair pair"))
		return ExitSuccess
	}
	fmt.Fprintf(a.out, "%s %s\n", a.bold("Device:"), a.green(firstNonEmpty(report.Device.Name, "iPhone")))
	fmt.Fprintf(a.out, "  UDID: %s\n", a.dim(report.Device.UDID))
	if !report.Device.Trusted {
		fmt.Fprintf(a.out, "  Trust: %s\n", a.yellow("not paired for libimobiledevice"))
		fmt.Fprintln(a.out, a.dim("  Run: idevicepair pair"))
		fmt.Fprintln(a.out, a.dim("  Keep the iPhone unlocked and tap Trust on the device when prompted."))
	}
	if report.Device.ProductType != "" {
		fmt.Fprintf(a.out, "  Model: %s\n", report.Device.ProductType)
	}
	if report.Device.IOSVersion != "" {
		fmt.Fprintf(a.out, "  iOS:   %s\n", report.Device.IOSVersion)
	}
	if report.Device.Storage != nil {
		s := report.Device.Storage
		bar := a.progressBar(s.UsedPercent)
		fmt.Fprintf(a.out, "  Storage: %s  %s / %s free\n",
			bar,
			a.cyan(human.Bytes(s.UsedData)),
			human.Bytes(s.AmountDataAvailable),
		)
		a.debug("storage: used=%d total=%d free=%.1f%%", s.UsedData, s.TotalDataCapacity, s.FreePercent)
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

package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chenhg5/imole/internal/filter"
	"github.com/chenhg5/imole/internal/human"
	"github.com/chenhg5/imole/internal/provider"
)

type cleanPlan struct {
	label       string
	description string
	emoji       string
	filter      filter.Filter
	files       int
	size        int64
}

func (a *App) runPlan(ctx context.Context, args []string) int {
	var providerName, source string
	fs := flagSet("plan")
	addProviderFlags(fs, &providerName, &source)
	if err := parseFlags(fs, args); err != nil {
		a.printError(usageError(err.Error()))
		return ExitUsage
	}

	// Scan device
	stopSpinner := a.startSpinner("Scanning device to build cleanup plans…")
	result, err := scanFromFlags(ctx, providerName, source, 0, 0)
	if err != nil {
		stopSpinner("")
		a.printError(runtimeError("scan_failed", err.Error(), scanHint(providerName, source), true))
		return ExitError
	}
	stopSpinner(fmt.Sprintf("Scan complete: %d files · %s", result.Summary.TotalFiles, human.Bytes(result.Summary.TotalSize)))

	now := time.Now()

	plans := []cleanPlan{
		{
			label:       "Safe",
			emoji:       "🟢",
			description: "Back up videos older than 180 days · keep all photos",
			filter:      filter.Filter{Only: filter.KindVideos, OlderThan: 180 * 24 * time.Hour, Now: now},
		},
		{
			label:       "Balanced",
			emoji:       "🟡",
			description: "Back up all media older than 90 days",
			filter:      filter.Filter{Only: filter.KindAll, OlderThan: 90 * 24 * time.Hour, Now: now},
		},
		{
			label:       "Aggressive",
			emoji:       "🔴",
			description: "Back up all media older than 30 days + files larger than 100 MB",
			filter:      filter.Filter{Only: filter.KindAll, OlderThan: 30 * 24 * time.Hour, LargeThan: 100 * 1024 * 1024, Now: now},
		},
	}

	// Estimate savings for each plan
	for i := range plans {
		matched := provider.FilteredItems(result, plans[i].filter)
		plans[i].files = len(matched)
		for _, item := range matched {
			plans[i].size += item.Size
		}
	}

	// Print plans
	fmt.Fprintln(a.out)
	fmt.Fprintln(a.out, a.bold("iMole Cleanup Plans"))
	fmt.Fprintf(a.out, "Device: %d files · %s total\n\n", result.Summary.TotalFiles, a.cyan(human.Bytes(result.Summary.TotalSize)))

	for i, p := range plans {
		pct := 0.0
		if result.Summary.TotalSize > 0 {
			pct = float64(p.size) * 100 / float64(result.Summary.TotalSize)
		}
		bar := a.progressBar(pct)
		fmt.Fprintf(a.out, "  %s  Plan %s%d — %s%s\n", p.emoji, a.bold(""), i+1, a.bold(p.label), "")
		fmt.Fprintf(a.out, "     %s\n", a.dim(p.description))
		fmt.Fprintf(a.out, "     %s  %s  (%d files)\n\n",
			bar,
			a.cyan(human.Bytes(p.size)),
			p.files,
		)
	}

	if !a.isTTY {
		return ExitSuccess
	}

	// Interactive selection
	fmt.Fprint(a.out, "Select a plan [1/2/3] or q to quit: ")
	reader := bufio.NewReader(a.in)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	var chosen *cleanPlan
	switch answer {
	case "1":
		chosen = &plans[0]
	case "2":
		chosen = &plans[1]
	case "3":
		chosen = &plans[2]
	case "q", "Q", "":
		fmt.Fprintln(a.out, a.dim("No plan selected. Run: imole backup --to PATH [--only videos] [--older-than 90d]"))
		return ExitSuccess
	default:
		a.printError(usageError(fmt.Sprintf("invalid choice %q — enter 1, 2, 3 or q", answer)))
		return ExitUsage
	}

	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Plan %s selected: %d files · %s\n",
		a.bold(chosen.label), chosen.files, a.cyan(human.Bytes(chosen.size)))

	// Ask for backup destination
	fmt.Fprint(a.out, "Backup destination [default: ~/iphone-backup]: ")
	destLine, _ := reader.ReadString('\n')
	dest := strings.TrimSpace(destLine)
	if dest == "" {
		home, _ := homeDir()
		dest = home + "/iphone-backup"
	}

	fmt.Fprintln(a.out)
	fmt.Fprintf(a.out, "Ready to back up %s to %s\n",
		a.cyan(fmt.Sprintf("%d files · %s", chosen.files, human.Bytes(chosen.size))),
		a.cyan(dest))
	fmt.Fprint(a.out, "Proceed? [y/N] ")
	confirmLine, _ := reader.ReadString('\n')
	if confirm := strings.TrimSpace(confirmLine); confirm != "y" && confirm != "Y" {
		fmt.Fprintln(a.out, "Cancelled. Run when ready:")
		a.printPlanCommand(chosen, dest, source, providerName)
		return ExitSuccess
	}

	// Execute backup
	fmt.Fprintln(a.out)
	backupArgs := a.planToBackupArgs(chosen, dest, source, providerName)
	return a.runBackup(ctx, backupArgs)
}

func (a *App) planToBackupArgs(p *cleanPlan, dest, source, providerName string) []string {
	args := []string{"--to", dest}
	if p.filter.OlderThan > 0 {
		args = append(args, "--older-than", formatAge(p.filter.OlderThan))
	}
	if p.filter.Only != filter.KindAll {
		args = append(args, "--only", string(p.filter.Only))
	}
	if p.filter.LargeThan > 0 {
		args = append(args, "--large-than", human.Bytes(p.filter.LargeThan))
	}
	if source != "" {
		args = append(args, "--source", source)
	}
	if providerName != "auto" {
		args = append(args, "--provider", providerName)
	}
	args = append(args, "--yes") // already confirmed above
	return args
}

func (a *App) printPlanCommand(p *cleanPlan, dest, source, providerName string) {
	args := a.planToBackupArgs(p, dest, source, providerName)
	fmt.Fprintf(a.out, a.dim("  imole backup %s\n"), strings.Join(args, " "))
}

func formatAge(d time.Duration) string {
	days := int(d.Hours() / 24)
	switch {
	case days%365 == 0:
		return fmt.Sprintf("%dy", days/365)
	case days%30 == 0:
		return fmt.Sprintf("%dm", days/30)
	default:
		return fmt.Sprintf("%dd", days)
	}
}

func homeDir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "~", err
	}
	return h, nil
}

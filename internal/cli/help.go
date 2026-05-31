package cli

import "fmt"

func (a *App) runScanHelp() {
	fmt.Fprint(a.out, a.renderScanHelp())
}

func (a *App) renderScanHelp() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-40s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}
	cmd := func(name, desc string) string {
		padded := fmt.Sprintf("%-40s", name)
		return "  " + a.cyan(padded) + a.dim(desc) + "\n"
	}

	return header("imole scan — scan iPhone storage") +
		"\n" +
		header("Subcommands") +
		cmd("scan [--flags]", "Scan media (photos + videos) — default") +
		cmd("scan media [--flags]", "Media-only scan") +
		cmd("scan apps [--top N]", "Rank apps by storage usage") +
		"\n" +
		header("Media scan flags") +
		flag("--summary", "Compact stats table: media + app storage") +
		flag("--top N [--only videos|photos]", "Largest N files sorted by size") +
		flag("--only all|photos|videos", "Filter by media type") +
		flag("--older-than 90d|6m|1y", "Filter files older than age") +
		flag("--large-than 500MB|1GB", "Filter files larger than size") +
		flag("--source PATH", "Scan local mount instead of USB device") +
		flag("--cache", "Use cached scan result (< 1 h old)") +
		"\n" +
		header("Output flags") +
		flag("--json", "Force JSON output") +
		flag("--fields a,b.c", "Select specific JSON fields (dot-path)") +
		"\n" +
		a.dim("  scan is read-only — --dry-run is not accepted.\n") +
		"\n" +
		a.dim("Examples:\n") +
		a.dim("  imole scan --summary\n") +
		a.dim("  imole scan --top 20 --only videos\n") +
		a.dim("  imole scan apps --top 20\n") +
		a.dim("  imole scan --cache --summary --json\n") +
		"\n"
}

func (a *App) runBackupHelp() {
	fmt.Fprint(a.out, a.renderBackupHelp())
}

func (a *App) renderBackupHelp() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-40s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}

	return header("imole backup — back up media to local disk") +
		"\n" +
		header("Flags") +
		flag("--to PATH", "Required. Destination directory for backup") +
		flag("--only all|photos|videos", "Filter by media type") +
		flag("--older-than 90d|6m|1y", "Filter files older than age") +
		flag("--large-than 500MB|1GB", "Filter files larger than size") +
		flag("--file REL_PATH", "Back up a specific file; repeatable") +
		flag("--source PATH", "Scan local mount instead of USB device") +
		flag("--dry-run", "Preview without copying (exit 10 = safe)") +
		flag("--json", "Force JSON output") +
		flag("--fields a,b.c", "Select specific JSON fields (dot-path)") +
		"\n" +
		a.dim("Examples:\n") +
		a.dim("  imole backup --to ~/iphone-backup --only videos --older-than 90d\n") +
		a.dim("  imole backup --to ~/backup --only videos --dry-run\n") +
		a.dim("  imole backup --to ~/backup --file DCIM/202507__/IMG_7523.MOV\n") +
		"\n"
}

func (a *App) runCleanHelp() {
	fmt.Fprint(a.out, a.renderCleanHelp())
}

func (a *App) renderCleanHelp() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-40s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}

	return header("imole clean — delete verified files from iPhone") +
		"\n" +
		header("Flags") +
		flag("--manifest PATH", "Required. Path to manifest.json from backup") +
		flag("--source PATH", "Delete from local mount instead of USB device") +
		flag("--file REL_PATH", "Delete one verified file; repeatable") +
		flag("--dry-run", "Preview without deleting (exit 10 = safe)") +
		flag("--yes", "Skip confirmation prompt") +
		"\n" +
		a.dim("  Only files marked verified:true in manifest are deleted.\n") +
		"\n" +
		a.dim("Examples:\n") +
		a.dim("  imole clean --manifest ~/iphone-backup/manifest.json --dry-run\n") +
		a.dim("  imole clean --manifest ~/backup/manifest.json --yes\n") +
		a.dim("  imole clean --manifest ~/backup/manifest.json --file DCIM/202507__/IMG_7523.MOV\n") +
		"\n"
}

func (a *App) runHelp() {
	// Banner: ASCII mole + phone art + project info
	fmt.Fprint(a.out, a.renderBanner())
	fmt.Fprint(a.out, a.renderCommands())
	fmt.Fprint(a.out, a.renderExamples())
}

func (a *App) renderBanner() string {
	mole := []string{
		`   /\_/\   `,
		`  / o o \  `,
		` |  =-=  | `,
		`  \_[I]_/  `,
		`  / | | \  `,
	}
	info := []string{
		a.boldCyan("iMole") + " — iPhone Storage Cleaner",
		a.dim("Back up, clean, and slim down your iPhone from the terminal."),
		a.dim("Inspired by Mole · ") + a.cyan("https://github.com/chenhg5/imole"),
		"",
		a.dim("usage: ") + a.bold("imole") + a.dim(" <command> [flags]"),
	}

	out := "\n"
	for i := 0; i < len(mole) || i < len(info); i++ {
		left := ""
		if i < len(mole) {
			left = a.yellow(mole[i])
		} else {
			left = "           "
		}
		right := ""
		if i < len(info) {
			right = info[i]
		}
		out += left + "  " + right + "\n"
	}
	return out + "\n"
}

func (a *App) renderCommands() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	cmd := func(name, desc string) string {
		padded := fmt.Sprintf("%-36s", name)
		return "  " + a.cyan(padded) + a.dim(desc) + "\n"
	}
	flag := func(name, desc string) string {
		padded := fmt.Sprintf("%-36s", name)
		return "  " + a.green(padded) + a.dim(desc) + "\n"
	}

	return header("Commands") +
		cmd("doctor", "Check device connection and dependencies") +
		cmd("scan", "Scan iPhone storage (media by default)") +
		cmd("scan apps [--top N]", "Rank apps by iPhone storage usage") +
		cmd("backup  --to PATH [filters]", "Back up media, write manifest.json") +
		cmd("report  --manifest PATH", "Summarize a backup manifest") +
		cmd("clean   --manifest PATH", "Delete verified files from iPhone") +
		cmd("guide   [topic]", "Cleanup guide; use guide analysis for agent playbook") +
		cmd("history [--limit N]", "Show recent backup and delete operations") +
		cmd("update  [--check] [--nightly]", "Update imole to the latest release") +
		cmd("completion [zsh|bash|fish]", "Generate shell tab-completion script") +
		cmd("schema  [command]", "Machine-readable command schema (agent use)") +
		"\n" +
		header("scan flags") +
		flag("(no flags)", "Full media scan report with next-step hints") +
		flag("--summary", "Combined summary: media + app storage") +
		flag("media --summary", "Media-only compact stats table") +
		flag("apps --top N", "App storage ranking") +
		flag("--top N [--only videos|photos]", "Largest N files sorted by size") +
		flag("--cache", "Use cached scan (< 1 h old), skip USB wait") +
		flag("--older-than 90d|6m|1y", "Filter: files older than age") +
		flag("--large-than 500MB|1GB", "Filter: files larger than size") +
		flag("--source PATH", "Scan local mount instead of USB device") +
		flag("backup --file REL_PATH", "Back up one file from scan output; repeatable") +
		flag("clean --file REL_PATH", "Delete one verified file from manifest; repeatable") +
		"\n" +
		header("Output flags") +
		flag("--json", "Force JSON output") +
		flag("--fields a,b.c", "Select specific JSON fields (dot-path)") +
		"\n" +
		header("Preview flags  (backup and clean only)") +
		flag("--dry-run", "Preview without side effects (exit 10 = safe)") +
		flag("--yes", "Skip interactive confirmation prompt (backup, clean)") +
		"\n" +
		header("Global flags") +
		flag("--debug", "Verbose output: provider selection, scan details, storage info") +
		a.dim("  scan, scan apps, doctor, report, history, schema, and guide are read-only and do not accept --dry-run.\n") +
		"\n"
}

func (a *App) renderExamples() string {
	header := func(s string) string { return a.bold(s) + "\n" }
	ex := func(comment, code string) string {
		return a.dim("  # "+comment) + "\n" +
			"  " + a.cyan("imole") + " " + code + "\n"
	}

	return header("Examples") +
		ex("Check device and see what's eating space",
			"doctor && scan --summary") +
		ex("Find the biggest video culprits",
			"scan --top 20 --only videos") +
		ex("Find apps with the largest private data",
			"scan apps --top 20") +
		ex("Show the storage analysis playbook for agents",
			"guide analysis") +
		ex("Preview then back up old videos",
			"backup --to ~/iphone-backup --only videos --older-than 90d --dry-run") +
		ex("Back up one file from scan output",
			"backup --to ~/iphone-backup --file DCIM/202507__/IMG_7523.MOV --dry-run") +
		ex("Delete one verified file from a manifest",
			"clean --manifest ~/iphone-backup/manifest.json --file DCIM/202507__/IMG_7523.MOV --dry-run") +
		ex("Execute backup and delete verified files",
			"backup --to ~/iphone-backup --only videos --older-than 90d\n  "+
				a.cyan("imole")+" clean  --manifest ~/iphone-backup/manifest.json") +
		ex("Agent-friendly JSON (auto when piped)",
			"scan --summary --json --fields device.storage.free_percent,media.video_size,apps.total_size") +
		ex("Use cached scan to skip 15 s USB wait",
			"scan --cache --top 30 --only videos") +
		"\n" +
		a.dim("  Exit codes: 0 success · 1 error · 2 bad args · 10 dry-run OK\n") +
		"\n"
}

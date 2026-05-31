package cli

import "fmt"

func (a *App) runHelp() {
	fmt.Fprint(a.out, helpText)
}

const helpText = `imole - open-source iPhone slimming toolkit

Usage:
  imole schema [command]                Show command structure and parameters (agent-friendly)
  imole doctor                          Check device and local dependencies
  imole scan [flags]                    Scan media — summary, top N, or compact stats
  imole backup --to PATH [filters]      Back up media and write manifest
  imole report --manifest PATH          Summarize a backup manifest
  imole guide [topic]                   Show cleanup guidance
  imole clean --manifest PATH           Delete verified files from iPhone
  imole history [--limit N]             Show recent backup and delete operations
  imole update [--check] [--nightly]    Update imole to the latest release

scan flags:
  (no flags)          Scan report with summary and next-step hints
  --summary           Compact stats table
  --top N             Top N largest files sorted by size
  --only videos|photos|all   Filter by media type
  --older-than 90d|6m|1y     Filter by age
  --large-than 500MB|1GB     Filter by size
  --cache             Use cached result (< 1h old); skips slow USB enumeration
  --provider auto|imagecapture|filesystem|gphoto
  --source PATH       Scan a local mounted path instead of USB

Other common flags:
  --json              Force JSON output
  --fields a,b.c      Select specific JSON fields (dot-path)

Exit codes:
  0   Success
  1   General error
  2   Invalid arguments / usage error
  10  Dry-run passed (safe to execute)

Examples:
  # Full scan report
  imole scan

  # Compact stats (what stats used to do)
  imole scan --summary

  # Top 30 largest videos  (what videos used to do)
  imole scan --top 30 --only videos

  # Top 20 largest photos
  imole scan --top 20 --only photos

  # Top 30 files older than 6 months
  imole scan --top 30 --older-than 6m

  # Agent-friendly JSON
  imole scan --json | jq '.summary'
  imole scan --summary --json --fields total_size_human,video_files

  # Preview backup (dry-run)
  imole backup --to /tmp/backup --only videos --older-than 90d --dry-run

  # Execute backup then delete
  imole backup --to ~/iphone-backup --only videos --older-than 90d
  imole clean  --manifest ~/iphone-backup/manifest.json
`

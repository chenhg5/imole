package cli

import "fmt"

func (a *App) runHelp() {
	fmt.Fprint(a.out, helpText)
}

const helpText = `imole - open-source iPhone slimming toolkit

Usage:
  imole schema [command]                Show command structure and parameters (agent-friendly)
  imole doctor                          Check device and local dependencies
  imole scan [flags]                    Scan visible iPhone media
  imole videos [--top N]               Show largest videos
  imole backup --to PATH [filters]      Back up media and write manifest
  imole report --manifest PATH          Summarize a backup manifest
  imole guide [topic]                   Show cleanup guidance
  imole clean                           Explain safe cleanup boundaries

Commands:
  doctor   Check system dependencies and device connectivity
  scan     Enumerate media files on connected iPhone or local path
  videos   List largest video files
  backup   Copy selected media to local destination with verification
  report   Summarize a backup manifest file
  guide    Show cleanup guidance for specific topics
  clean    Explain safe cleanup boundaries

Common flags:
  --provider auto|filesystem|imagecapture|gphoto
  --source PATH
  --only all|photos|videos
  --older-than 90d|6m|1y
  --large-than 500MB|1GB
  --fields path.to.field,path.to.field2  (JSON field filtering)

Output:
  JSON is output automatically when stdout is not a terminal.
  Use --json to force JSON output in terminal mode.
  Use --fields to select specific JSON fields (dot-path notation).

Exit codes:
  0   Success
  1   General error
  2   Invalid arguments / usage error
  3   Resource not found
  5   Conflict / already exists
  10  Dry-run passed (preview successful, safe to execute)

Examples:
  # Scan iPhone media
  imole scan --json | jq '.summary'

  # Scan with field filtering
  imole scan --json --fields summary.total_files,summary.photo_files

  # Find largest videos
  imole videos --top 30 --json

  # Preview backup (dry-run)
  imole backup --to /tmp/backup --only videos --older-than 90d --dry-run

  # Execute backup
  imole backup --to /path/to/backup --only videos --older-than 90d

  # Check manifest with field filtering
  imole report --manifest /path/to/backup/manifest.json --json --fields files_count,verified_size

Notes:
  iMole v0.1 focuses on diagnosis, backup, verification, and guidance.
  It does not automatically delete iPhone Photos library content by default.
  Double-dash (--flag) and single-dash (-flag) are both supported for all flags.
`

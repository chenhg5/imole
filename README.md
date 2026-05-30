<div align="center">
  <h1>iMole</h1>
  <p><em>🐹 Back up, clean, and slim down your iPhone from the terminal.</em></p>
</div>

<p align="center">
  <a href="https://github.com/chenhg5/imole/stargazers"><img src="https://img.shields.io/github/stars/chenhg5/imole?style=flat-square" alt="Stars"></a>
  <a href="https://github.com/chenhg5/imole/releases"><img src="https://img.shields.io/github/v/tag/chenhg5/imole?label=version&style=flat-square" alt="Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square" alt="License"></a>
  <a href="https://github.com/chenhg5/imole/commits"><img src="https://img.shields.io/github/commit-activity/m/chenhg5/imole?style=flat-square" alt="Commits"></a>
</p>

> **Free up your iPhone without buying more iCloud.** iMole scans what's eating your iPhone storage, backs up photos and videos to your computer, verifies each file, and then safely deletes the originals from the device — all from a single command.

## Quick Start

**Install**

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

**Run the full cleanup flow**

```bash
imole doctor                                           # check device is connected

imole stats                                            # see what's eating space
# Total:   38,421 files · 286.4 GB
# Videos:   1,204 files · 172.8 GB
# Photos:  37,217 files · 113.6 GB

imole videos --top 10                                  # find the biggest culprits

imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run   # preview
imole backup --to ~/iphone-backup --only videos --older-than 90d              # back up

imole report --manifest ~/iphone-backup/manifest.json  # confirm all verified

imole clean  --manifest ~/iphone-backup/manifest.json  # delete from iPhone
# → on iPhone: Photos → Recently Deleted → Delete All  → space freed 🎉
```

## Features

- **Space diagnosis** — scan DCIM over USB, rank by size, filter by age or kind
- **Smart backup** — copy to any local path, organized by year/month, verify by size
- **Manifest** — every backup writes a `manifest.json` with source path, size, and verification status
- **Safe deletion** — `imole clean` only deletes files that are `verified: true` in the manifest
- **Cross-platform** — macOS (ImageCaptureCore, native USB), Linux (gphoto2 / ifuse), Windows (`--source PATH`)
- **Agent-friendly** — `--json` output, `--fields` selection, `imole schema` for machine-readable API surface
- **Operation log** — `imole history` shows what was backed up and deleted

## Platform Support

| Feature | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| USB auto-scan | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ |
| Scan via `--source PATH` | ✅ | ✅ | ✅ |
| Backup (copy + verify) | ✅ | ✅ | ✅ |
| Delete from device (USB) | ✅ | ❌ | ❌ |
| Device detection | ✅ | ✅ | ✅ |

## Install

### Script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

### Homebrew (coming soon)

```bash
brew install imole
```

### From source

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

## Dependencies

**macOS** — no extra installs needed. ImageCaptureCore is built in. For device info:

```bash
brew install libimobiledevice   # optional, for imole doctor device details
```

**Linux**

```bash
sudo apt install libimobiledevice-utils gphoto2   # USB scan
sudo apt install ifuse                             # mount DCIM as filesystem
```

**Windows** — install iTunes (provides USB drivers), then mount via Windows Explorer and use `--source PATH`.

## Commands

```
imole doctor                         Check device connection and dependencies
imole scan    [filters]              List all media files on connected iPhone
imole stats   [filters]              Summary stats: total, photos, videos, old, large
imole videos  [--top N]             Ranked list of largest videos
imole backup  --to PATH [filters]   Back up matching media, write manifest.json
imole report  --manifest PATH       Summarize a backup manifest
imole clean   --manifest PATH       Delete verified files from iPhone
imole guide   [topic]               Step-by-step cleanup guide (WeChat, Telegram…)
imole history [--limit N]           Show recent backup and delete operations
imole update  [--check]            Update imole to the latest release
imole update  --nightly            Install latest unreleased build from main branch (requires go)
imole schema  [command]             Machine-readable command schema (agent-friendly)
```

**Common filters**

```bash
--only all|photos|videos
--older-than 90d|6m|1y
--large-than 500MB|1GB
--dry-run        # preview without side effects (exit 10 = safe to proceed)
--json           # force JSON output
--fields a,b     # select JSON fields (dot-path notation)
```

**Output**

JSON is emitted automatically when stdout is not a terminal. Use `--json` to force it in interactive mode. Use `--fields` to select specific fields:

```bash
imole stats  --json --fields total_size_human,video_size_human
imole report --manifest ./manifest.json --json --fields verified,cleanable_size
```

## Detailed Examples

### Diagnose what's eating space

```bash
$ imole stats

iMole Stats

Total:   38,421 files · 286.4 GB
Photos:  37,217 files · 113.6 GB
Videos:   1,204 files · 172.8 GB

$ imole videos --top 5

Top 5 videos by size

  1. IMG_8821.MOV   8.2 GB   2025-10-02
  2. IMG_7731.MOV   4.6 GB   2025-08-11
  3. IMG_6602.MOV   3.9 GB   2024-12-31
  4. IMG_5501.MOV   2.1 GB   2024-09-15
  5. IMG_4412.MOV   1.8 GB   2024-06-20
```

### Back up old videos and delete from device

```bash
# 1. Preview what will be backed up
$ imole backup --to ~/iphone-backup --only videos --older-than 90d --dry-run
Dry-run: 48 files (62.4 GB) would be copied (exit 10)

# 2. Execute backup
$ imole backup --to ~/iphone-backup --only videos --older-than 90d
Backup complete
Destination: /Users/you/iphone-backup
Selected:    48 files · 62.4 GB
Copied:      48 files · 62.4 GB
Verified:    48 files · 62.4 GB
Manifest:    /Users/you/iphone-backup/manifest.json

# 3. Delete verified files from iPhone
$ imole clean --manifest ~/iphone-backup/manifest.json
Clean plan

Manifest:       /Users/you/iphone-backup/manifest.json
Verified files: 48 (62.4 GB)

Files to delete (showing 15 of 48):
    1. IMG_8821.MOV                          8.2 GB
    2. IMG_7731.MOV                          4.6 GB
    ...

Warning: This will delete the files listed above from your iPhone.
         iMole only deletes files verified in the manifest.
         Files will remain in Recently Deleted for 30 days.

Proceed? [y/N] y
Deleting 48 files via auto provider...

Delete complete
  Deleted: 48 files · 62.4 GB

Final step to reclaim space:
  On iPhone → Photos → Albums → Recently Deleted → Delete All
  Estimated space freed after that step: ~62.4 GB
```

### Audit what iMole has done

```bash
$ imole history

iMole Operation History

  2026-05-31 02:41  backup   48 files · 62.4 GB → ~/iphone-backup
  2026-05-31 02:45  clean    48 files · 62.4 GB  [manifest: ~/iphone-backup/manifest.json]

$ imole history --json | jq '.[0]'
```

### Non-interactive usage (scripting / agents)

```bash
# Machine-readable stats
imole stats --json --fields total_size_human,video_files,old_size_human

# Full backup + clean pipeline with no prompts
imole backup --to ~/backup --only videos --older-than 90d
imole clean  --manifest ~/backup/manifest.json --yes

# Discover available flags
imole schema backup
```

## Safety Design

iMole treats iPhone media as irreplaceable data, not cache.

- **Preview first** — every destructive command supports `--dry-run`.
- **Backup before delete** — `clean` reads a `manifest.json`; it refuses to run without one.
- **Verify before delete** — only files marked `verified: true` in the manifest are eligible for deletion.
- **Audit trail** — `imole history` and `~/.local/share/imole/operations.jsonl` log every backup and delete.
- **Recently Deleted** — deleted files sit in iOS "Recently Deleted" for 30 days. iMole reminds you to clear it.
- **iCloud warning** — if iCloud Photos is enabled, deleting from iPhone also removes from iCloud. iMole warns you.

iMole cannot automatically clean:

- WeChat, Telegram, or other app sandbox storage (use `imole guide` for step-by-step instructions)
- iOS System Data
- iCloud-only content (not downloaded to the device)

## Architecture

```
cmd/imole          CLI entrypoint
internal/cli       command parsing and presentation
internal/device    local dependency and iPhone detection
internal/media     DCIM/media scanning and classification
internal/backup    copy, fast verification, manifest
internal/report    manifest summaries
internal/filter    shared filter parsing and matching
internal/provider  media backends: filesystem, gphoto2, ImageCaptureCore
internal/human     terminal formatting helpers
internal/history   operation log (backup and delete audit)
```

## Tips

- **Start with videos** — one 4K video can be larger than thousands of photos. Run `imole videos --top 20` first.
- **Use `--dry-run`** — always preview before committing. Exit code `10` means the preview passed.
- **Narrow the filter** — `--only videos --older-than 1y` recovers the most space with the least risk.
- **iCloud users** — if iCloud Photos sync is on, deleting via iMole also removes from iCloud. Back up first.
- **Linux/Windows** — mount the iPhone DCIM folder first (`ifuse` on Linux, iTunes on Windows), then pass `--source PATH`.

## Acknowledgments

iMole was inspired by [Mole](https://github.com/tw93/mole) — a fantastic macOS system cleanup tool by [@tw93](https://github.com/tw93). Mole proved that a single CLI binary can replace heavyweight GUI apps for system maintenance, and its agent-friendly design principles deeply influenced how iMole is built. If you're looking to clean up your Mac, check it out — it's excellent.

## Contributing

Issues and PRs welcome. Run `go test ./...` before submitting.

## License

MIT

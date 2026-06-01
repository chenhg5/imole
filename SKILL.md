---
name: imole
version: 0.1.0
description: |
  Open-source iPhone storage cleaner. Scan media on a connected iPhone, back up
  photos and videos to the computer, delete verified files from the device via
  ImageCaptureCore, and guide users through steps only Apple allows on the phone.
  Use when the user says their iPhone is full, wants to free up space, back up
  photos/videos, or delete old media from their phone.
allowed-tools:
  - Shell
  - Read
  - AskQuestion
---

# iMole — iPhone Storage Cleaner Skill

Help users slim down their iPhone storage. Scan media, back up files to the computer, delete verified files from the device, and guide users through the steps only Apple allows on the phone itself.

Use this skill whenever the user says: "my iPhone is full", "clean up my iPhone", "free up iPhone storage", "back up iPhone photos/videos", "iPhone space is running out", or asks to delete old videos from their phone.

---

## Platform Support

| Feature | macOS | Linux | Windows |
|---------|:-----:|:-----:|:-------:|
| USB scan (auto) | ✅ ImageCaptureCore | ✅ gphoto2 | ➖ use `--source` |
| Scan via `--source PATH` | ✅ | ✅ | ✅ |
| Backup (copy files) | ✅ | ✅ | ✅ |
| Delete from device (USB) | ✅ ImageCaptureCore | ❌ planned | ❌ not supported |
| Delete via mounted path | ✅ | ✅ (ifuse) | ✅ (iTunes mount) |
| Device detection (`doctor`) | ✅ | ✅ | ✅ |

### macOS prerequisites

- iPhone connected via USB, screen unlocked, "Trust This Computer" accepted.
- `swift` available: `xcode-select --install`
- ImageCaptureCore used automatically — no extra install needed.

### Linux prerequisites

```bash
# Device detection and trust pairing
sudo apt install libimobiledevice-utils

# Scan via USB (gphoto2 — auto-detected)
sudo apt install gphoto2

# Mount DCIM as filesystem (required for backup + delete workflow)
sudo apt install ifuse
idevicepair pair          # trust the device (one-time)
mkdir -p ~/iphone
ifuse ~/iphone            # mount iPhone DCIM
```

### Linux full cleanup workflow

```bash
# 1. Mount
ifuse ~/iphone

# 2. Scan what's on the phone
imole scan --source ~/iphone/DCIM

# 3. Backup (verifies every file with SHA-256)
imole backup --source ~/iphone/DCIM --to ~/backup/iphone --only videos --older-than 90d

# 4. Review what will be deleted
imole report --manifest ~/backup/iphone/manifest.json

# 5. Delete verified files directly from the mount (space freed immediately, no Recently Deleted)
imole clean --manifest ~/backup/iphone/manifest.json --source ~/iphone/DCIM

# 6. Unmount
fusermount -u ~/iphone
```

### Windows prerequisites

- Install **iTunes** (provides USB drivers and the Apple Mobile Device service).
- Connect iPhone, unlock it, tap "Trust This Computer".
- Open **Windows Explorer** → This PC → [Your iPhone] → Internal Storage → DCIM.
- Note the path (e.g. `\\Apple\iPhone\Internal Storage\DCIM`) or copy it to a local folder.
- Use `--source` with that path:

```powershell
imole.exe scan --source "\\Apple\iPhone\Internal Storage\DCIM"
imole.exe backup --source "\\Apple\iPhone\Internal Storage\DCIM" --to C:\backup
imole.exe clean --manifest C:\backup\manifest.json --source "\\Apple\iPhone\Internal Storage\DCIM"
```

---

## Installation

### Option A — npm (recommended — works on macOS, Linux, Windows)

```bash
npm install -g @getimole/imole
```

Works on all platforms with Node.js installed. Downloads the pre-built binary automatically. Verify with:

```bash
imole --version
```

### Option B — Install script (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

Installs the pre-built binary to `/usr/local/bin/imole`. Verify with:

```bash
imole --version
```

### Option C — Build from source

```bash
git clone https://github.com/chenhg5/imole.git
cd imole
go build -o imole ./cmd/imole
sudo mv imole /usr/local/bin/
```

---

## Full Cleanup Workflow (end-to-end)

Follow these steps in order. Never skip the backup-and-verify step before cleaning.

### Step 1 — Check device and environment

```bash
imole doctor --json
```

Expected: `"device"` block with `"connected": true`. If the device is not connected, ask the user to plug in the iPhone, unlock it, and tap "Trust" on the device screen.

Use device storage pressure to decide how aggressive the cleanup plan should be:

| Free space | Pressure | Agent response |
|---:|---|---|
| `< 5%` | Critical | Target immediate reclaim. Prefer large videos and old videos first. |
| `5-10%` | High | Recommend a concrete backup target large enough to reach at least 15% free. |
| `10-20%` | Moderate | Offer a conservative plan: oldest/large videos first, then app cleanup guidance. |
| `> 20%` | Low | Diagnose only; do not push deletion unless user asks. |

Useful fields:

```bash
imole doctor --json --fields device.name,device.product_type,device.storage.total_data_capacity,device.storage.amount_data_available,device.storage.free_percent
```

### Step 2 — Scan to understand what's taking space

```bash
imole scan --summary --json
```

Key fields in the response:

| Field | Meaning |
|---|---|
| `device.storage.free_percent` | Remaining device storage percentage |
| `media.total_size` | Total media size (photos + videos) |
| `media.video_size` | Videos only — usually the biggest offender |
| `apps.total_size` | App storage estimate from iOS installation_proxy |
| `top_video.size` | Largest video candidate |

To show only the most useful subset:

```bash
imole scan --summary --json --fields device.storage.free_percent,media.total_size,media.video_size,apps.total_size,top_video
```

To focus on videos older than 90 days:

```bash
imole scan --summary --only videos --older-than 90d --json
```

If you already ran a scan recently and don't want to wait ~15 s for USB enumeration:

```bash
imole scan --cache --summary --json
```

### Step 3 — Inspect the biggest files

```bash
imole scan --top 20 --only videos --json
```

Use this to show the user which specific videos are taking the most space. Let the user decide which age or size threshold to target.

`scan`, `scan apps`, `doctor`, `report`, `history`, `schema`, and `guide` are read-only. Do not add `--dry-run` to them; `--dry-run` is only valid on `backup` and `clean`.

### Step 4 — Dry-run the backup (preview)

Always dry-run before committing. This shows what would be backed up without touching anything.

```bash
imole backup --to ~/imole-backup --only videos --older-than 90d --dry-run
```

Exit code `10` means dry-run passed — safe to proceed. Adjust `--only`, `--older-than`, and `--large-than` based on user preference.

### Step 5 — Run the backup

```bash
imole backup --to ~/imole-backup --only videos --older-than 90d
```

This copies matching files from the iPhone to `~/imole-backup/`, verifies each file by size, and writes `~/imole-backup/manifest.json`.

The manifest records every file's `source_rel` (path on device), `dest_rel` (local copy), `size`, and `verified` flag. **Only verified files are ever eligible for deletion.**

### Step 6 — Check the backup report

```bash
imole report --manifest ~/imole-backup/manifest.json --json
```

Key fields:

| Field | Meaning |
|---|---|
| `verified` | Number of files confirmed copied correctly |
| `verified_size` | Total bytes confirmed |
| `cleanable` | Files safe to delete from iPhone |
| `cleanable_size` | Bytes that can be freed |

Only proceed to clean if `verified == cleanable` (no failures).

### Step 7 — Dry-run the deletion (preview)

```bash
imole clean --manifest ~/imole-backup/manifest.json --dry-run
```

Exit code `10` = safe to proceed.

### Step 8 — Delete verified files from iPhone

```bash
imole clean --manifest ~/imole-backup/manifest.json --yes
```

- Only files with `verified: true` in the manifest are sent to the device for deletion.
- Uses `ICCameraDevice.requestDeleteFiles` via a Swift helper (macOS ImageCaptureCore).
- The iPhone **may show a confirmation prompt** on its screen — tell the user to accept it.

### Step 9 — Clear Recently Deleted (manual, on iPhone)

Deleted files stay in "Recently Deleted" for 30 days and **still occupy space** until cleared.

Tell the user:

> On your iPhone: open **Photos** → **Albums** → scroll to **Utilities** → **Recently Deleted** → tap **Select** → **Delete All**.

Space is freed only after this step.

---

## Command Reference

### `imole doctor`

Check connectivity and dependencies.

```bash
imole doctor --json
imole doctor --json --fields device.name,device.connected
```

### `imole scan`

Unified scan command — summary, top N by size, or full result.

```bash
# Compact stats (for a quick overview)
imole scan --summary --json
imole scan --summary --only videos --older-than 1y --json
imole scan --summary --json --fields total_size_human,video_files,old_size_human

# Top N largest files (replaces old `videos` command)
imole scan --top 30 --only videos --json
imole scan --top 10 --only photos --older-than 1y --json

# Use cached scan to skip slow USB enumeration (< 1 h old)
imole scan --cache --summary --json

# Full scan report with next-step hints
imole scan --json
imole scan --only videos --older-than 90d --json
imole scan --json --fields summary.total_files,summary.video_size

# Limit result to first N items (largest first) after filtering
imole scan --only videos --limit 20 --json

# Metadata scan (GPS, taken date, dimensions) — first run ~30-60 s, cached 7 days
imole scan --with-meta --summary --json

# Filter by GPS country / region (auto-enables --with-meta)
imole scan --country Japan --only photos --json
imole scan --country CN --only videos --json

# Filter by date range (YYYY-MM-DD)
imole scan --taken-after 2024-01-01 --taken-before 2024-12-31 --only photos --json

# Videos longer than N seconds
imole scan --duration-gt 120 --only videos --json

# Photos/videos with no GPS metadata
imole scan --no-gps --only photos --json

# Filter by file extension (no --with-meta needed)
# On iPhone: .png ≈ screenshots, .heic = camera photos, .mov = videos
imole scan --ext png --json                             # likely screenshots
imole scan --ext heic --only photos --json             # HEIC camera photos only

# Filter by dimensions (requires --with-meta)
# iPhone 15 Pro screenshot = 1179×2556; iPhone 14 = 1170×2532
imole scan --with-meta --ext png --min-width 1100 --min-height 2400 --json   # precise screenshot detection
imole scan --with-meta --min-width 4000 --only photos --json                 # high-res camera shots
```

### `imole backup`

Back up media to local disk with verification.

```bash
imole backup --to /Volumes/External/iphone-backup --only videos --older-than 90d
imole backup --to ~/backup --file DCIM/202507__/IMG_7523.MOV --dry-run
imole backup --to ~/backup --only photos --older-than 1y --dry-run
imole backup --to ~/backup --large-than 500MB --json

# Back up at most N files (largest first)
imole backup --to ~/backup --only videos --limit 10 --dry-run

# Back up by GPS country (auto-enables --with-meta)
imole backup --to ~/backup --country Japan --only photos
imole backup --to ~/backup --country Australia --only videos --dry-run

# Back up by date range
imole backup --to ~/backup --taken-after 2024-01-01 --taken-before 2024-12-31 --only photos

# Back up videos longer than 2 minutes
imole backup --to ~/backup --duration-gt 120 --only videos

# Back up items with no GPS (privacy-sensitive cleanup)
imole backup --to ~/backup --no-gps --only photos

# Back up screenshots (PNG files)
imole backup --to ~/backup/screenshots --ext png --dry-run
imole backup --to ~/backup/screenshots --ext png

# Back up screenshots precisely: PNG + screen dimensions (auto-enables --with-meta)
imole backup --to ~/backup/screenshots --ext png --min-width 1100 --min-height 2400
```

Use `--file REL_PATH` when the user chooses an exact item from `imole scan --top ...` output. The value should be the item's `rel_path`, and `--file` can be repeated.

### `imole report`

Summarise a manifest file.

```bash
imole report --manifest ~/backup/manifest.json --json
imole report --manifest ~/backup/manifest.json --json --fields verified,cleanable_size
```

### `imole clean`

Delete verified-backup files from iPhone.

```bash
imole clean --manifest ~/backup/manifest.json --dry-run   # preview
imole clean --manifest ~/backup/manifest.json --file DCIM/202507__/IMG_7523.MOV --dry-run
imole clean --manifest ~/backup/manifest.json --yes        # delete without prompt
imole clean                                                 # show recommended flow
```

Use `clean --file REL_PATH` only to narrow deletion to a file that is already present and verified in the manifest. It must never be described as direct arbitrary device-file deletion.

### `imole guide`

Step-by-step guidance for app cache and system data that iMole cannot auto-clean.

```bash
imole guide
imole guide analysis
imole guide wechat
imole guide telegram
imole guide system-data
```

If the agent does not have this skill document loaded, it can recover the same diagnosis workflow from:

```bash
imole guide analysis
```

### `imole uninstall`

Remove a **user-installed** app from the iPhone. System apps are protected and cannot be uninstalled.

Requires `IMOLE_NO_DELETE` to be **unset** (same guard as `clean`). Get bundle IDs from `imole scan apps --json`.

```bash
# Dry-run — preview what would be uninstalled
imole uninstall --bundle-id com.example.myapp --dry-run

# Uninstall with prompt
imole uninstall --bundle-id com.example.myapp

# Skip confirmation (scripting)
imole uninstall --bundle-id com.example.myapp --yes
```

> Never use `uninstall` on system apps or apps critical to device function. iMole blocks known system bundle ID prefixes (`com.apple.*`, `io.appstore`, etc.).

### `imole schema`

Machine-readable command schema (flags, types, defaults). Use this to discover available flags.

```bash
imole schema
imole schema scan
imole schema backup
imole schema clean
imole schema uninstall
```

---

## Agent Decision Logic

Use the following decision tree when helping a user free iPhone storage:

```
1. Run `imole doctor --json`
   → device not connected?  Ask user to connect, unlock, and trust the Mac.
   → read device.storage.free_percent and amount_data_available.
   → set a target: reach 15-20% free, or reclaim at least 10 GiB if critically low.

2. Run `imole scan --summary --json`
   → compare media.video_size, media.photo_size, apps.total_size, and device free space.
   → if video_size is large enough to meet the target, recommend videos first.
   → if app storage dominates, run `imole scan apps --top 20 --json` and give in-app cleanup guidance.
   → tip: use --cache to skip USB scan if result is already fresh.
   → do not add --dry-run; scan is read-only.

3. Run `imole scan --top 20 --only videos --json` (optional)
   → show the user which specific files are eating space.
   → recommend one or two filters that meet the target:
     `--older-than 1y`, `--older-than 6m`, or `--large-than 500MB`.
   → ask for confirmation before backup/clean.

4. Ask the user which files to target:
   - Old videos (--only videos --older-than 90d)?
   - Large files (--large-than 500MB)?
   - Everything (omit --only)?
   Prefer the smallest-risk filter that reaches the reclaim target:
   old videos > large videos > all videos > photos.

5. Run `imole backup --to ~/imole-backup [filters] --dry-run`
   → confirm count and size with user.
   → exit 10 = safe to continue.

6. Run `imole backup --to ~/imole-backup [filters]`
   → check `manifest.summary.failed_files == 0`.
   → if failures > 0: warn user, do NOT proceed with clean.

7. Run `imole report --manifest ~/imole-backup/manifest.json --json`
   → confirm cleanable == verified.

8. Run `imole clean --manifest ~/imole-backup/manifest.json --dry-run`
   → show user what will be deleted.

9. Run `imole clean --manifest ~/imole-backup/manifest.json --yes`
   → tell user to accept the iPhone prompt if one appears.

10. Remind user to clear Recently Deleted in Photos on the iPhone.
```

---

## Limitations

| Limitation | Detail |
|---|---|
| **USB delete: macOS only** | `imole clean --manifest` deletes via ImageCaptureCore (macOS). On Linux/Windows use `--source PATH` (ifuse or iTunes mount) for filesystem deletion — space is freed immediately, no "Recently Deleted" buffer. |
| **USB scan: Linux needs gphoto2** | Install `gphoto2`; or mount with `ifuse` and use `--source PATH`. |
| **Windows: use `--source PATH`** | Auto USB scan not supported; mount DCIM via iTunes/Windows Explorer. |
| **Recently Deleted** | Deleted files occupy space for 30 days until the user clears them manually (Photos → Albums → Recently Deleted → Delete All). |
| **iCloud sync** | If iCloud Photos is on, deleting from iPhone also removes from iCloud. Warn the user before proceeding. |
| **App caches** | WeChat, Telegram, Spotify downloads cannot be touched via USB. Use `imole guide` for instructions. |
| **iPhone prompt (macOS)** | iOS may display "Allow [Computer] to delete photos?" — the user must tap Allow on the device. |
| **iMole only deletes verified files** | The `clean` command refuses to delete any file not marked `verified: true` in the manifest. |

---

## Common Scenarios

### "My iPhone is full and I don't know what's eating space"

```bash
imole doctor --json
imole scan --summary --json
imole scan --top 10 --only videos --json
```

Show the user the numbers, then ask which files they want to back up and remove.

### "Back up and delete videos older than 6 months"

```bash
imole backup --to ~/imole-backup --only videos --older-than 6m --dry-run
imole backup --to ~/imole-backup --only videos --older-than 6m
imole report --manifest ~/imole-backup/manifest.json --json
imole clean  --manifest ~/imole-backup/manifest.json --yes
```

Then remind user to clear Recently Deleted.

### "Just back up, I'll delete myself"

```bash
imole backup --to ~/imole-backup --only videos --older-than 90d
```

Done. User can then delete files manually in Image Capture or Photos on the Mac.

### "What about WeChat / app data?"

```bash
imole guide wechat
```

iMole cannot auto-clean app sandboxes. The guide gives step-by-step instructions.

### "Find and back up screenshots"

iPhone screenshots are **always PNG**; camera photos are HEIC or JPEG. `--ext png` is therefore a reliable screenshot filter (high-confidence, not absolute — PNG files received via AirDrop or messaging would also match).

For near-certain identification, combine with screen-dimension filters (requires `--with-meta`, cached after first run):

```bash
# Quick count — no metadata needed
imole scan --ext png --json

# Precise: PNG + screen height ≥ 2400 px (covers all modern iPhones)
imole scan --ext png --min-width 1100 --min-height 2400 --json

# Back up screenshots (fast, extension only)
imole backup --to ~/iphone-backup/screenshots --ext png --dry-run
imole backup --to ~/iphone-backup/screenshots --ext png

# Back up screenshots (precise, with dimensions)
imole backup --to ~/iphone-backup/screenshots \
  --ext png --min-width 1100 --min-height 2400 --dry-run
imole backup --to ~/iphone-backup/screenshots \
  --ext png --min-width 1100 --min-height 2400
```

Common iPhone screen resolutions for reference:

| Model | Resolution |
|---|---|
| iPhone 16 Pro | 1206 × 2622 |
| iPhone 15 Pro | 1179 × 2556 |
| iPhone 14 / 15 | 1170 × 2532 |
| iPhone SE (3rd) | 750 × 1334 |

### "Back up photos from a specific country / trip"

```bash
# See what GPS countries are represented
imole scan --with-meta --only photos --json --fields items[].country

# Back up Japan trip photos
imole backup --to ~/imole-backup/japan --country Japan --only photos --dry-run
imole backup --to ~/imole-backup/japan --country Japan --only photos

# Back up by date range (e.g. a holiday period)
imole backup --to ~/imole-backup/holiday --taken-after 2024-12-20 --taken-before 2025-01-05 --only photos
```

### "Back up just a few of the largest videos (test run)"

```bash
imole backup --to ~/imole-backup --only videos --limit 5 --dry-run
```

### "Remove an app from iPhone"

```bash
# Find the bundle ID first
imole scan apps --json --top 20

# Dry-run to confirm
imole uninstall --bundle-id com.example.myapp --dry-run

# Uninstall
imole uninstall --bundle-id com.example.myapp
```

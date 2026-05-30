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

# Mount DCIM as filesystem (for --source PATH workflow)
sudo apt install ifuse
ifuse ~/iphone         # mount
imole scan --source ~/iphone/DCIM
fusermount -u ~/iphone  # unmount when done
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
```

---

## Installation

### Option A — Install script (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/imole/main/install.sh | bash
```

Installs the pre-built binary to `/usr/local/bin/imole`. Verify with:

```bash
imole --version
```

### Option B — Build from source

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

### Step 2 — Scan to understand what's taking space

```bash
imole stats --json
```

Key fields in the response:

| Field | Meaning |
|---|---|
| `total_size_human` | Total media size (photos + videos) |
| `video_size_human` | Videos only — usually the biggest offender |
| `old_size_human` | Media older than the `--older-than` threshold |
| `large_size_human` | Files above the `--large-than` threshold |

To show only the most useful subset:

```bash
imole stats --json --fields total_size_human,video_size_human,old_size_human
```

To focus on videos older than 90 days:

```bash
imole stats --only videos --older-than 90d --json
```

### Step 3 — Inspect the biggest files

```bash
imole videos --top 20 --json
```

Use this to show the user which specific videos are taking the most space. Let the user decide which age or size threshold to target.

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

### `imole stats`

Agent-friendly summary with pre-computed human-readable sizes. Prefer this over `scan` when you only need totals.

```bash
imole stats --json
imole stats --only videos --older-than 1y --json
imole stats --json --fields total_size_human,video_files,old_size_human
```

### `imole scan`

Full file list with per-item metadata.

```bash
imole scan --json
imole scan --only videos --older-than 90d --json
imole scan --json --fields summary.total_files,summary.video_size
```

### `imole videos`

Ranked list of largest video files.

```bash
imole videos --top 30 --json
imole videos --older-than 1y --top 10 --json
```

### `imole backup`

Back up media to local disk with verification.

```bash
imole backup --to /Volumes/External/iphone-backup --only videos --older-than 90d
imole backup --to ~/backup --only photos --older-than 1y --dry-run
imole backup --to ~/backup --large-than 500MB --json
```

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
imole clean --manifest ~/backup/manifest.json --yes        # delete without prompt
imole clean                                                 # show recommended flow
```

### `imole guide`

Step-by-step guidance for app cache and system data that iMole cannot auto-clean.

```bash
imole guide
imole guide wechat
imole guide telegram
imole guide system-data
```

### `imole schema`

Machine-readable command schema (flags, types, defaults). Use this to discover available flags.

```bash
imole schema
imole schema backup
imole schema clean
```

---

## Agent Decision Logic

Use the following decision tree when helping a user free iPhone storage:

```
1. Run `imole doctor --json`
   → device not connected?  Ask user to connect, unlock, and trust the Mac.

2. Run `imole stats --json`
   → show total_size_human, video_size_human to the user.
   → if video_size > 5 GB: recommend --only videos first.

3. Ask the user which files to target:
   - Old videos (--only videos --older-than 90d)?
   - Large files (--large-than 500MB)?
   - Everything (omit --only)?

4. Run `imole backup --to ~/imole-backup [filters] --dry-run`
   → confirm count and size with user.
   → exit 10 = safe to continue.

5. Run `imole backup --to ~/imole-backup [filters]`
   → check `manifest.summary.failed_files == 0`.
   → if failures > 0: warn user, do NOT proceed with clean.

6. Run `imole report --manifest ~/imole-backup/manifest.json --json`
   → confirm cleanable == verified.

7. Run `imole clean --manifest ~/imole-backup/manifest.json --dry-run`
   → show user what will be deleted.

8. Run `imole clean --manifest ~/imole-backup/manifest.json --yes`
   → tell user to accept the iPhone prompt if one appears.

9. Remind user to clear Recently Deleted in Photos on the iPhone.
```

---

## Limitations

| Limitation | Detail |
|---|---|
| **USB delete: macOS only** | `imole clean --manifest` deletes via ImageCaptureCore (macOS). On Linux mount with `ifuse` and delete from the mounted path. On Windows not supported via USB. |
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
imole stats --json
imole videos --top 10 --json
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

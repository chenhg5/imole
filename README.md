<div align="center">
  <h1>iMole</h1>
  <p><em>Open-source iPhone slimming toolkit from the terminal.</em></p>
</div>

iMole helps you inspect visible iPhone media, find large videos, back up selected
photos/videos to your computer, verify the copy, and produce a cleanup plan.

It is intentionally conservative. iMole v0.1 does not automatically delete your
iPhone Photos library. iOS media deletion is gated by Photos, Image Capture, and
iCloud behavior, so the first release focuses on diagnosis, backup, verification,
and clear guidance.

## Install

```bash
go install github.com/chenhg5/imole/cmd/imole@latest
```

For local development:

```bash
make build
./bin/imole help
```

## Dependencies

For iPhone device detection:

```bash
brew install libimobiledevice
```

Optional fallback for PTP/camera-style media enumeration:

```bash
brew install gphoto2
```

On macOS, iMole is moving toward Apple ImageCaptureCore as the primary media
provider so users do not need macFUSE/ifuse. You can also pass `--source PATH`
to scan an existing media directory or a fixture directory.

The ImageCaptureCore provider is experimental in v0.1. Run it from the Mac's
Terminal app, not an SSH session. SSH can detect the iPhone through
`libimobiledevice` while ImageCaptureCore still returns an empty catalog because
it is tied to the interactive macOS user session and privacy prompts.

## Commands

```bash
imole doctor
imole scan --provider auto
imole videos --top 50
imole backup --provider imagecapture --to /Volumes/Backup/iPhone --only videos --older-than 90d
imole report --manifest /Volumes/Backup/iPhone/manifest.json
imole guide
imole clean
```

Common filters:

```bash
--only all|photos|videos
--provider auto|filesystem|imagecapture|gphoto
--source PATH
--older-than 90d
--large-than 500MB
--dry-run
--json
```

## Strategy

iMole follows the same safety-first spirit as Mole:

- Preview and report before destructive operations.
- Treat user photos/videos as irreplaceable data, not cache.
- Back up and verify before recommending deletion.
- Keep command code thin and move behavior into focused internal packages.
- Make unsupported iOS areas explicit instead of pretending they can be cleaned.

## Architecture

```text
cmd/imole          CLI entrypoint
internal/cli       command parsing and presentation
internal/device    local dependency and iPhone detection
internal/media     DCIM/media scanning and classification
internal/backup    copy, fast verification, manifest
internal/report    manifest summaries
internal/filter    shared filter parsing and matching
internal/provider  media backends: filesystem, gphoto, ImageCaptureCore
internal/human     terminal formatting helpers
```

## Safety Boundaries

iMole v0.1 can automatically:

- Detect local tools and connected devices.
- Scan visible DCIM media.
- Find large videos and old media.
- Back up selected media to local storage.
- Verify copied files by size.
- Write a `manifest.json` for auditability.

iMole v0.1 does not automatically:

- Delete iPhone Photos library content.
- Clean WeChat or other app-private storage.
- Clean iOS System Data.
- Bypass iCloud Photos synchronization rules.
- Access non-media app sandboxes unless iOS exposes them.

Use Apple Image Capture or Photos to delete imported items after you have a
verified backup, then empty Recently Deleted on the iPhone.

For a small first real backup test, use a narrow filter:

```bash
imole backup --provider imagecapture --to ~/code/imole/testdata/ic-backup --only videos --large-than 330MB
```

# iMole Security Audit

This document describes the security-relevant behavior of the current `main` branch, updated for v0.1.0 on 2026-05-31.

## Executive Summary

iMole is a local iPhone media management tool. Its main risk surface is unintended data loss caused by deletion of media files from a connected iPhone.

The project is designed around safety-first defaults:

- deletion only happens after verified backup
- only files marked `verified: true` in `manifest.json` are eligible for deletion
- all destructive commands support `--dry-run` preview
- manifest is required for any deletion operation
- operation logging provides an audit trail

iMole prioritizes conservative cleanup over aggressive cleanup. When uncertainty exists, the tool should refuse or skip instead of widening deletion scope.

## Threat Surface

The highest-risk areas in iMole are:

- deletion of media files from a connected iPhone over USB
- manifest manipulation that could trick deletion logic
- path traversal during backup copy
- USB protocol handling that could corrupt device data
- release, install, and update trust signals for distributed artifacts

## Destructive Operation Boundaries

Core controls include:

- **Verified-only deletion**: `provider.Delete()` only sends files with `verified: true` in the manifest to the device for deletion
- **Manifest required**: `clean` refuses to run without a `--manifest` pointing to a valid `manifest.json`
- **Dry-run**: `clean --dry-run` previews what would be deleted without making changes (exit 10 = safe to proceed)
- **Path validation**: backup copy uses `filepath.Abs()` and rejects relative paths and `..` traversal
- **Operation logging**: `internal/history/` records every backup and delete with timestamp, file count, size, and manifest path

## USB Operation Safety

- ImageCaptureCore (macOS) uses Apple's official API for device communication
- Files are copied first, verified by size, then the manifest is written
- Only after manifest creation can deletion be attempted
- Device-side deletion uses `ICCameraDevice.requestDeleteFiles` — iOS may prompt the user to confirm

## Manifest Integrity

The manifest (`manifest.json`) is the source of truth for what was backed up and verified:

```json
{
  "version": 1,
  "created_at": "...",
  "root": "/path/to/DCIM",
  "files": [
    {
      "source_rel": "IMG_1234.MOV",
      "dest_rel": "2025/01/IMG_1234.MOV",
      "size": 12345678,
      "verified": true,
      "error": ""
    }
  ]
}
```

Only `verified: true` entries are eligible for deletion. The manifest is written after copy + verify completes, not before.

## Known Limitations

- Linux/Windows do not have USB auto-scan (`imagecapture` provider unavailable)
- Device deletion via USB is macOS-only (ImageCaptureCore API)
- Recently Deleted: iOS keeps deleted files for 30 days — iMole reminds users to clear this manually
- iCloud Photos: if enabled, deleting from device also removes from iCloud — iMole warns before this

## Release Integrity

- CI builds are triggered by version tags (`v*`)
- Each release upload includes a `SHA256SUMS` file
- Homebrew tap formula is updated automatically by CI with version and checksums

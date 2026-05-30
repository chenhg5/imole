# iMole Agent Guide

This file is the shared source of truth for any AI agent working on this repo (Claude Code, Codex, etc.). `CLAUDE.md` is a symlink to this file.

## Project

iMole is a Go CLI for backing up and cleaning up iPhone media over USB. It scans media, copies files to a local path with verification, and can delete verified files from the device. Safety-first design: delete only happens after verified backup.

## Repository Map

```
cmd/imole/          CLI entrypoint
internal/cli/       command parsing, help, schema, error handling
internal/backup/    copy + verify + manifest
internal/filter/    size/age/kind filtering
internal/media/      DCIM scanning and classification
internal/provider/   backends: filesystem, gphoto2, ImageCaptureCore
internal/device/     dependency and device detection
internal/report/     manifest summaries
internal/history/    operation log (audit trail)
internal/human/      terminal formatting helpers
install.sh          cross-platform installer
Makefile            build targets
SKILL.md            agent skill prompt for using imole
```

## Commands

```bash
make build           # build locally
make release        # build all platforms
go test ./...       # run tests
go vet ./...        # vet
gofmt -w cmd internal  # format
```

## Critical Safety Rules

- Never delete files that are not verified in a manifest.
- Delete operations must go through `provider.Delete()` only.
- All destructive commands support `--dry-run` — always verify with dry-run first.
- Path validation must reject `..` traversal, relative paths, and empty paths.
- Only files marked `verified: true` in `manifest.json` are eligible for deletion.

## Working Rules

- Use `make build` and `go test ./...` before committing.
- Keep Go code formatted with `gofmt -w cmd internal`.
- New provider backends must implement the `Provider` interface in `internal/provider/provider.go`.
- Add tests for new functionality in the same package.
- Operation logging through `internal/history/` should be preserved for auditability.

## Hotspot Ownership

- `internal/provider/provider.go` — provider interface and dispatch. Test with actual device or `--source PATH`.
- `internal/backup/backup.go` — copy, verify, manifest write. Test with dry-run first.
- `internal/cli/` — command parsing. All commands must return semantic exit codes.

## Verification

```bash
go build ./...
go test ./...
go vet ./...
gofmt -w cmd internal
```

## Release

Tag-driven. Push `v*` tag → CI builds all platforms → GitHub Release → Homebrew tap update.

```
git tag v0.1.0
git push origin v0.1.0
```

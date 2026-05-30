# Security Policy

iMole is a local iPhone media management tool. It performs backup, verification, and deletion of media files from a connected iPhone. Safety boundaries and deletion logic are security-sensitive areas.

## Reporting a Vulnerability

Please report suspected security issues privately.

- Email: the maintainer (see GitHub profile)
- Subject line: `iMole security report`

Do not open a public GitHub issue for an unpatched vulnerability.

Include as much of the following as possible:

- iMole version and install method
- macOS / OS version
- Exact command or workflow involved
- Reproduction steps or proof of concept
- Whether the issue involves deletion boundaries, path validation, USB protocol, or release/install integrity

## Response Expectations

- We aim to acknowledge new reports within 7 calendar days.
- We aim to provide a status update within 30 days if a fix or mitigation is not yet available.
- We will coordinate disclosure after a fix, mitigation, or clear user guidance is ready.

## Supported Versions

Security fixes are only guaranteed for:

- The latest published release
- The current `main` branch

## What We Consider a Security Issue

Examples of security-relevant issues include:

- Path validation bypasses (deleting outside intended cleanup boundaries)
- Deletion of unverified files
- USB protocol handling that could corrupt device data
- Manifest tampering that could trick deletion logic
- Release, installation, or checksum integrity issues
- Privilege escalation through sudo or device authorization

## What Usually Does Not Qualify

The following are normal bugs, feature requests, or documentation issues:

- Cleanup misses that leave recoverable junk behind
- False negatives where iMole refuses to delete something
- Cosmetic UI problems
- Requests for broader cleanup behavior
- Compatibility issues without a plausible security impact

If you are unsure whether something is security-relevant, report it privately first.

## Security-Focused Design

iMole treats iPhone media as irreplaceable data:

- **Verified-only deletion**: only files marked `verified: true` in `manifest.json` are eligible for deletion
- **Manifest required**: `clean` refuses to run without a manifest
- **Dry-run first**: all destructive commands support `--dry-run` preview
- **Audit trail**: `imole history` logs every backup and delete operation
- **Cross-platform safety**: Linux/Windows require explicit `--source PATH` (no USB auto-scan)

For the current technical design and known limitations, see `SECURITY_AUDIT.md`.

# imole

🐹 Back up, clean, and slim down your iPhone from the terminal.

## Install

```bash
npm install -g imole
```

Requires [Node.js](https://nodejs.org) for the install script. The actual binary has no runtime dependencies.

## Usage

```bash
imole doctor           # check device connection
imole scan --summary   # see what's eating space
imole backup --to ~/iphone-backup --only videos  # back up videos
imole clean --manifest ~/iphone-backup/manifest.json  # delete verified files
```

## Supported Platforms

- macOS (darwin) x64 / arm64
- Linux x64 / arm64
- Windows x64

## Details

The npm package downloads the pre-built binary from GitHub Releases during `npm install`. The binary is extracted to `node_modules/.bin/imole` and ready to use immediately.

On macOS, the quarantine attribute is automatically removed so the binary can run without Gatekeeper complaints.
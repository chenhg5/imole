#!/usr/bin/env node

"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");
const https = require("https");
const http = require("http");

const PACKAGE = require("./package.json");
const VERSION = `v${PACKAGE.version}`;
const NAME = "imole";

const GITHUB_REPO = "chenhg5/imole";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

function getPlatformInfo() {
  const platform = PLATFORM_MAP[process.platform];
  const arch = ARCH_MAP[process.arch];
  if (!platform || !arch) {
    throw new Error(
      `Unsupported platform: ${process.platform}/${process.arch}. ` +
        `Supported: darwin/linux/win32 x64/arm64`
    );
  }
  const ext = platform === "windows" ? ".exe" : "";
  const filename = `${NAME}-${platform}-${arch}${ext}`;
  const binaryName = platform === "windows" ? `${NAME}.exe` : NAME;
  return { platform, arch, ext, filename, binaryName };
}

function fetch(url, redirects = 5) {
  return new Promise((resolve, reject) => {
    if (redirects <= 0) return reject(new Error("Too many redirects"));
    const mod = url.startsWith("https") ? https : http;
    mod
      .get(url, { headers: { "User-Agent": "imole-npm" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return resolve(fetch(res.headers.location, redirects - 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`HTTP ${res.statusCode} for ${url}`));
        }
        const chunks = [];
        res.on("data", (c) => chunks.push(c));
        res.on("end", () => resolve(Buffer.concat(chunks)));
        res.on("error", reject);
      })
      .on("error", reject);
  });
}

async function download(url) {
  console.log(`[@getimole/imole] Downloading ${NAME} ${VERSION} for ${process.platform}/${process.arch}...`);
  try {
    const data = await fetch(url);
    console.log(`[@chenhg5/imole] Downloaded ${(data.length / 1024 / 1024).toFixed(1)} MB`);
    return data;
  } catch (err) {
    throw new Error(
      `[@getimole/imole] Could not download binary.\n` +
        `  ${err.message}\n` +
        `  You can download manually from https://github.com/${GITHUB_REPO}/releases/tag/${VERSION}`
    );
  }
}

function extractTarGz(buffer, destDir, binaryName) {
  const tmpFile = path.join(destDir, "_tmp.tar.gz");
  fs.writeFileSync(tmpFile, buffer);
  try {
    execSync(`tar xzf "${tmpFile}" -C "${destDir}"`, { stdio: "pipe" });
  } finally {
    fs.unlinkSync(tmpFile);
  }
  const entries = fs.readdirSync(destDir);
  const extracted = entries.find(
    (f) => f.startsWith(NAME) && !f.endsWith(".tar.gz") && !f.endsWith(".zip")
  );
  if (extracted && extracted !== binaryName) {
    fs.renameSync(path.join(destDir, extracted), path.join(destDir, binaryName));
  }
}

function extractZip(buffer, destDir, binaryName) {
  const tmpFile = path.join(destDir, "_tmp.zip");
  fs.writeFileSync(tmpFile, buffer);
  try {
    try {
      execSync(`unzip -o "${tmpFile}" -d "${destDir}"`, { stdio: "pipe" });
    } catch {
      execSync(
        `powershell -Command "Expand-Archive -Force '${tmpFile}' '${destDir}'"`,
        { stdio: "pipe" }
      );
    }
  } finally {
    try { fs.unlinkSync(tmpFile); } catch {}
  }
  const entries = fs.readdirSync(destDir);
  const extracted = entries.find((f) => f.startsWith(NAME) && f.endsWith(".exe"));
  if (extracted && extracted !== binaryName) {
    fs.renameSync(path.join(destDir, extracted), path.join(destDir, binaryName));
  }
}

async function main() {
  const { platform, arch, ext, filename, binaryName } = getPlatformInfo();
  const binDir = path.join(__dirname, "bin");
  fs.mkdirSync(binDir, { recursive: true });

  const binaryPath = path.join(binDir, binaryName);

  if (fs.existsSync(binaryPath)) {
    try {
      const out = execSync(`"${binaryPath}" --version`, {
        encoding: "utf8",
        timeout: 5000,
      });
      const expectedVer = VERSION.slice(1);
      if (out.includes(expectedVer)) {
        console.log(`[@getimole/imole] Binary ${VERSION} already installed, skipping.`);
        return;
      }
      console.log(`[@getimole/imole] Existing binary is outdated, upgrading to ${VERSION}...`);
      fs.unlinkSync(binaryPath);
    } catch {
      console.log(`[@getimole/imole] Replacing existing binary with ${VERSION}...`);
      fs.unlinkSync(binaryPath);
    }
  }

  const url = `https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${filename}`;
  const data = await download(url);

  fs.writeFileSync(binaryPath, data);

  if (platform !== "windows") {
    fs.chmodSync(binaryPath, 0o755);
  }

  if (platform === "darwin") {
    try {
      execSync(`xattr -d com.apple.quarantine "${binaryPath}"`, { stdio: "pipe" });
      console.log(`[@getimole/imole] Removed macOS quarantine attribute`);
    } catch {
      // xattr fails if the attribute doesn't exist, which is fine
    }
  }

  console.log(`[@getimole/imole] Installed to ${binaryPath}`);
}

main().catch((err) => {
  console.error(err.message);
  process.exit(1);
});
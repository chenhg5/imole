#!/usr/bin/env node

"use strict";

const path = require("path");
const { spawn } = require("child_process");

const binDir = path.join(__dirname, "bin");
const platform = process.platform === "win32" ? "windows" : process.platform;
const arch = process.arch === "x64" ? "amd64" : process.arch;
const binaryName = platform === "windows" ? "imole.exe" : "imole";
const binaryPath = path.join(binDir, binaryName);

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: "inherit",
  cwd: process.cwd(),
});

child.on("exit", (code) => {
  process.exit(code || 0);
});
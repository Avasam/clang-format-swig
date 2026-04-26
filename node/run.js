#!/usr/bin/env node
// Resolve the platform binary from the installed optional dependency.

"use strict";

const { spawnSync } = require("child_process");

/** @type {Record<string, string>} */
const PLATFORM_PACKAGES = {
  "linux-x64": "@clang-format-swig/linux-x64",
  "linux-arm64": "@clang-format-swig/linux-arm64",
  "darwin-x64": "@clang-format-swig/darwin-x64",
  "darwin-arm64": "@clang-format-swig/darwin-arm64",
  "win32-x64": "@clang-format-swig/win32-x64",
};

const key = `${process.platform}-${process.arch}`;
const pkg = PLATFORM_PACKAGES[key];
if (!pkg) {
  console.error(`clang-format-swig: unsupported platform ${key}.`);
  process.exit(1);
}

let binary;
try {
  binary = require(pkg); // each platform package re-exports its binary path
} catch {
  console.error(
    `clang-format-swig: ${pkg} is not installed. Reinstall without --no-optional.`,
  );
  process.exit(1);
}

const result = spawnSync(binary, process.argv.slice(2), { stdio: "inherit" });
if (result.signal) process.kill(process.pid, result.signal);
process.exit(result.status ?? 1);

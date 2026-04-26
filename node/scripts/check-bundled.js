#!/usr/bin/env node
// Refuse to publish a platform package without its bundled native binary.

"use strict";

const fs = require("fs");
const path = require("path");

const candidates = ["clang-format-swig", "clang-format-swig.exe"];
const found = candidates.some((name) => fs.existsSync(path.join(process.cwd(), name)));

if (!found) {
  console.error(
    `clang-format-swig binary missing in ${process.cwd()}\n`
    + `Expected one of: ${candidates.join(", ")}\n`
    + `The release workflow must download and extract the binary before publish.`,
  );
  process.exit(1);
}

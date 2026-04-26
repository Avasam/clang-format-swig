# clang-format-swig

`clang-format` for [SWIG](https://www.swig.org/) `.i` interface files.

## Why

SWIG `.i` files are mostly C/C++ with a handful of SWIG-specific directives sprinkled in (lines that start with `%`):

```c
%module mylib

%{
#include "mylib.h"
%}

%include "typemaps.i"

%typemap(in) MyStruct * {
    if (!convert($input, &$1)) return NULL;
}
```

Running `clang-format` directly on these files produces broken output because:

- `%{` and `%}` contain braces that shift clang-format's tracked brace depth, causing everything after them to be indented as if inside a block.
- `%module` is parsed as a C++20 module declaration, altering scope state for all subsequent code.

The C/C++ content (the bulk of every `.i` file) ends up incorrectly indented or not formatted at all.

## How it works

Before passing the file to `clang-format`, each `%`-prefixed line is swapped for a `#pragma` placeholder:

| Original         | Placeholder             |
| ---------------- | ----------------------- |
| `%module my_lib` | `#pragma SWIG_3F9A12_0` |
| `%{`             | `#pragma SWIG_3F9A12_1` |
| `%}`             | `#pragma SWIG_3F9A12_2` |

`#pragma` lines are preserved verbatim by clang-format and have no effect on brace depth or scope, so all surrounding C/C++ is formatted correctly. Afterwards the placeholders are replaced back with the original `%` lines.

Your `.clang-format` config is fully respected: `clang-format-swig` is a thin wrapper, not a reimplementation. It is written in Go and compiles to a single static binary, so it can ship through any package registry without requiring a runtime or build toolchain on the target system.

## Installation

Pick whichever channel fits your stack, every install resolves to the same Go binary.\
If there's no distribution available for your platform architecture, you can [open a feature request](https://github.com/Avasam/clang-format-swig/issues).

<!-- Keep below message in sync with CONTRIBUTING.md -->
`clang-format` must be installed separately and be on `PATH`. Install it however you'd normally get LLVM tooling on your platform (e.g. `apt install clang-format`, `brew install clang-format`, `winget install LLVM.LLVM`, `uv tool install clang-format`, `npm install -g clang-format`, ...).

### Go

```sh
go install github.com/Avasam/clang-format-swig@latest
```

Requires Go (see [`go.mod`](go.mod) for the minimum version).

### pre-commit

Add to your `.pre-commit-config.yaml`:

```yaml
# Requires having clang-format pre-installed and available on PATH
# Has go installation and build overhead
- repo: https://github.com/Avasam/clang-format-swig
  rev: v...
  hooks:
    - id: clang-format-swig
```

or

```yaml
# Has Python installation overhead
- repo: local
  hooks:
    - id: clang-format-swig
      name: clang-format SWIG files
      language: python
      entry: clang-format-swig
      files: \.i$
      additional_dependencies:
        - clang-format-swig==...
        - clang-format==...
```

### Python

```sh
uv tool install clang-format-swig
```

The platform binary is bundled in the wheel.

### Node.js

```sh
npm install -g clang-format-swig
```

The platform binary is fetched as an [optional dependency](https://docs.npmjs.com/cli/v10/configuring-npm/package-json#optionaldependencies), so no install-time scripts run.

### .NET

```sh
dotnet tool install -g clang-format-swig
```

See [`clang-format-swig.csproj`](dotnet/clang-format-swig.csproj) for the minimum .NET runtime. The platform binary is embedded in the package and extracted to your user cache on first run.

## Usage

```sh
# Format files in-place
clang-format-swig src/mylibrary.i

# Format multiple files
clang-format-swig **/*.i

# Check without writing (useful in CI)
clang-format-swig --check **/*.i

# Print version
clang-format-swig --version
```

`clang-format-swig` exits `0` if no files were changed (or `--check` found nothing to change), and `1` if any file was reformatted (or would be reformatted under `--check`).

`clang-format` itself must be on `PATH`; the wrapper does not vendor it.

### pre-commit usage

Once the hook is configured, it runs automatically on staged `.i` files:

```sh
pre-commit run clang-format-swig --all-files
```

---

*This project was initially scaffolded with Claude (Sonnet 4.6 and Opus 4.7). Every change is reviewed by a human before being merged.*

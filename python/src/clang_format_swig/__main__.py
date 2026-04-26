"""Run the clang-format-swig binary bundled in this wheel."""

import subprocess
import sys
from pathlib import Path

BIN_NAME = "clang-format-swig.exe" if sys.platform == "win32" else "clang-format-swig"
BIN_PATH = Path(__file__).parent / "_bin" / BIN_NAME


def main() -> None:
    if not BIN_PATH.exists():
        raise RuntimeError(
            f"{BIN_NAME} is missing from the installed wheel at {BIN_PATH}. Reinstall the package."
        )
    # BIN_PATH is bundled in the wheel (fixed at install time, not user input),
    # and we forward the user's own argv unchanged with no shell
    # So S603 (untrusted input) doesn't apply here.
    sys.exit(subprocess.call([str(BIN_PATH), *sys.argv[1:]]))  # noqa: S603


if __name__ == "__main__":
    main()

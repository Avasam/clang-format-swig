"""Hatchling build hook: mark wheel as platform-specific and set its tag."""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any

from hatchling.builders.hooks.plugin.interface import BuildHookInterface


class CustomBuildHook(BuildHookInterface):
    def initialize(self, version: str, build_data: dict[str, Any]) -> None:
        bin_dir = Path(self.root) / "src" / "clang_format_swig" / "_bin"
        if not any(bin_dir.glob("clang-format-swig*")):
            return
            # Somehow this prevents uv sync even with `--no-install-project --no-editable`
            raise RuntimeError(
                "sdist not supported, "
                + "clang-format-swig executable must be prebuilt and found under _bin"
            )
        build_data["pure_python"] = False
        # WHEEL_PLATFORM_TAG is set by the release CI
        build_data["tag"] = f"py3-none-{os.environ['WHEEL_PLATFORM_TAG']}"

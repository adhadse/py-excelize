"""Build hook: compile the Go c-shared library and tag the wheel per platform.

The wheel contains a native library loaded through ctypes, so it does not
depend on the Python ABI: it is tagged ``py3-none-<platform>``.
"""

import os
import shutil
import subprocess
import sys
from pathlib import Path

from setuptools import Distribution, setup
from setuptools.command.build_py import build_py

try:  # setuptools >= 70.1
    from setuptools.command.bdist_wheel import bdist_wheel
except ImportError:  # pragma: no cover
    from wheel.bdist_wheel import bdist_wheel

ROOT = Path(__file__).parent.resolve()
PKG = ROOT / "src" / "py_excelize"
LIB_NAME = {"win32": "pyexcelize.dll", "darwin": "libpyexcelize.dylib"}.get(
    sys.platform, "libpyexcelize.so"
)


def build_go_library(dest: Path) -> Path:
    go = shutil.which("go")
    if go is None:
        raise RuntimeError("the Go toolchain (https://go.dev/dl/) is required to build py-excelize")
    out = dest / LIB_NAME
    env = dict(os.environ, CGO_ENABLED="1")
    subprocess.check_call(
        [go, "build", "-trimpath", "-ldflags=-s -w", "-buildmode=c-shared",
         "-o", str(out), "./cmd/libpyexcelize"],
        cwd=ROOT,
        env=env,
    )
    header = out.with_suffix(".h")
    if header.exists():
        header.unlink()
    return out


class BuildPy(build_py):
    def run(self):
        if not os.environ.get("PY_EXCELIZE_SKIP_BUILD"):
            build_go_library(PKG)
        super().run()


class BinaryDistribution(Distribution):
    """Make setuptools treat the package as platform specific (platlib)."""

    def has_ext_modules(self):
        return True


class BdistWheel(bdist_wheel):
    def finalize_options(self):
        super().finalize_options()
        self.root_is_pure = False

    def get_tag(self):
        _, _, plat = super().get_tag()
        return "py3", "none", plat


setup(
    distclass=BinaryDistribution,
    cmdclass={"build_py": BuildPy, "bdist_wheel": BdistWheel},
)

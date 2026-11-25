"""Thin ctypes binding to the Go shared library.

All communication uses JSON strings; this module hides pointers, freeing and
error translation so the rest of the package only sees Python objects.
"""

from __future__ import annotations

import ctypes
import json
import os
import sys
from typing import Any, Optional

from .errors import ExcelizeError, error_from_kind

__all__ = ["library_path", "Library", "get_library"]

_NAMES = {
    "win32": "pyexcelize.dll",
    "darwin": "libpyexcelize.dylib",
}


def library_path(platform: Optional[str] = None, directory: Optional[str] = None) -> str:
    """Return the expected location of the shared library for ``platform``."""
    platform = sys.platform if platform is None else platform
    name = _NAMES.get(platform, "libpyexcelize.so")
    directory = os.path.dirname(os.path.abspath(__file__)) if directory is None else directory
    return os.path.join(directory, name)


class Library:
    """A loaded libpyexcelize with typed entry points."""

    def __init__(self, path: Optional[str] = None) -> None:
        path = path or os.environ.get("PY_EXCELIZE_LIB") or library_path()
        if not os.path.exists(path):
            raise ExcelizeError(
                f"native library not found at {path!r}; "
                "reinstall py-excelize or build it with `go build -buildmode=c-shared`"
            )
        self.path = path
        self._cdll = ctypes.CDLL(path)
        c = self._cdll
        for fn in (c.PexVersion, c.PexOpen, c.PexOpenBytes, c.PexClose, c.PexCall):
            fn.restype = ctypes.c_void_p
        c.PexOpen.argtypes = [ctypes.c_char_p, ctypes.c_char_p]
        c.PexOpenBytes.argtypes = [ctypes.c_char_p, ctypes.c_size_t, ctypes.c_char_p]
        c.PexClose.argtypes = [ctypes.c_longlong]
        c.PexCall.argtypes = [ctypes.c_longlong, ctypes.c_char_p, ctypes.c_char_p]
        c.PexFree.argtypes = [ctypes.c_void_p]
        c.PexFree.restype = None

    def _take(self, ptr: Optional[int]) -> str:
        if not ptr:
            raise ExcelizeError("native library returned a null pointer")
        try:
            return ctypes.string_at(ptr).decode("utf-8")
        finally:
            self._cdll.PexFree(ptr)

    def _unwrap(self, raw: str) -> Any:
        resp = json.loads(raw)
        if not resp["ok"]:
            err = resp["error"]
            raise error_from_kind(err["kind"], err["message"])
        return resp.get("result")

    @property
    def version(self) -> str:
        return self._take(self._cdll.PexVersion())

    def open(self, path: str, password: Optional[str] = None) -> int:
        raw = self._take(self._cdll.PexOpen(os.fsencode(path), _enc(password)))
        return self._unwrap(raw)["handle"]

    def open_bytes(self, data: bytes, password: Optional[str] = None) -> int:
        raw = self._take(self._cdll.PexOpenBytes(data, len(data), _enc(password)))
        return self._unwrap(raw)["handle"]

    def close(self, handle: int) -> None:
        self._unwrap(self._take(self._cdll.PexClose(handle)))

    def call(self, handle: int, method: str, **args: Any) -> Any:
        payload = json.dumps(args).encode("utf-8") if args else b""
        raw = self._take(self._cdll.PexCall(handle, method.encode("ascii"), payload))
        return self._unwrap(raw)


def _enc(value: Optional[str]) -> bytes:
    return value.encode("utf-8") if value else b""


_instance: Optional[Library] = None


def get_library() -> Library:
    """Return the process-wide :class:`Library`, loading it on first use."""
    global _instance
    if _instance is None:
        _instance = Library()
    return _instance

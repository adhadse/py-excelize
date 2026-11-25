import ctypes
import os
import sys

import pytest

import py_excelize as pex
from py_excelize import _lib
from py_excelize._lib import Library, get_library, library_path


@pytest.mark.parametrize(
    "platform, name",
    [
        ("linux", "libpyexcelize.so"),
        ("linux2", "libpyexcelize.so"),
        ("freebsd13", "libpyexcelize.so"),
        ("darwin", "libpyexcelize.dylib"),
        ("win32", "pyexcelize.dll"),
    ],
)
def test_library_path_per_platform(platform, name):
    assert os.path.basename(library_path(platform=platform)) == name


def test_library_path_custom_directory():
    p = library_path(platform="linux", directory="/x/y")
    assert p == os.path.join("/x/y", "libpyexcelize.so")


def test_library_path_default_directory_is_package_dir():
    assert os.path.dirname(library_path()) == os.path.dirname(os.path.abspath(pex.__file__))


def test_get_library_is_singleton():
    assert get_library() is get_library()


def test_version_matches_python_package():
    assert get_library().version == pex.__version__


def test_missing_library_raises_helpful_error(tmp_path):
    with pytest.raises(pex.ExcelizeError, match="native library not found"):
        Library(str(tmp_path / "nope.so"))


def test_env_override_is_respected(monkeypatch, tmp_path):
    monkeypatch.setenv("PY_EXCELIZE_LIB", str(tmp_path / "missing"))
    with pytest.raises(pex.ExcelizeError, match="missing"):
        Library()


def test_env_override_can_point_to_real_library(monkeypatch):
    monkeypatch.setenv("PY_EXCELIZE_LIB", get_library().path)
    assert Library().version == pex.__version__


def test_library_remembers_path():
    assert os.path.exists(get_library().path)


def test_open_returns_positive_handle(cells_path):
    lib = get_library()
    h = lib.open(str(cells_path))
    try:
        assert isinstance(h, int) and h > 0
    finally:
        lib.close(h)


def test_handles_are_unique(cells_path):
    lib = get_library()
    hs = [lib.open(str(cells_path)) for _ in range(5)]
    try:
        assert len(set(hs)) == 5
    finally:
        for h in hs:
            lib.close(h)


def test_open_bytes(cells_path):
    lib = get_library()
    h = lib.open_bytes(cells_path.read_bytes())
    try:
        assert lib.call(h, "sheet_names") == ["Sheet1", "Sheet 2", "Ünï Çødé"]
    finally:
        lib.close(h)


def test_close_unknown_handle_is_ok():
    get_library().close(10**9)


def test_double_close_is_ok(cells_path):
    lib = get_library()
    h = lib.open(str(cells_path))
    lib.close(h)
    lib.close(h)


def test_call_after_close_raises_closed(cells_path):
    lib = get_library()
    h = lib.open(str(cells_path))
    lib.close(h)
    with pytest.raises(pex.WorkbookClosedError):
        lib.call(h, "sheet_names")


def test_call_unknown_handle():
    with pytest.raises(pex.WorkbookClosedError):
        get_library().call(424242, "sheet_names")


def test_unknown_method_raises_base_error(cells_path):
    lib = get_library()
    h = lib.open(str(cells_path))
    try:
        with pytest.raises(pex.ExcelizeError, match="unknown method"):
            lib.call(h, "explode")
    finally:
        lib.close(h)


@pytest.mark.parametrize(
    "path, exc",
    [
        ("/definitely/not/here.xlsx", pex.WorkbookNotFoundError),
    ],
)
def test_open_errors(path, exc):
    with pytest.raises(exc):
        get_library().open(path)


def test_open_bytes_garbage_is_invalid():
    with pytest.raises(pex.InvalidWorkbookError):
        get_library().open_bytes(b"not a workbook")


def test_open_bytes_empty_is_invalid():
    with pytest.raises(pex.InvalidWorkbookError):
        get_library().open_bytes(b"")


def test_password_required(encrypted_path):
    with pytest.raises(pex.PasswordError):
        get_library().open(str(encrypted_path))


def test_password_wrong(encrypted_path):
    with pytest.raises(pex.PasswordError):
        get_library().open(str(encrypted_path), "wrong")


def test_password_correct(encrypted_path):
    lib = get_library()
    h = lib.open(str(encrypted_path), "secret")
    lib.close(h)


def test_unicode_password_roundtrip_does_not_crash(encrypted_path):
    with pytest.raises(pex.PasswordError):
        get_library().open(str(encrypted_path), "pässwörd✓")


def test_take_raises_on_null_pointer():
    with pytest.raises(pex.ExcelizeError, match="null pointer"):
        get_library()._take(None)


def test_take_frees_pointer():
    lib = get_library()
    ptr = lib._cdll.PexVersion()
    assert lib._take(ptr) == pex.__version__


def test_unwrap_error_payload():
    lib = get_library()
    with pytest.raises(pex.SheetNotFoundError, match="gone"):
        lib._unwrap('{"ok":false,"error":{"kind":"sheet_not_found","message":"gone"}}')


def test_unwrap_ok_without_result_is_none():
    assert get_library()._unwrap('{"ok":true}') is None


def test_enc_helper():
    assert _lib._enc(None) == b""
    assert _lib._enc("") == b""
    assert _lib._enc("é") == "é".encode()


def test_arguments_with_special_characters_are_json_safe(cells_path):
    lib = get_library()
    h = lib.open(str(cells_path))
    try:
        with pytest.raises(pex.SheetNotFoundError):
            lib.call(h, "rows", sheet='quote" back\\slash \n newline')
    finally:
        lib.close(h)


def test_path_with_unicode(tmp_path, cells_path):
    dst = tmp_path / "dätä ✓.xlsx"
    dst.write_bytes(cells_path.read_bytes())
    with pex.open(dst) as wb:
        assert "Sheet1" in wb


@pytest.mark.skipif(sys.platform == "win32", reason="posix only")
def test_library_is_loadable_by_ctypes_directly():
    assert ctypes.CDLL(get_library().path) is not None

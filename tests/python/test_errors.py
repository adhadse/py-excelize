import pytest

import py_excelize as pex
from py_excelize.errors import ExcelizeError, error_from_kind

ALL = [
    pex.WorkbookNotFoundError,
    pex.InvalidWorkbookError,
    pex.PasswordError,
    pex.WorkbookClosedError,
    pex.SheetNotFoundError,
    pex.InvalidCellError,
]

KINDS = [
    ("closed", pex.WorkbookClosedError, ValueError),
    ("sheet_not_found", pex.SheetNotFoundError, KeyError),
    ("invalid_cell", pex.InvalidCellError, ValueError),
    ("file_not_found", pex.WorkbookNotFoundError, FileNotFoundError),
    ("format", pex.InvalidWorkbookError, ValueError),
    ("password", pex.PasswordError, ValueError),
]


@pytest.mark.parametrize("cls", ALL)
def test_every_error_derives_from_base(cls):
    assert issubclass(cls, ExcelizeError)
    assert issubclass(cls, Exception)


@pytest.mark.parametrize("cls", ALL)
def test_errors_are_catchable_via_base(cls):
    with pytest.raises(pex.ExcelizeError):
        raise cls("boom")


@pytest.mark.parametrize("kind, cls, builtin", KINDS)
def test_kind_maps_to_class_and_builtin(kind, cls, builtin):
    err = error_from_kind(kind, "message")
    assert type(err) is cls
    assert isinstance(err, builtin)
    assert isinstance(err, ExcelizeError)


@pytest.mark.parametrize("kind", ["", "error", "does-not-exist", "CLOSED"])
def test_unknown_kind_falls_back_to_base(kind):
    err = error_from_kind(kind, "m")
    assert type(err) is ExcelizeError


@pytest.mark.parametrize("kind, cls, builtin", KINDS)
def test_message_is_preserved(kind, cls, builtin):
    assert "hello world" in str(error_from_kind(kind, "hello world"))


def test_sheet_not_found_str_is_not_quoted():
    assert str(pex.SheetNotFoundError("sheet 'x' does not exist")) == "sheet 'x' does not exist"


def test_sheet_not_found_still_a_keyerror():
    with pytest.raises(KeyError):
        raise pex.SheetNotFoundError("x")


def test_workbook_not_found_is_oserror():
    assert issubclass(pex.WorkbookNotFoundError, OSError)


def test_base_error_has_no_builtin_parent_besides_exception():
    assert ExcelizeError.__bases__ == (Exception,)


@pytest.mark.parametrize("cls", ALL)
def test_errors_are_picklable_by_args(cls):
    import copy

    err = cls("msg")
    assert str(copy.copy(err)) == str(err)


def test_public_exports_present():
    for cls in ALL + [pex.ExcelizeError]:
        assert cls.__name__ in pex.__all__

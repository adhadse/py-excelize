import gc

import pytest

import py_excelize as pex
from py_excelize import Workbook, Worksheet

from conftest import EXPECTED_CONTROLS


def test_sheetnames(cells_wb):
    assert cells_wb.sheetnames == ["Sheet1", "Sheet 2", "Ünï Çødé"]


def test_sheetnames_returns_a_copy(cells_wb):
    names = cells_wb.sheetnames
    names.append("junk")
    assert "junk" not in cells_wb.sheetnames


def test_len(cells_wb, controls_wb, empty_wb):
    assert (len(cells_wb), len(controls_wb), len(empty_wb)) == (3, 2, 1)


def test_iter_yields_worksheets_in_order(cells_wb):
    sheets = list(cells_wb)
    assert all(isinstance(s, Worksheet) for s in sheets)
    assert [s.title for s in sheets] == cells_wb.sheetnames


def test_worksheets_property(cells_wb):
    assert [w.title for w in cells_wb.worksheets] == cells_wb.sheetnames


@pytest.mark.parametrize("name", ["Sheet1", "Sheet 2", "Ünï Çødé"])
def test_getitem_by_name(cells_wb, name):
    ws = cells_wb[name]
    assert ws.title == name
    assert ws.workbook is cells_wb


@pytest.mark.parametrize("name", ["sheet1", "SHEET1", "sheet 2", "ÜNÏ ÇØDÉ"])
def test_getitem_case_insensitive_like_excel(cells_wb, name):
    assert cells_wb[name].title == name


@pytest.mark.parametrize("index, title", [(0, "Sheet1"), (1, "Sheet 2"), (2, "Ünï Çødé"), (-1, "Ünï Çødé"), (-3, "Sheet1")])
def test_getitem_by_index(cells_wb, index, title):
    assert cells_wb[index].title == title


@pytest.mark.parametrize("index", [3, 99, -4])
def test_getitem_index_out_of_range(cells_wb, index):
    with pytest.raises(IndexError):
        cells_wb[index]


@pytest.mark.parametrize("name", ["", "Nope", "Sheet", "Sheet1 ", " Sheet1", "Sheet1\n"])
def test_getitem_missing_sheet(cells_wb, name):
    with pytest.raises(pex.SheetNotFoundError):
        cells_wb[name]


def test_missing_sheet_is_keyerror(cells_wb):
    with pytest.raises(KeyError):
        cells_wb["x"]


def test_missing_sheet_message_names_the_sheet(cells_wb):
    with pytest.raises(pex.SheetNotFoundError, match="'Nope'"):
        cells_wb["Nope"]


@pytest.mark.parametrize("key", [None, 1.5, b"Sheet1", ("Sheet1",), True, False])
def test_getitem_rejects_bad_key_types(cells_wb, key):
    with pytest.raises(TypeError):
        cells_wb[key]


@pytest.mark.parametrize("name, expected", [("Sheet1", True), ("sheet1", True), ("Nope", False), ("", False)])
def test_contains(cells_wb, name, expected):
    assert (name in cells_wb) is expected


@pytest.mark.parametrize("value", [None, 1, b"Sheet1", ["Sheet1"]])
def test_contains_non_str_is_false(cells_wb, value):
    assert value not in cells_wb


def test_get_default(cells_wb):
    assert cells_wb.get("Nope") is None
    assert cells_wb.get("Nope", cells_wb[0]) == cells_wb[0]


def test_get_existing(cells_wb):
    assert cells_wb.get("Sheet1").title == "Sheet1"


def test_controls_all_sheets(controls_wb):
    got = [(c.cell, c.type.value, c.checked, c.linked_cell, c.text) for c in controls_wb.controls]
    assert got == EXPECTED_CONTROLS


def test_controls_empty_when_no_controls(cells_wb, empty_wb):
    assert cells_wb.controls == []
    assert empty_wb.controls == []


def test_controls_carry_sheet_name(controls_wb):
    assert {c.sheet for c in controls_wb.controls} == {"Sheet 1"}


def test_to_dict(controls_wb):
    d = controls_wb.to_dict()
    assert [s["title"] for s in d["sheets"]] == ["Sheet 1", "Plain"]
    assert len(d["sheets"][0]["controls"]) == len(EXPECTED_CONTROLS)
    assert d["sheets"][1]["controls"] == []


def test_to_dict_is_json_serialisable(controls_wb):
    import json

    assert json.loads(json.dumps(controls_wb.to_dict())) == controls_wb.to_dict()


def test_repr_open(cells_wb):
    assert repr(cells_wb) == "<Workbook sheets=['Sheet1', 'Sheet 2', 'Ünï Çødé']>"


def test_repr_closed(cells_path):
    wb = pex.open(cells_path)
    wb.close()
    assert repr(wb) == "<Workbook closed>"


def test_closed_flag(cells_path):
    wb = pex.open(cells_path)
    assert wb.closed is False
    wb.close()
    assert wb.closed is True


def test_close_is_idempotent(cells_path):
    wb = pex.open(cells_path)
    wb.close()
    wb.close()
    assert wb.closed


def test_context_manager_closes(cells_path):
    with pex.open(cells_path) as wb:
        assert not wb.closed
    assert wb.closed


def test_context_manager_closes_on_exception(cells_path):
    with pytest.raises(RuntimeError):
        with pex.open(cells_path) as wb:
            raise RuntimeError("x")
    assert wb.closed


def test_context_manager_returns_self(cells_path):
    wb = pex.open(cells_path)
    with wb as entered:
        assert entered is wb


@pytest.mark.parametrize(
    "use",
    [
        lambda wb: wb.sheetnames,
        lambda wb: wb.worksheets,
        lambda wb: len(wb),
        lambda wb: list(wb),
        lambda wb: "Sheet1" in wb,
        lambda wb: wb["Sheet1"],
        lambda wb: wb[0],
        lambda wb: wb.get("Sheet1"),
        lambda wb: wb.controls,
        lambda wb: wb.to_dict(),
    ],
    ids=["sheetnames", "worksheets", "len", "iter", "contains", "getitem_str", "getitem_int", "get", "controls", "to_dict"],
)
def test_every_operation_after_close_raises(cells_path, use):
    wb = pex.open(cells_path)
    wb.close()
    with pytest.raises(pex.WorkbookClosedError):
        use(wb)


def test_closed_error_is_valueerror(cells_path):
    wb = pex.open(cells_path)
    wb.close()
    with pytest.raises(ValueError):
        wb.sheetnames


def test_worksheet_use_after_workbook_close(cells_path):
    wb = pex.open(cells_path)
    ws = wb["Sheet1"]
    wb.close()
    with pytest.raises(pex.WorkbookClosedError):
        ws.rows
    with pytest.raises(pex.WorkbookClosedError):
        ws["A1"]


def test_del_closes_handle(cells_path):
    from py_excelize._lib import get_library

    wb = pex.open(cells_path)
    handle = wb._handle
    del wb
    gc.collect()
    with pytest.raises(pex.WorkbookClosedError):
        get_library().call(handle, "sheet_names")


def test_del_swallows_errors():
    class Boom:
        def close(self, h):
            raise RuntimeError("library gone")

    wb = Workbook(1, lib=Boom())  # type: ignore[arg-type]
    wb.__del__()
    wb._handle = None


def test_workbook_is_not_hashable_by_value_but_usable_in_sets(cells_wb):
    assert cells_wb in {cells_wb}


def test_two_workbooks_do_not_interfere(cells_path, controls_path):
    with pex.open(cells_path) as a, pex.open(controls_path) as b:
        assert a.sheetnames != b.sheetnames
        assert a.controls == []
        assert len(b.controls) == len(EXPECTED_CONTROLS)


def test_closing_one_workbook_leaves_the_other_usable(cells_path):
    a, b = pex.open(cells_path), pex.open(cells_path)
    a.close()
    assert b.sheetnames
    b.close()


def test_empty_workbook(empty_wb):
    assert empty_wb.sheetnames == ["Sheet1"]
    assert empty_wb["Sheet1"].rows == []
    assert empty_wb["Sheet1"].dimensions == ""

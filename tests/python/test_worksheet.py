import pytest

import py_excelize as pex
from py_excelize import ControlType, MergedRange

from conftest import EXPECTED_CONTROLS

ROWS = [
    ["name", "qty", "price"],
    ["apple", "3", "1.5", "", "", "TRUE"],
    ["pear", "10", "0.25"],
    [],
    ["gap row"],
    [],
    ["merged title"],
]


@pytest.fixture
def ws(cells_wb):
    return cells_wb["Sheet1"]


@pytest.fixture
def cws(controls_wb):
    return controls_wb["Sheet 1"]


def test_title_and_repr(ws):
    assert ws.title == "Sheet1"
    assert repr(ws) == "<Worksheet 'Sheet1'>"


def test_equality_and_hash(cells_wb, controls_wb):
    assert cells_wb["Sheet1"] == cells_wb["Sheet1"]
    assert hash(cells_wb["Sheet1"]) == hash(cells_wb["Sheet1"])
    assert cells_wb["Sheet1"] != cells_wb["Sheet 2"]
    assert cells_wb["Sheet1"] != controls_wb["Sheet 1"]
    assert cells_wb["Sheet1"] != "Sheet1"


def test_usable_as_dict_key(cells_wb):
    assert {cells_wb["Sheet1"]: 1}[cells_wb["Sheet1"]] == 1


@pytest.mark.parametrize(
    "ref, expected",
    [
        ("A1", "name"),
        ("B1", "qty"),
        ("A2", "apple"),
        ("B2", "3"),
        ("C2", "1.5"),
        ("C3", "0.25"),
        ("F2", "TRUE"),
        ("A5", "gap row"),
        ("A7", "merged title"),
        ("a1", "name"),
        ("Z99", ""),
        ("D4", ""),
        ("XFD1", ""),
    ],
)
def test_cell_values(ws, ref, expected):
    assert ws.cell(ref) == expected
    assert ws[ref] == expected


@pytest.mark.parametrize(
    "ref",
    ["", "1", "A", "A0", "0A", "AAAA1", "A-1", " A1", "A1 ", "A1:B2", "$A$1", "A 1", "Ä1", "A1048577x", "A1.5"],
)
def test_invalid_cell_refs(ws, ref):
    with pytest.raises(pex.InvalidCellError):
        ws.cell(ref)
    with pytest.raises(pex.InvalidCellError):
        ws.formula(ref)


def test_invalid_cell_is_valueerror(ws):
    with pytest.raises(ValueError):
        ws["!!"]


@pytest.mark.parametrize("ref, expected", [("D2", "B2*C2"), ("A2", ""), ("Z99", ""), ("d2", "B2*C2")])
def test_formula(ws, ref, expected):
    assert ws.formula(ref) == expected


def test_rows(ws):
    assert ws.rows == ROWS


def test_rows_returns_fresh_lists(ws):
    ws.rows[0].append("junk")
    assert ws.rows == ROWS


def test_iter_rows(ws):
    assert list(ws.iter_rows()) == ROWS


def test_iter_rows_is_lazy_iterator(ws):
    it = ws.iter_rows()
    assert next(it) == ROWS[0]
    assert next(it) == ROWS[1]


@pytest.mark.parametrize(
    "sheet, dim",
    [("Sheet1", "A1:F7"), ("Sheet 2", "B2"), ("Ünï Çødé", "A1")],
)
def test_dimensions(cells_wb, sheet, dim):
    assert cells_wb[sheet].dimensions == dim


def test_dimensions_of_sheet_without_data(controls_wb):
    assert controls_wb["Plain"].dimensions == ""


def test_unicode_sheet_content(cells_wb):
    assert cells_wb["Ünï Çødé"]["A1"] == "héllo wörld ✓"


def test_sheet_with_space(cells_wb):
    assert cells_wb["Sheet 2"]["B2"] == "spaced"
    assert cells_wb["Sheet 2"].rows == [[], ["", "spaced"]]


def test_merged_cells(ws):
    merged = ws.merged_cells
    assert merged == [
        MergedRange("A7:C7", "A7", "C7", "merged title"),
        MergedRange("E1:E3", "E1", "E3", ""),
    ]
    assert [str(m) for m in merged] == ["A7:C7", "E1:E3"]


def test_merged_cells_empty(cells_wb):
    assert cells_wb["Sheet 2"].merged_cells == []


def test_records_default_header(ws):
    recs = ws.to_records()
    assert recs[0] == {"name": "apple", "qty": "3", "price": "1.5"}
    assert recs[1] == {"name": "pear", "qty": "10", "price": "0.25"}


def test_records_pad_short_rows(ws):
    assert ws.to_records()[3] == {"name": "gap row", "qty": "", "price": ""}


def test_records_length(ws):
    assert len(ws.to_records()) == len(ROWS) - 1


def test_records_empty_row_gives_blank_dict_values(ws):
    assert ws.to_records()[2] == {"name": "", "qty": "", "price": ""}


def test_records_other_header_row(ws):
    recs = ws.to_records(header_row=2)
    assert recs[0] == {"apple": "pear", "3": "10", "1.5": "0.25", "TRUE": ""}


def test_records_header_row_beyond_data(ws):
    assert ws.to_records(header_row=100) == []


def test_records_on_empty_sheet(empty_wb):
    assert empty_wb["Sheet1"].to_records() == []


@pytest.mark.parametrize("bad", [0, -1])
def test_records_invalid_header_row(ws, bad):
    with pytest.raises(ValueError):
        ws.to_records(header_row=bad)


def test_records_skip_blank_headers(cells_wb):
    ws = cells_wb["Sheet 2"]
    assert ws.to_records(header_row=2) == []
    assert ws.to_records(header_row=1) == [{}]


def test_controls_count(cws):
    assert len(cws.controls) == len(EXPECTED_CONTROLS)


@pytest.mark.parametrize("index, expected", list(enumerate(EXPECTED_CONTROLS)))
def test_each_control(cws, index, expected):
    cell, kind, checked, linked, text = expected
    c = cws.controls[index]
    assert c.sheet == "Sheet 1"
    assert c.cell == cell
    assert c.type is ControlType(kind)
    assert c.checked is checked
    assert c.linked_cell == linked
    assert c.text == text


def test_spin_button_values(cws):
    c = cws.controls_of(ControlType.SPIN_BUTTON)[0]
    assert (c.current_value, c.min_value, c.max_value, c.inc_change) == (7, 1, 20, 2)
    assert c.horizontal is False


def test_scroll_bar_values(cws):
    c = cws.controls_of(ControlType.SCROLL_BAR)[0]
    assert (c.current_value, c.min_value, c.max_value) == (30, 0, 100)
    assert (c.inc_change, c.page_change) == (1, 10)
    assert c.horizontal is True


def test_button_macro(cws):
    assert cws.controls_of(ControlType.BUTTON)[0].macro == "Button1_Click"


def test_multi_run_paragraph(cws):
    c = cws.controls_of("check_box")[-1]
    assert c.paragraphs == ["Line one", "Line two"]
    assert c.text == "Line one Line two"


@pytest.mark.parametrize(
    "kind, count",
    [
        (ControlType.CHECK_BOX, 3),
        (ControlType.OPTION_BUTTON, 2),
        (ControlType.SPIN_BUTTON, 1),
        (ControlType.SCROLL_BAR, 1),
        (ControlType.GROUP_BOX, 1),
        (ControlType.LABEL, 1),
        (ControlType.BUTTON, 1),
        (ControlType.NOTE, 0),
        (ControlType.UNKNOWN, 0),
    ],
)
def test_controls_of(cws, kind, count):
    assert len(cws.controls_of(kind)) == count
    assert len(cws.controls_of(kind.value)) == count


def test_controls_of_unrecognised_string_means_unknown(cws):
    assert cws.controls_of("slider") == []


def test_checked_checkboxes(cws):
    checked = [c.cell for c in cws.controls_of(ControlType.CHECK_BOX) if c.checked]
    assert checked == ["A1"]


def test_option_group_selected(cws):
    selected = [c.text for c in cws.controls_of(ControlType.OPTION_BUTTON) if c.checked]
    assert selected == ["Option A"]


def test_sheet_without_controls(controls_wb):
    assert controls_wb["Plain"].controls == []
    assert controls_wb["Plain"].controls_of(ControlType.CHECK_BOX) == []


def test_controls_are_hashable_values(cws):
    c = cws.controls[0]
    assert c == cws.controls[0]
    assert c != cws.controls[1]

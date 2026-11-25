"""Cross-product checks of every sheet / cell reference combination."""

import pytest

import py_excelize as pex

SHEETS = ["Sheet1", "Sheet 2", "Ünï Çødé"]
CELLS = {
    "Sheet1": {"A1": "name", "A2": "apple", "B3": "10", "F2": "TRUE", "C1": "price", "Q9": ""},
    "Sheet 2": {"A1": "", "B2": "spaced", "C3": ""},
    "Ünï Çødé": {"A1": "héllo wörld ✓", "B1": "", "A2": ""},
}
CASES = [(s, r, v) for s, cells in CELLS.items() for r, v in cells.items()]
VARIANTS = [str, str.lower]


@pytest.mark.parametrize("sheet, ref, value", CASES)
@pytest.mark.parametrize("variant", VARIANTS, ids=["exact", "lower"])
def test_cell_matrix(cells_wb, sheet, ref, value, variant):
    assert cells_wb[sheet][variant(ref)] == value


@pytest.mark.parametrize("sheet, ref, value", CASES)
def test_cell_matrix_matches_rows(cells_wb, sheet, ref, value):
    rows = cells_wb[sheet].rows
    import re

    m = re.match(r"([A-Z]+)(\d+)", ref)
    col = sum((ord(ch) - 64) * 26**i for i, ch in enumerate(reversed(m.group(1)))) - 1
    row = int(m.group(2)) - 1
    got = rows[row][col] if row < len(rows) and col < len(rows[row]) else ""
    assert got == value


@pytest.mark.parametrize("sheet", SHEETS)
@pytest.mark.parametrize("path_kind", ["path", "bytes", "encrypted"])
def test_sources_agree(cells_path, encrypted_path, sheet, path_kind):
    if path_kind == "path":
        wb = pex.open(cells_path)
    elif path_kind == "bytes":
        wb = pex.load(cells_path.read_bytes())
    else:
        wb = pex.open(encrypted_path, password="secret")
    with wb:
        assert wb[sheet].rows == pex.open(cells_path)[sheet].rows


@pytest.mark.parametrize("sheet", SHEETS)
@pytest.mark.parametrize("ref", ["", "A0", "1", "AAAA1", "A1:B2", "$A$1", "ZZZ1"])
def test_invalid_refs_on_every_sheet(cells_wb, sheet, ref):
    with pytest.raises(pex.InvalidCellError):
        cells_wb[sheet][ref]


@pytest.mark.parametrize("sheet", SHEETS)
def test_every_sheet_has_no_controls_and_no_error(cells_wb, sheet):
    assert cells_wb[sheet].controls == []
    assert cells_wb[sheet].controls_of(pex.ControlType.CHECK_BOX) == []


@pytest.mark.parametrize("sheet", SHEETS)
def test_every_sheet_dimensions_and_rows_consistent(cells_wb, sheet):
    ws = cells_wb[sheet]
    assert bool(ws.dimensions) == any(any(c for c in r) for r in ws.rows)


@pytest.mark.parametrize("index", range(-3, 3))
def test_index_and_name_lookup_agree(cells_wb, index):
    assert cells_wb[index] == cells_wb[cells_wb.sheetnames[index]]


@pytest.mark.parametrize("header_row", [1, 2, 3, 5, 7, 8])
def test_records_row_counts(cells_wb, header_row):
    ws = cells_wb["Sheet1"]
    assert len(ws.to_records(header_row)) == max(len(ws.rows) - header_row, 0)


@pytest.mark.parametrize("kind", list(pex.ControlType))
def test_controls_of_partitions_all_controls(controls_wb, kind):
    ws = controls_wb["Sheet 1"]
    total = sum(len(ws.controls_of(k)) for k in pex.ControlType)
    assert total == len(ws.controls)
    assert all(c.type is kind for c in ws.controls_of(kind))

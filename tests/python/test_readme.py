"""Keep the README honest: every claim about the public API is exercised here."""

import builtins
import re
from pathlib import Path

import pytest

import py_excelize as pex

README = (Path(__file__).resolve().parents[2] / "README.md").read_text(encoding="utf-8")


def test_all_documented_exceptions_exist():
    for name in re.findall(r"`(\w+Error)`", README):
        assert hasattr(pex, name) or hasattr(builtins, name), name


def test_documented_exceptions_have_documented_builtin_bases():
    rows = re.findall(r"\| `(\w+Error)` \| `(\w+)` \|", README)
    assert len(rows) == 6
    for exc, builtin in rows:
        assert issubclass(getattr(pex, exc), getattr(builtins, builtin))


def test_documented_control_types_exist():
    line = next(l for l in README.splitlines() if l.startswith("| `ControlType`"))
    for name in re.findall(r"`([A-Z_]+)`", line):
        assert hasattr(pex.ControlType, name), name


def test_documented_workbook_members_exist(cells_wb):
    line = next(l for l in README.splitlines() if l.startswith("| `Workbook`"))
    for member in ("sheetnames", "worksheets", "get", "controls", "to_dict", "close", "closed"):
        assert f"`{member}" in line
        assert hasattr(cells_wb, member)


def test_documented_worksheet_members_exist(cells_wb):
    ws = cells_wb["Sheet1"]
    for member in ("title", "dimensions", "cell", "formula", "rows", "iter_rows", "to_records", "merged_cells", "controls", "controls_of"):
        assert hasattr(ws, member)


def test_documented_form_control_fields_exist():
    import dataclasses

    fields = {f.name for f in dataclasses.fields(pex.FormControl)}
    line = next(l for l in README.splitlines() if l.startswith("| `FormControl`"))
    for name in re.findall(r"\b([a-z]+(?:_[a-z]+)*)\b", line.split("dataclass:")[1].split("`")[1]):
        assert name in fields, name


def test_quick_start_example():
    path = Path(__file__).resolve().parents[2] / "testdata" / "controls.xlsx"
    with pex.open(path) as wb:
        assert wb.sheetnames == ["Sheet 1", "Plain"]
        ws = wb["Sheet 1"]
        ticked = [c.text for c in ws.controls_of(pex.ControlType.CHECK_BOX) if c.checked]
        assert ticked == ["Agree"]


def test_cells_example(cells_wb):
    ws = cells_wb["Sheet1"]
    assert ws["A2"] == "apple"
    assert ws.formula("D2") == "B2*C2"
    assert ws.dimensions == "A1:F7"
    assert ws.rows[0] == ["name", "qty", "price"]
    assert ws.to_records()[0]["name"] == "apple"
    assert ws.merged_cells[0].range == "A7:C7"


def test_inmemory_and_password_examples(cells_path, encrypted_path):
    assert pex.load(cells_path.read_bytes()).sheetnames
    assert pex.open(encrypted_path, password="secret").sheetnames


@pytest.mark.parametrize(
    "call, exc",
    [
        (lambda wb: wb["nope"], pex.SheetNotFoundError),
        (lambda wb: wb["Sheet1"]["A0"], pex.InvalidCellError),
        (lambda wb: wb["Sheet1"]["1A"], pex.InvalidCellError),
    ],
)
def test_error_examples(cells_wb, call, exc):
    with pytest.raises(exc):
        call(cells_wb)


def test_version_documented_consistently():
    from py_excelize._lib import get_library

    assert get_library().version == pex.__version__

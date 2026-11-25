import io
import pathlib

import pytest

import py_excelize as pex


def test_open_str_path(cells_path):
    with pex.open(str(cells_path)) as wb:
        assert wb.sheetnames[0] == "Sheet1"


def test_open_pathlib(cells_path):
    with pex.open(cells_path) as wb:
        assert isinstance(cells_path, pathlib.Path)
        assert len(wb) == 3


def test_open_relative_path(cells_path, monkeypatch):
    monkeypatch.chdir(cells_path.parent)
    with pex.open("cells.xlsx") as wb:
        assert "Sheet 2" in wb


def test_open_missing_file(tmp_path):
    with pytest.raises(pex.WorkbookNotFoundError):
        pex.open(tmp_path / "missing.xlsx")


def test_open_missing_file_is_filenotfounderror(tmp_path):
    with pytest.raises(FileNotFoundError):
        pex.open(tmp_path / "missing.xlsx")


def test_open_directory_fails(tmp_path):
    with pytest.raises(pex.ExcelizeError):
        pex.open(tmp_path)


def test_open_text_file(tmp_path):
    p = tmp_path / "a.xlsx"
    p.write_text("hello")
    with pytest.raises(pex.InvalidWorkbookError):
        pex.open(p)


def test_open_empty_file(tmp_path):
    p = tmp_path / "a.xlsx"
    p.write_bytes(b"")
    with pytest.raises(pex.InvalidWorkbookError):
        pex.open(p)


def test_open_truncated_file(tmp_path, cells_path):
    p = tmp_path / "t.xlsx"
    p.write_bytes(cells_path.read_bytes()[:200])
    with pytest.raises(pex.InvalidWorkbookError):
        pex.open(p)


def test_open_rejects_non_path():
    with pytest.raises(TypeError):
        pex.open(123)  # type: ignore[arg-type]


def test_open_password_required(encrypted_path):
    with pytest.raises(pex.PasswordError, match="password"):
        pex.open(encrypted_path)


def test_open_wrong_password(encrypted_path):
    with pytest.raises(pex.PasswordError):
        pex.open(encrypted_path, password="nope")


def test_open_right_password(encrypted_path):
    with pex.open(encrypted_path, password="secret") as wb:
        assert wb.sheetnames == ["Sheet1", "Sheet 2", "Ünï Çødé"]


def test_password_on_plain_workbook_is_ignored(cells_path):
    with pex.open(cells_path, password="whatever") as wb:
        assert len(wb) == 3


def test_password_is_keyword_only(cells_path):
    with pytest.raises(TypeError):
        pex.open(cells_path, "secret")  # type: ignore[misc]


def test_load_bytes(cells_path):
    with pex.load(cells_path.read_bytes()) as wb:
        assert wb["Sheet1"]["A2"] == "apple"


def test_load_bytearray(cells_path):
    with pex.load(bytearray(cells_path.read_bytes())) as wb:
        assert len(wb) == 3


def test_load_memoryview(cells_path):
    with pex.load(memoryview(cells_path.read_bytes())) as wb:
        assert len(wb) == 3


def test_load_bytesio(cells_path):
    with pex.load(io.BytesIO(cells_path.read_bytes())) as wb:
        assert len(wb) == 3


def test_load_file_object(cells_path):
    with cells_path.open("rb") as fh, pex.load(fh) as wb:
        assert len(wb) == 3


def test_load_encrypted_bytes(encrypted_path):
    with pex.load(encrypted_path.read_bytes(), password="secret") as wb:
        assert len(wb) == 3


def test_load_encrypted_without_password(encrypted_path):
    with pytest.raises(pex.PasswordError):
        pex.load(encrypted_path.read_bytes())


@pytest.mark.parametrize("junk", [b"", b"PK", b"PK\x03\x04", b"x" * 1000, b"\x00" * 64])
def test_load_garbage(junk):
    with pytest.raises(pex.InvalidWorkbookError):
        pex.load(junk)


def test_load_same_bytes_twice_gives_independent_workbooks(cells_path):
    data = cells_path.read_bytes()
    a, b = pex.load(data), pex.load(data)
    a.close()
    assert not b.closed
    assert b.sheetnames
    b.close()


def test_open_and_load_agree(cells_path):
    with pex.open(cells_path) as a, pex.load(cells_path.read_bytes()) as b:
        assert a.sheetnames == b.sheetnames
        assert a["Sheet1"].rows == b["Sheet1"].rows


def test_open_many_workbooks_sequentially(cells_path):
    for _ in range(50):
        with pex.open(cells_path) as wb:
            assert wb["Sheet1"]["A1"] == "name"


def test_package_metadata():
    assert pex.__version__.count(".") >= 2
    assert set(pex.__all__) >= {"open", "load", "Workbook", "Worksheet", "ControlType"}

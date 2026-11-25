"""High level, Pythonic wrappers around an open workbook."""

from __future__ import annotations

import os
import threading
from typing import Any, BinaryIO, Dict, Iterator, List, Optional, Union

from ._lib import Library, get_library
from .errors import SheetNotFoundError, WorkbookClosedError
from .models import ControlType, FormControl, MergedRange

__all__ = ["Workbook", "Worksheet", "open", "load"]

PathLike = Union[str, "os.PathLike[str]"]


class Workbook:
    """An open XLSX workbook. Use :func:`open` or :func:`load` to create one.

    Workbooks are context managers and behave like a read-only mapping of
    sheet name to :class:`Worksheet`::

        with open("form.xlsx") as wb:
            for ws in wb:
                print(ws.title, len(ws.controls))
    """

    def __init__(self, handle: int, lib: Optional[Library] = None) -> None:
        self._lib = lib or get_library()
        self._handle: Optional[int] = handle
        self._lock = threading.RLock()

    # -- lifecycle -------------------------------------------------------
    @property
    def closed(self) -> bool:
        return self._handle is None

    def close(self) -> None:
        """Release the workbook. Calling it again is a no-op."""
        with self._lock:
            handle, self._handle = self._handle, None
        if handle is not None:
            self._lib.close(handle)

    def __enter__(self) -> "Workbook":
        return self

    def __exit__(self, *exc: object) -> None:
        self.close()

    def __del__(self) -> None:
        try:
            self.close()
        except Exception:  # interpreter shutdown, library gone, ...
            pass

    def __repr__(self) -> str:
        if self.closed:
            return "<Workbook closed>"
        return f"<Workbook sheets={self.sheetnames!r}>"

    def _call(self, method: str, **args: Any) -> Any:
        with self._lock:
            if self._handle is None:
                raise WorkbookClosedError("workbook is closed")
            return self._lib.call(self._handle, method, **args)

    # -- sheets ----------------------------------------------------------
    @property
    def sheetnames(self) -> List[str]:
        """Sheet names in workbook order."""
        return list(self._call("sheet_names"))

    @property
    def worksheets(self) -> List["Worksheet"]:
        return [Worksheet(self, name) for name in self.sheetnames]

    def __len__(self) -> int:
        return len(self.sheetnames)

    def __iter__(self) -> Iterator["Worksheet"]:
        return iter(self.worksheets)

    def __contains__(self, name: object) -> bool:
        return isinstance(name, str) and bool(self._call("has_sheet", sheet=name))

    def __getitem__(self, key: Union[str, int]) -> "Worksheet":
        if isinstance(key, bool) or not isinstance(key, (str, int)):
            raise TypeError(f"sheet key must be str or int, not {type(key).__name__}")
        if isinstance(key, int):
            return Worksheet(self, self.sheetnames[key])
        if key not in self:
            raise SheetNotFoundError(f"sheet {key!r} does not exist")
        return Worksheet(self, key)

    def get(self, name: str, default: Optional["Worksheet"] = None) -> Optional["Worksheet"]:
        """Return the sheet called ``name`` or ``default``."""
        return self[name] if name in self else default

    @property
    def controls(self) -> List[FormControl]:
        """Form controls of every sheet, in sheet order."""
        return [FormControl.from_dict(d) for d in self._call("all_form_controls")]

    def to_dict(self) -> Dict[str, Any]:
        """JSON-serialisable summary: ``{"sheets": [{"title", "controls"}...]}``."""
        return {
            "sheets": [
                {"title": ws.title, "controls": [c.to_dict() for c in ws.controls]}
                for ws in self
            ]
        }


class Worksheet:
    """A single sheet. Obtain it with ``workbook["Sheet1"]``."""

    def __init__(self, workbook: Workbook, title: str) -> None:
        self.workbook = workbook
        self.title = title

    def __repr__(self) -> str:
        return f"<Worksheet {self.title!r}>"

    def __eq__(self, other: object) -> bool:
        return (
            isinstance(other, Worksheet)
            and other.workbook is self.workbook
            and other.title == self.title
        )

    def __hash__(self) -> int:
        return hash((id(self.workbook), self.title))

    def _call(self, method: str, **args: Any) -> Any:
        return self.workbook._call(method, sheet=self.title, **args)

    @property
    def dimensions(self) -> str:
        """Bounding box of non-empty cells, e.g. ``"A1:C10"`` (``""`` if empty)."""
        return self._call("dimension")

    def cell(self, ref: str) -> str:
        """Value of the cell ``ref`` as displayed by Excel (``""`` if empty)."""
        return self._call("cell_value", cell=ref)

    __getitem__ = cell

    def formula(self, ref: str) -> str:
        """Formula of the cell ``ref`` (``""`` if it has none)."""
        return self._call("cell_formula", cell=ref)

    @property
    def rows(self) -> List[List[str]]:
        """All rows. Trailing empty cells are trimmed, so lengths may differ."""
        return [list(r) for r in self._call("rows")]

    def iter_rows(self) -> Iterator[List[str]]:
        return iter(self.rows)

    def to_records(self, header_row: int = 1) -> List[Dict[str, str]]:
        """Rows below ``header_row`` (1-based) as dicts keyed by header text.

        Short rows are padded with ``""``; blank header cells are skipped.
        """
        if header_row < 1:
            raise ValueError("header_row must be >= 1")
        rows = self.rows
        if len(rows) < header_row:
            return []
        header = rows[header_row - 1]
        records = []
        for row in rows[header_row:]:
            padded = row + [""] * (len(header) - len(row))
            records.append({h: v for h, v in zip(header, padded) if h})
        return records

    @property
    def merged_cells(self) -> List[MergedRange]:
        return [MergedRange.from_dict(d) for d in self._call("merged_cells")]

    @property
    def controls(self) -> List[FormControl]:
        """Legacy form controls placed on this sheet."""
        return [FormControl.from_dict(d) for d in self._call("form_controls")]

    def controls_of(self, kind: Union[ControlType, str]) -> List[FormControl]:
        """Controls of one :class:`ControlType`, e.g. ``ControlType.CHECK_BOX``."""
        kind = ControlType(kind)
        return [c for c in self.controls if c.type is kind]


def open(path: PathLike, *, password: Optional[str] = None) -> Workbook:  # noqa: A001
    """Open the workbook at ``path``."""
    lib = get_library()
    return Workbook(lib.open(os.fspath(path), password), lib)


def load(
    source: Union[bytes, bytearray, memoryview, BinaryIO], *, password: Optional[str] = None
) -> Workbook:
    """Open a workbook from bytes or a binary file object."""
    if hasattr(source, "read"):
        source = source.read()  # type: ignore[union-attr]
    lib = get_library()
    return Workbook(lib.open_bytes(bytes(source), password), lib)  # type: ignore[arg-type]

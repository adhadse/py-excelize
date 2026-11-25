"""py-excelize: read Excel workbooks, including legacy form controls.

>>> import py_excelize as pex
>>> with pex.open("form.xlsx") as wb:          # doctest: +SKIP
...     for control in wb["Sheet1"].controls:
...         print(control.type, control.cell, control.checked)
"""

from ._version import __version__
from .errors import (
    ExcelizeError,
    InvalidCellError,
    InvalidWorkbookError,
    PasswordError,
    SheetNotFoundError,
    WorkbookClosedError,
    WorkbookNotFoundError,
)
from .models import ControlType, FormControl, MergedRange
from .workbook import Workbook, Worksheet, load, open

__all__ = [
    "__version__",
    "open",
    "load",
    "Workbook",
    "Worksheet",
    "ControlType",
    "FormControl",
    "MergedRange",
    "ExcelizeError",
    "InvalidCellError",
    "InvalidWorkbookError",
    "PasswordError",
    "SheetNotFoundError",
    "WorkbookClosedError",
    "WorkbookNotFoundError",
]

"""Exception hierarchy.

Every error raised by this package derives from :class:`ExcelizeError`. Where
a built-in exception is a natural fit the error also derives from it, so
``except FileNotFoundError`` or ``except KeyError`` keep working.
"""

from __future__ import annotations

__all__ = [
    "ExcelizeError",
    "WorkbookNotFoundError",
    "InvalidWorkbookError",
    "PasswordError",
    "WorkbookClosedError",
    "SheetNotFoundError",
    "InvalidCellError",
]


class ExcelizeError(Exception):
    """Base class for all py-excelize errors."""


class WorkbookNotFoundError(ExcelizeError, FileNotFoundError):
    """The workbook file does not exist."""


class InvalidWorkbookError(ExcelizeError, ValueError):
    """The data is not a readable XLSX workbook."""


class PasswordError(ExcelizeError, ValueError):
    """The workbook is encrypted and the password is missing or wrong."""


class WorkbookClosedError(ExcelizeError, ValueError):
    """The workbook (or one of its worksheets) was used after ``close()``."""


class SheetNotFoundError(ExcelizeError, KeyError):
    """No sheet with the requested name exists."""

    def __str__(self) -> str:
        return Exception.__str__(self)


class InvalidCellError(ExcelizeError, ValueError):
    """A cell reference such as ``"A1"`` is malformed."""


_BY_KIND = {
    "closed": WorkbookClosedError,
    "sheet_not_found": SheetNotFoundError,
    "invalid_cell": InvalidCellError,
    "file_not_found": WorkbookNotFoundError,
    "format": InvalidWorkbookError,
    "password": PasswordError,
}


def error_from_kind(kind: str, message: str) -> ExcelizeError:
    """Build the exception matching an error ``kind`` reported by the Go core."""
    return _BY_KIND.get(kind, ExcelizeError)(message)

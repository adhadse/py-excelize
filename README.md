# py-excelize

[![CI](https://github.com/adhadse/py-excelize/actions/workflows/ci.yml/badge.svg)](https://github.com/adhadse/py-excelize/actions/workflows/ci.yml)
[![PyPI](https://img.shields.io/pypi/v/py-excelize.svg)](https://pypi.org/project/py-excelize/)
[![Go Reference](https://pkg.go.dev/badge/github.com/adhadse/py-excelize.svg)](https://pkg.go.dev/github.com/adhadse/py-excelize)

Read Excel workbooks from Python, **including legacy form controls** (check boxes, option
buttons, spin buttons, scroll bars, ...), which most spreadsheet readers ignore. Powered by the
Go library [excelize](https://github.com/xuri/excelize).

- Typed, Pythonic API: context managers, dataclasses, enums, a real exception hierarchy.
- Prebuilt wheels for **Linux** (x86_64, aarch64), **macOS** (Intel, Apple Silicon) and
  **Windows** (x64). No compiler needed to install.
- Works on Python 3.9+ (the wheel is ABI independent).
- Also usable as a plain Go library: see [pkg.go.dev](https://pkg.go.dev/github.com/adhadse/py-excelize).

## Install

```bash
pip install py-excelize
```

## Quick start

```python
import py_excelize as pex

with pex.open("form.xlsx") as wb:
    print(wb.sheetnames)                      # ['Sheet 1', 'Plain']

    ws = wb["Sheet 1"]
    for control in ws.controls:
        print(control.type, control.cell, control.checked, control.text)
    # check_box  A1 True  Agree
    # check_box  A3 False Subscribe

    ticked = [c.text for c in ws.controls_of(pex.ControlType.CHECK_BOX) if c.checked]
```

### Cells and rows

```python
with pex.open("data.xlsx") as wb:
    ws = wb["Sheet1"]
    ws["A2"]                # 'apple'      (displayed value)
    ws.formula("D2")        # 'B2*C2'
    ws.dimensions           # 'A1:F7'
    ws.rows                 # [['name', 'qty'], ['apple', '3'], ...]
    ws.to_records()         # [{'name': 'apple', 'qty': '3'}, ...]  first row = header
    ws.merged_cells         # [MergedRange(range='A7:C7', ...)]
```

### In-memory and encrypted workbooks

```python
wb = pex.load(response.content)                       # bytes or any binary file object
wb = pex.open("secret.xlsx", password="hunter2")
```

### Errors

All exceptions derive from `py_excelize.ExcelizeError` *and* a matching built-in:

| Exception | Also a | Raised when |
|---|---|---|
| `WorkbookNotFoundError` | `FileNotFoundError` | the file does not exist |
| `InvalidWorkbookError` | `ValueError` | the data is not an XLSX file |
| `PasswordError` | `ValueError` | encrypted workbook and password missing/wrong |
| `SheetNotFoundError` | `KeyError` | `wb["nope"]` |
| `InvalidCellError` | `ValueError` | `ws["A0"]`, `ws["1A"]`, ... |
| `WorkbookClosedError` | `ValueError` | using a workbook after `close()` |

## API overview

| Object | Members |
|---|---|
| `pex.open(path, *, password=None)` / `pex.load(data, *, password=None)` | return a `Workbook` |
| `Workbook` | `sheetnames`, `worksheets`, `wb[name or index]`, `in`, `len`, iteration, `get`, `controls`, `to_dict()`, `close()`, `closed` |
| `Worksheet` | `title`, `dimensions`, `ws[ref]` / `cell(ref)`, `formula(ref)`, `rows`, `iter_rows()`, `to_records(header_row=1)`, `merged_cells`, `controls`, `controls_of(type)` |
| `FormControl` | frozen dataclass: `sheet, type, cell, linked_cell, checked, current_value, min_value, max_value, inc_change, page_change, horizontal, macro, text, paragraphs, offset_x, offset_y`, plus `to_dict()` |
| `ControlType` | `CHECK_BOX, OPTION_BUTTON, SPIN_BUTTON, SCROLL_BAR, GROUP_BOX, LABEL, BUTTON, NOTE, UNKNOWN` (a `str` enum) |

Workbooks are safe to share between Python threads (calls are serialised per workbook).

## Use from Go

```go
wb, err := pyexcelize.Open("form.xlsx")
if err != nil { log.Fatal(err) }
defer wb.Close()

controls, _ := wb.FormControls("Sheet 1")
for _, c := range controls {
    fmt.Println(c.Type, c.Cell, c.Checked, c.Text)
}
```

```bash
go get github.com/adhadse/py-excelize
```

Full reference with runnable examples: <https://pkg.go.dev/github.com/adhadse/py-excelize>

## How it works

```
Python (ctypes, dataclasses)  ->  libpyexcelize (Go, -buildmode=c-shared, JSON over a tiny C ABI)
                                       ->  root Go package  ->  excelize
```

The Go package is the source of truth; `internal/capi` implements the JSON protocol and
`cmd/libpyexcelize` exports it as five C functions.

## Development

Requirements: Go (see `go.mod`), a C compiler, Python 3.9+.

```bash
go test -race -cover ./...            # Go tests
pip install -e ".[test]" && pytest    # builds the Go library, runs Python tests
go run ./cmd/genfixtures              # regenerate testdata/*.xlsx
```

`PY_EXCELIZE_SKIP_BUILD=1` skips the Go build during `pip install` if you already built
`src/py_excelize/libpyexcelize.*` yourself.

### Releasing

1. Bump the version in `src/py_excelize/_version.py` **and** `internal/capi/capi.go`.
2. Update `CHANGELOG.md`, commit, then `git tag vX.Y.Z && git push --tags`.
3. The *Release* workflow tests, builds wheels for every platform, verifies them on Python
   3.9-3.13 and publishes to PyPI via [trusted publishing](https://docs.pypi.org/trusted-publishers/).

## License

MIT, see [LICENSE](LICENSE). excelize is BSD-3-Clause licensed.

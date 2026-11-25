# Changelog

## 0.1.0

Complete rewrite as **py-excelize**.

- New public Go package (`github.com/adhadse/py-excelize`) with documented API and examples.
- New Pythonic API: `open`/`load`, `Workbook`, `Worksheet`, `FormControl`, `ControlType`, typed errors.
- Native library is now loaded through ctypes (Go `c-shared` + JSON) instead of gopy:
  one ABI-independent wheel per platform.
- Wheels for Linux x86_64/aarch64, macOS x86_64/arm64 and Windows x64.
- Fixed: control text was always empty; `Type` was always `"Unknown"`.
- Added: cell values/formulas, rows, merged cells, dimensions, password-protected workbooks.

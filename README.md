# Excel extractor

Extract excel form content into structured data.

## Python package (preview)

We are introducing a minimal Python package `py-excelize` backed by Go `excelize` via gopy. Initial API:

- Open a workbook and list sheets
- List form controls on a sheet

Example (to be built in CI):

```python
import py_excelize as pex

wb = pex.open("/path/to/workbook.xlsx")
print(wb.sheets())
controls = wb.list_controls("Sheet1")
for c in controls:
    print(c.type, c.cell, c.linked_cell, c.checked, c.text)
wb.close()()
```

This is a foundation for a pandas-compatible reader with form controls support.

## Usage

1. SECCF extraction: supplier export control classification Form/declaration

```python
from py_excelize import *

company_names = extractor.CompanyNameList()  # the company name which can be mentioned in the file
for company_name in ["Amazon", "Amazon Ltd"]:
    company_names.append(company_name)

extr = extractor.make_seccf_extractor("Example.xlsx", company_names)
extraction = extr.extract()

# convert to JSON string
extr_json = extr.to_json()
```

## BUILD

1. Building the go binary

```bash
go build -o gobinary ./pkg/pyexcelize/pyexcelize.go
```

2. Running the program without building the binary

```bash
go build -o ./bin/excel-extrator ./cmd/excelFormExtractor/main.go
```

3. Run the binary:

```bash
./bin/excel-extrator
```

## Local Python bindings generation and installation

<!-- ```bash
pip3 install pybindgen wheel
gopy build --output=py_excelize -vm=python3 ./pkg/pyexcelize
RELEASE_VERSION=YOUR_UPDATED_PACKAGE_VERSION python3 setup_ci.py build_ext -i --force

# install module in place (editable build places compiled module under build/lib*/py_excelize)
python -c "import sys,glob; import site; import shutil; import os; import pathlib as p; d=glob.glob('build/lib.*')[0]; src=os.path.join(d,'py_excelize'); dst=site.getusersitepackages(); shutil.copytree(src, os.path.join(dst,'py_excelize'), dirs_exist_ok=True); print('installed to', dst)"
``` -->

```bash
pip3 install pybindgen wheel
make regen
# Install the Python package in editable mode
pip install -e .
# verify import
python -c 'import py_excelize; from py_excelize import _pyexcelize; print("ok")'
```

If you see a loader error about .so not found, run from the repo (or point to your site-packages path):

```bash
export LD_LIBRARY_PATH=$PWD/py_excelize:$LD_LIBRARY_PATH
python -c 'import py_excelize; from py_excelize import _pyexcelize; print("ok")'
```

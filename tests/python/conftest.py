from pathlib import Path

import pytest

import py_excelize as pex

TESTDATA = Path(__file__).resolve().parents[2] / "testdata"
PASSWORD = "secret"

# (cell, type, checked, linked_cell, text) for every control in controls.xlsx
EXPECTED_CONTROLS = [
    ("A1", "check_box", True, "$D$1", "Agree"),
    ("A3", "check_box", False, "$D$3", "Subscribe"),
    ("A5", "option_button", True, "", "Option A"),
    ("A7", "option_button", False, "", "Option B"),
    ("A9", "spin_button", False, "$D$9", ""),
    ("A11", "scroll_bar", False, "$D$11", ""),
    ("A13", "group_box", False, "", "Group"),
    ("A15", "label", False, "", "A label"),
    ("A17", "button", False, "", "Click"),
    ("A19", "check_box", False, "", "Line one Line two"),
]


@pytest.fixture(scope="session")
def testdata() -> Path:
    return TESTDATA


@pytest.fixture
def cells_path(testdata):
    return testdata / "cells.xlsx"


@pytest.fixture
def controls_path(testdata):
    return testdata / "controls.xlsx"


@pytest.fixture
def empty_path(testdata):
    return testdata / "empty.xlsx"


@pytest.fixture
def encrypted_path(testdata):
    return testdata / "encrypted.xlsx"


@pytest.fixture
def cells_wb(cells_path):
    with pex.open(cells_path) as wb:
        yield wb


@pytest.fixture
def controls_wb(controls_path):
    with pex.open(controls_path) as wb:
        yield wb


@pytest.fixture
def empty_wb(empty_path):
    with pex.open(empty_path) as wb:
        yield wb

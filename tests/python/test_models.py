import dataclasses
import json

import pytest

from py_excelize import ControlType, FormControl, MergedRange

BASE = {"sheet": "S", "type_name": "check_box", "cell": "A1"}


@pytest.mark.parametrize(
    "name, member",
    [
        ("note", ControlType.NOTE),
        ("button", ControlType.BUTTON),
        ("option_button", ControlType.OPTION_BUTTON),
        ("spin_button", ControlType.SPIN_BUTTON),
        ("check_box", ControlType.CHECK_BOX),
        ("group_box", ControlType.GROUP_BOX),
        ("label", ControlType.LABEL),
        ("scroll_bar", ControlType.SCROLL_BAR),
        ("unknown", ControlType.UNKNOWN),
    ],
)
def test_control_type_from_name(name, member):
    assert ControlType(name) is member
    assert member == name
    assert str(member) == name
    assert member.value == name


@pytest.mark.parametrize("bad", ["", "CHECK_BOX", "slider", "check box", None, 4])
def test_control_type_unknown_values_become_unknown(bad):
    assert ControlType(bad) is ControlType.UNKNOWN


def test_control_type_is_str():
    assert isinstance(ControlType.CHECK_BOX, str)
    assert f"{ControlType.LABEL}" == "label"


def test_control_type_has_nine_members():
    assert len(ControlType) == 9


def test_from_dict_defaults():
    c = FormControl.from_dict(BASE)
    assert c == FormControl(sheet="S", type=ControlType.CHECK_BOX, cell="A1")
    assert c.paragraphs == []
    assert c.checked is False
    assert c.current_value == 0


def test_from_dict_full():
    c = FormControl.from_dict(
        {
            **BASE,
            "type_name": "scroll_bar",
            "linked_cell": "$A$1",
            "checked": 1,
            "current_value": 3,
            "min_value": 1,
            "max_value": 9,
            "inc_change": 2,
            "page_change": 4,
            "horizontal": True,
            "macro": "m",
            "text": "t",
            "paragraphs": ["a", "b"],
            "offset_x": 5,
            "offset_y": -6,
        }
    )
    assert c.type is ControlType.SCROLL_BAR
    assert c.checked is True
    assert (c.current_value, c.min_value, c.max_value) == (3, 1, 9)
    assert (c.inc_change, c.page_change) == (2, 4)
    assert c.horizontal and c.macro == "m" and c.text == "t"
    assert c.paragraphs == ["a", "b"]
    assert (c.offset_x, c.offset_y) == (5, -6)


def test_from_dict_none_paragraphs_become_empty_list():
    assert FormControl.from_dict({**BASE, "paragraphs": None}).paragraphs == []


@pytest.mark.parametrize("missing", ["sheet", "type_name", "cell"])
def test_from_dict_requires_core_keys(missing):
    data = dict(BASE)
    del data[missing]
    with pytest.raises(KeyError):
        FormControl.from_dict(data)


def test_unknown_type_name_does_not_raise():
    assert FormControl.from_dict({**BASE, "type_name": "mystery"}).type is ControlType.UNKNOWN


def test_form_control_is_frozen():
    c = FormControl.from_dict(BASE)
    with pytest.raises(dataclasses.FrozenInstanceError):
        c.cell = "B2"  # type: ignore[misc]


def test_form_control_equality_and_replace():
    a = FormControl.from_dict(BASE)
    b = dataclasses.replace(a, checked=True)
    assert a != b
    assert a == FormControl.from_dict(BASE)


def test_to_dict_is_json_serialisable():
    c = FormControl.from_dict({**BASE, "paragraphs": ["x"]})
    d = c.to_dict()
    assert d["type"] == "check_box"
    assert json.loads(json.dumps(d)) == d


def test_to_dict_keys_match_dataclass_fields():
    c = FormControl.from_dict(BASE)
    assert set(c.to_dict()) == {f.name for f in dataclasses.fields(FormControl)}


def test_to_dict_round_trips_through_from_dict():
    c = FormControl.from_dict({**BASE, "checked": True, "text": "hi", "paragraphs": ["hi"]})
    d = c.to_dict()
    d["type_name"] = d.pop("type")
    assert FormControl.from_dict(d) == c


def test_merged_range_from_dict():
    m = MergedRange.from_dict({"range": "A1:B2", "start": "A1", "end": "B2", "value": "v"})
    assert (m.range, m.start, m.end, m.value) == ("A1:B2", "A1", "B2", "v")
    assert str(m) == "A1:B2"


def test_merged_range_value_default():
    assert MergedRange.from_dict({"range": "A1:B2", "start": "A1", "end": "B2"}).value == ""


def test_merged_range_is_frozen_and_hashable():
    m = MergedRange("A1:B2", "A1", "B2")
    assert hash(m) == hash(MergedRange("A1:B2", "A1", "B2"))
    with pytest.raises(dataclasses.FrozenInstanceError):
        m.value = "x"  # type: ignore[misc]

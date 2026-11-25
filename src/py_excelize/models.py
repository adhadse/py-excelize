"""Plain data objects returned by the API."""

from __future__ import annotations

import enum
from dataclasses import asdict, dataclass, field
from typing import Any, Dict, List

__all__ = ["ControlType", "FormControl", "MergedRange"]


class ControlType(str, enum.Enum):
    """Kind of a legacy form control. Compares equal to its string value."""

    NOTE = "note"
    BUTTON = "button"
    OPTION_BUTTON = "option_button"
    SPIN_BUTTON = "spin_button"
    CHECK_BOX = "check_box"
    GROUP_BOX = "group_box"
    LABEL = "label"
    SCROLL_BAR = "scroll_bar"
    UNKNOWN = "unknown"

    @classmethod
    def _missing_(cls, value: object) -> "ControlType":
        return cls.UNKNOWN

    def __str__(self) -> str:
        return self.value


@dataclass(frozen=True)
class FormControl:
    """A legacy Excel form control (check box, option button, ...)."""

    sheet: str
    type: ControlType
    cell: str
    linked_cell: str = ""
    checked: bool = False
    current_value: int = 0
    min_value: int = 0
    max_value: int = 0
    inc_change: int = 0
    page_change: int = 0
    horizontal: bool = False
    macro: str = ""
    text: str = ""
    paragraphs: List[str] = field(default_factory=list)
    offset_x: int = 0
    offset_y: int = 0

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "FormControl":
        """Build a control from the JSON emitted by the Go core."""
        return cls(
            sheet=data["sheet"],
            type=ControlType(data["type_name"]),
            cell=data["cell"],
            linked_cell=data.get("linked_cell", ""),
            checked=bool(data.get("checked", False)),
            current_value=int(data.get("current_value", 0)),
            min_value=int(data.get("min_value", 0)),
            max_value=int(data.get("max_value", 0)),
            inc_change=int(data.get("inc_change", 0)),
            page_change=int(data.get("page_change", 0)),
            horizontal=bool(data.get("horizontal", False)),
            macro=data.get("macro", ""),
            text=data.get("text", ""),
            paragraphs=list(data.get("paragraphs") or []),
            offset_x=int(data.get("offset_x", 0)),
            offset_y=int(data.get("offset_y", 0)),
        )

    def to_dict(self) -> Dict[str, Any]:
        """Return a JSON-serialisable dict (``type`` becomes its string value)."""
        d = asdict(self)
        d["type"] = self.type.value
        return d


@dataclass(frozen=True)
class MergedRange:
    """A merged cell range such as ``A1:C1``."""

    range: str
    start: str
    end: str
    value: str = ""

    def __str__(self) -> str:
        return self.range

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "MergedRange":
        return cls(data["range"], data["start"], data["end"], data.get("value", ""))

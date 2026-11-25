package pyexcelize

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// ControlType identifies the kind of a legacy form control.
type ControlType int

// Supported control types. The numeric values mirror excelize.FormControlType
// for every type excelize knows about.
const (
	ControlNote         ControlType = ControlType(excelize.FormControlNote)
	ControlButton       ControlType = ControlType(excelize.FormControlButton)
	ControlOptionButton ControlType = ControlType(excelize.FormControlOptionButton)
	ControlSpinButton   ControlType = ControlType(excelize.FormControlSpinButton)
	ControlCheckBox     ControlType = ControlType(excelize.FormControlCheckBox)
	ControlGroupBox     ControlType = ControlType(excelize.FormControlGroupBox)
	ControlLabel        ControlType = ControlType(excelize.FormControlLabel)
	ControlScrollBar    ControlType = ControlType(excelize.FormControlScrollBar)
)

var controlNames = map[ControlType]string{
	ControlNote:         "note",
	ControlButton:       "button",
	ControlOptionButton: "option_button",
	ControlSpinButton:   "spin_button",
	ControlCheckBox:     "check_box",
	ControlGroupBox:     "group_box",
	ControlLabel:        "label",
	ControlScrollBar:    "scroll_bar",
}

// String returns the snake_case name of the control type, or "unknown".
func (t ControlType) String() string {
	if n, ok := controlNames[t]; ok {
		return n
	}
	return "unknown"
}

// FormControl is a normalized view of a legacy form control on a sheet.
type FormControl struct {
	Sheet        string      `json:"sheet"`
	Type         ControlType `json:"type"`
	TypeName     string      `json:"type_name"`
	Cell         string      `json:"cell"`
	LinkedCell   string      `json:"linked_cell"`
	Checked      bool        `json:"checked"`
	CurrentValue uint        `json:"current_value"`
	MinValue     uint        `json:"min_value"`
	MaxValue     uint        `json:"max_value"`
	IncChange    uint        `json:"inc_change"`
	PageChange   uint        `json:"page_change"`
	Horizontal   bool        `json:"horizontal"`
	Macro        string      `json:"macro"`
	Text         string      `json:"text"`
	Paragraphs   []string    `json:"paragraphs"`
	OffsetX      int         `json:"offset_x"`
	OffsetY      int         `json:"offset_y"`
}

func newFormControl(sheet string, c excelize.FormControl) FormControl {
	t := ControlType(c.Type)
	texts := make([]string, 0, len(c.Paragraph)+1)
	texts = append(texts, c.Text)
	for _, r := range c.Paragraph {
		texts = append(texts, r.Text)
	}
	return FormControl{
		Sheet:        sheet,
		Type:         t,
		TypeName:     t.String(),
		Cell:         c.Cell,
		LinkedCell:   c.CellLink,
		Checked:      c.Checked,
		CurrentValue: c.CurrentVal,
		MinValue:     c.MinVal,
		MaxValue:     c.MaxVal,
		IncChange:    c.IncChange,
		PageChange:   c.PageChange,
		Horizontal:   c.Horizontally,
		Macro:        c.Macro,
		Text:         flattenRuns(texts),
		Paragraphs:   runTexts(texts),
		OffsetX:      c.Format.OffsetX,
		OffsetY:      c.Format.OffsetY,
	}
}

// flattenRuns joins the trimmed, non-empty fragments with a single space.
func flattenRuns(texts []string) string {
	return strings.Join(runTexts(texts), " ")
}

// runTexts returns the trimmed, non-empty fragments. It never returns nil.
func runTexts(texts []string) []string {
	out := make([]string, 0, len(texts))
	for _, t := range texts {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}

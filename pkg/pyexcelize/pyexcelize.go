package pyexcelize

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Workbook is a lightweight handle around excelize.File exposed to Python via gopy.
type Workbook struct {
	f *excelize.File
}

// Control represents a normalized form control on a sheet.
type Control struct {
	Sheet      string   // Sheet name
	Type       string   // Human-friendly type name when known, otherwise "Unknown"
	TypeCode   int      // Raw excelize form control type code
	Cell       string   // Anchor cell
	LinkedCell string   // Linked cell address if present
	Checked    bool     // For checkboxes/option buttons
	CurrentVal string   // For list/spin/scroll current value
	Text       string   // Flattened text/paragraph content
	Options    []string // Optional: extracted paragraph texts
	OffsetX    int      // Drawing offset X
	OffsetY    int      // Drawing offset Y
}

// Open opens an Excel workbook and returns a handle.
func Open(path string) (*Workbook, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open workbook: %w", err)
	}
	return &Workbook{f: f}, nil
}

// Close closes the underlying workbook.
func (w *Workbook) Close() error {
	if w == nil || w.f == nil {
		return nil
	}
	return w.f.Close()
}

// Sheets returns the list of sheet names in the workbook.
func (w *Workbook) Sheets() []string {
	if w == nil || w.f == nil {
		return []string{}
	}
	return w.f.GetSheetList()
}

// ListControls lists all form controls on the given sheet.
func (w *Workbook) ListControls(sheet string) ([]*Control, error) {
	if w == nil || w.f == nil {
		return nil, fmt.Errorf("workbook is not open")
	}
	controls, err := w.f.GetFormControls(sheet)
	if err != nil {
		return nil, fmt.Errorf("get form controls: %w", err)
	}
	var out []*Control
	for _, c := range controls {
		out = append(out, &Control{
			Sheet:      sheet,
			Type:       "Unknown",
			TypeCode:   int(c.Type),
			Cell:       c.Cell,
			LinkedCell: c.CellLink,
			Checked:    c.Checked,
			CurrentVal: fmt.Sprint(c.CurrentVal),
			Text:       flattenParagraph(c.Paragraph),
			Options:    paragraphTexts(c.Paragraph),
			OffsetX:    c.Format.OffsetX,
			OffsetY:    c.Format.OffsetY,
		})
	}
	return out, nil
}

func flattenParagraph(pr []excelize.RichTextRun) string {
	if len(pr) == 0 {
		return ""
	}
	var parts []string
	for _, p := range pr {
		parts = append(parts, p.Text)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func paragraphTexts(pr []excelize.RichTextRun) []string {
	if len(pr) == 0 {
		return nil
	}
	var parts []string
	for _, p := range pr {
		text := strings.TrimSpace(p.Text)
		if text != "" {
			parts = append(parts, text)
		}
	}
	return parts
}

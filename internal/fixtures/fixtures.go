// Package fixtures builds the small workbooks used by the Go and Python test
// suites. Everything is generated with excelize so no binary blobs need to be
// reviewed by hand.
package fixtures

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

// Password protects the workbook returned by Encrypted.
const Password = "secret"

// Names of the files written by WriteAll.
const (
	FileEmpty     = "empty.xlsx"
	FileCells     = "cells.xlsx"
	FileControls  = "controls.xlsx"
	FileEncrypted = "encrypted.xlsx"
)

// Empty returns a workbook with a single blank "Sheet1".
func Empty() *excelize.File { return excelize.NewFile() }

// Cells returns a workbook with plain data on Sheet1, merged cells, a formula
// and extra sheets with awkward names.
func Cells() *excelize.File {
	f := excelize.NewFile()
	must(f.SetSheetRow("Sheet1", "A1", &[]any{"name", "qty", "price"}))
	must(f.SetSheetRow("Sheet1", "A2", &[]any{"apple", 3, 1.5}))
	must(f.SetSheetRow("Sheet1", "A3", &[]any{"pear", 10, 0.25}))
	must(f.SetCellFormula("Sheet1", "D2", "B2*C2"))
	must(f.SetCellBool("Sheet1", "F2", true))
	must(f.SetCellValue("Sheet1", "A5", "gap row"))
	must(f.MergeCell("Sheet1", "A7", "C7"))
	must(f.SetCellValue("Sheet1", "A7", "merged title"))
	must(f.MergeCell("Sheet1", "E1", "E3"))
	_, err := f.NewSheet("Sheet 2")
	must(err)
	must(f.SetCellValue("Sheet 2", "B2", "spaced"))
	_, err = f.NewSheet("Ünï Çødé")
	must(err)
	must(f.SetCellValue("Ünï Çødé", "A1", "héllo wörld ✓"))
	return f
}

// Controls returns a workbook whose "Sheet 1" holds one control of every
// supported type and whose "Plain" sheet holds none.
func Controls() *excelize.File {
	f := excelize.NewFile()
	must(f.SetSheetName("Sheet1", "Sheet 1"))
	add := func(c excelize.FormControl) { must(f.AddFormControl("Sheet 1", c)) }
	add(excelize.FormControl{Cell: "A1", Type: excelize.FormControlCheckBox, Text: "Agree", Checked: true, CellLink: "$D$1"})
	add(excelize.FormControl{Cell: "A3", Type: excelize.FormControlCheckBox, Text: "Subscribe", CellLink: "$D$3"})
	add(excelize.FormControl{Cell: "A5", Type: excelize.FormControlOptionButton, Text: "Option A", Checked: true, CellLink: "$D$5"})
	add(excelize.FormControl{Cell: "A7", Type: excelize.FormControlOptionButton, Text: "Option B", CellLink: "$D$5"})
	add(excelize.FormControl{Cell: "A9", Type: excelize.FormControlSpinButton, CurrentVal: 7, MinVal: 1, MaxVal: 20, IncChange: 2, CellLink: "$D$9"})
	add(excelize.FormControl{Cell: "A11", Type: excelize.FormControlScrollBar, CurrentVal: 30, MinVal: 0, MaxVal: 100, IncChange: 1, PageChange: 10, Horizontally: true, CellLink: "$D$11"})
	add(excelize.FormControl{Cell: "A13", Type: excelize.FormControlGroupBox, Text: "Group"})
	add(excelize.FormControl{Cell: "A15", Type: excelize.FormControlLabel, Text: "A label"})
	add(excelize.FormControl{Cell: "A17", Type: excelize.FormControlButton, Text: "Click", Macro: "Button1_Click"})
	add(excelize.FormControl{Cell: "A19", Type: excelize.FormControlCheckBox, Paragraph: []excelize.RichTextRun{{Text: "Line one"}, {Text: "Line two"}}})
	_, err := f.NewSheet("Plain")
	must(err)
	return f
}

// WriteAll saves every fixture into dir (created if needed) and returns the
// paths written, keyed by file name.
func WriteAll(dir string) (map[string]string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	builders := map[string]func() (*excelize.File, excelize.Options){
		FileEmpty:     func() (*excelize.File, excelize.Options) { return Empty(), excelize.Options{} },
		FileCells:     func() (*excelize.File, excelize.Options) { return Cells(), excelize.Options{} },
		FileControls:  func() (*excelize.File, excelize.Options) { return Controls(), excelize.Options{} },
		FileEncrypted: func() (*excelize.File, excelize.Options) { return Cells(), excelize.Options{Password: Password} },
	}
	out := make(map[string]string, len(builders))
	for name, build := range builders {
		f, opts := build()
		p := filepath.Join(dir, name)
		err := f.SaveAs(p, opts)
		_ = f.Close()
		if err != nil {
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
		out[name] = p
	}
	return out, nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

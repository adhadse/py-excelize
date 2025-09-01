package tests

import (
	"fmt"
	"testing"

	lib "github.com/adhadse/pyexcelize/pkg/pyexcelize"
)

// TestListControlsWrapper opens the workbook via our wrapper and logs controls on Sheet 1.
func TestListControlsWrapper(t *testing.T) {
	const filePath = "/home/adhadse/Downloads/test_form_controls_0.xlsx"
	const sheetName = "Sheet 1"

	wb, err := lib.Open(filePath)
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer func() {
		if cerr := wb.Close(); cerr != nil {
			t.Fatalf("close workbook: %v", cerr)
		}
	}()

	controls, err := wb.ListControls(sheetName)
	if err != nil {
		t.Fatalf("list controls: %v", err)
	}

	if len(controls) == 0 {
		t.Log("no form controls found")
	}

	for idx, c := range controls {
		fmt.Printf("Control %d: type=%s(typeCode=%d) cell=%s linked=%s checked=%v text=%s current=%s opts=%v off=(%d,%d)\n",
			idx, c.Type, c.TypeCode, c.Cell, c.LinkedCell, c.Checked, c.Text, c.CurrentVal, c.Options, c.OffsetX, c.OffsetY)
	}
}

package pyexcelize_test

import (
	"errors"
	"fmt"

	"github.com/xuri/excelize/v2"

	pyexcelize "github.com/adhadse/py-excelize"
)

func exampleWorkbook() *pyexcelize.Workbook {
	f := excelize.NewFile()
	_ = f.SetSheetRow("Sheet1", "A1", &[]any{"name", "qty"})
	_ = f.SetSheetRow("Sheet1", "A2", &[]any{"apple", 3})
	_ = f.SetCellFormula("Sheet1", "C2", "B2*2")
	_ = f.AddFormControl("Sheet1", excelize.FormControl{
		Cell: "E1", Type: excelize.FormControlCheckBox, Text: "Agree", Checked: true, CellLink: "$F$1",
	})
	_ = f.AddFormControl("Sheet1", excelize.FormControl{
		Cell: "E3", Type: excelize.FormControlSpinButton, CurrentVal: 4, MinVal: 0, MaxVal: 10, IncChange: 1,
	})
	return pyexcelize.FromFile(f)
}

func ExampleFromFile() {
	wb := exampleWorkbook()
	defer wb.Close()

	fmt.Println(wb.SheetNames())
	fmt.Println(wb.HasSheet("sheet1"))
	// Output:
	// [Sheet1]
	// true
}

func ExampleWorkbook_FormControls() {
	wb := exampleWorkbook()
	defer wb.Close()

	controls, err := wb.FormControls("Sheet1")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, c := range controls {
		fmt.Println(c.Type, c.Cell, c.Checked, c.CurrentValue, c.Text)
	}
	// Output:
	// check_box E1 true 0 Agree
	// spin_button E3 false 4
}

func ExampleWorkbook_Rows() {
	wb := exampleWorkbook()
	defer wb.Close()

	rows, _ := wb.Rows("Sheet1")
	for _, row := range rows {
		fmt.Printf("%q\n", row)
	}
	dim, _ := wb.Dimension("Sheet1")
	formula, _ := wb.CellFormula("Sheet1", "C2")
	fmt.Println(dim, formula)
	// Output:
	// ["name" "qty"]
	// ["apple" "3" ""]
	// A1:B2 B2*2
}

func ExampleErrorKind() {
	wb := exampleWorkbook()
	defer wb.Close()

	_, err := wb.CellValue("Missing", "A1")
	fmt.Println(pyexcelize.ErrorKind(err))

	_, err = wb.CellValue("Sheet1", "1A")
	fmt.Println(pyexcelize.ErrorKind(err))

	_, err = pyexcelize.OpenBytes([]byte("not a workbook"))
	fmt.Println(pyexcelize.ErrorKind(err))

	_, err = pyexcelize.Open("does-not-exist.xlsx")
	fmt.Println(pyexcelize.ErrorKind(err))

	_ = wb.Close()
	_, err = wb.Rows("Sheet1")
	fmt.Println(pyexcelize.ErrorKind(err), errors.Is(err, pyexcelize.ErrClosed))
	// Output:
	// sheet_not_found
	// invalid_cell
	// format
	// file_not_found
	// closed true
}

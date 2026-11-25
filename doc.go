// Package pyexcelize is a small, read-oriented façade over the excelize
// spreadsheet library. It focuses on extracting structured data from
// existing XLSX workbooks, including legacy Form Controls (check boxes,
// option buttons, spin buttons, ...) which most spreadsheet readers ignore.
//
// The same package also powers the py-excelize Python distribution; see
// https://github.com/adhadse/py-excelize for the Python API.
//
// # Quick start
//
//	wb, err := pyexcelize.Open("form.xlsx")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer wb.Close()
//
//	for _, sheet := range wb.SheetNames() {
//	    controls, err := wb.FormControls(sheet)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    for _, c := range controls {
//	        fmt.Println(sheet, c.Type, c.Cell, c.Checked, c.Text)
//	    }
//	}
//
// # Errors
//
// Errors can be inspected with [errors.Is] and [errors.As]: see [ErrClosed],
// [SheetNotFoundError] and [InvalidCellError]. Use [ErrorKind] to obtain a
// stable machine readable classification.
package pyexcelize

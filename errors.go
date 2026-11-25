package pyexcelize

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/fs"

	"github.com/xuri/excelize/v2"
)

// ErrClosed is returned when a method is called on a closed or nil Workbook.
var ErrClosed = errors.New("pyexcelize: workbook is closed")

// ErrPasswordRequired is returned when an encrypted workbook is opened without
// a password.
var ErrPasswordRequired = errors.New("pyexcelize: workbook is encrypted, a password is required")

// ErrWrongPassword is returned when the supplied password does not match.
var ErrWrongPassword = errors.New("pyexcelize: incorrect workbook password")

// SheetNotFoundError is returned when a named sheet does not exist.
type SheetNotFoundError struct {
	Name string
}

// Error implements error.
func (e *SheetNotFoundError) Error() string {
	return fmt.Sprintf("pyexcelize: sheet %q does not exist", e.Name)
}

// InvalidCellError is returned when a cell reference such as "A1" is malformed.
type InvalidCellError struct {
	Ref string
}

// Error implements error.
func (e *InvalidCellError) Error() string {
	return fmt.Sprintf("pyexcelize: invalid cell reference %q", e.Ref)
}

// Error kinds returned by [ErrorKind].
const (
	KindClosed        = "closed"
	KindSheetNotFound = "sheet_not_found"
	KindInvalidCell   = "invalid_cell"
	KindFileNotFound  = "file_not_found"
	KindFormat        = "format"
	KindPassword      = "password"
	KindOther         = "error"
)

// ErrorKind classifies err into one of the Kind* constants. It returns an
// empty string for a nil error.
func ErrorKind(err error) string {
	var (
		sheetErr *SheetNotFoundError
		cellErr  *InvalidCellError
		exSheet  excelize.ErrSheetNotExist
	)
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrClosed):
		return KindClosed
	case errors.As(err, &sheetErr), errors.As(err, &exSheet):
		return KindSheetNotFound
	case errors.As(err, &cellErr):
		return KindInvalidCell
	case errors.Is(err, fs.ErrNotExist):
		return KindFileNotFound
	case errors.Is(err, ErrPasswordRequired), errors.Is(err, ErrWrongPassword):
		return KindPassword
	case errors.Is(err, excelize.ErrWorkbookFileFormat), errors.Is(err, zip.ErrFormat):
		return KindFormat
	}
	return KindOther
}

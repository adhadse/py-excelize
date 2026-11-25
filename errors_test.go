package pyexcelize_test

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	pyexcelize "github.com/adhadse/py-excelize"
)

func TestErrorKind(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"closed", pyexcelize.ErrClosed, pyexcelize.KindClosed},
		{"closed wrapped", fmt.Errorf("ctx: %w", pyexcelize.ErrClosed), pyexcelize.KindClosed},
		{"closed double wrapped", fmt.Errorf("a: %w", fmt.Errorf("b: %w", pyexcelize.ErrClosed)), pyexcelize.KindClosed},
		{"sheet not found", &pyexcelize.SheetNotFoundError{Name: "x"}, pyexcelize.KindSheetNotFound},
		{"sheet not found wrapped", fmt.Errorf("w: %w", &pyexcelize.SheetNotFoundError{Name: "x"}), pyexcelize.KindSheetNotFound},
		{"excelize sheet not exist", excelize.ErrSheetNotExist{SheetName: "x"}, pyexcelize.KindSheetNotFound},
		{"excelize sheet not exist wrapped", fmt.Errorf("w: %w", excelize.ErrSheetNotExist{SheetName: "x"}), pyexcelize.KindSheetNotFound},
		{"invalid cell", &pyexcelize.InvalidCellError{Ref: "!"}, pyexcelize.KindInvalidCell},
		{"invalid cell wrapped", fmt.Errorf("w: %w", &pyexcelize.InvalidCellError{Ref: "!"}), pyexcelize.KindInvalidCell},
		{"fs not exist", fs.ErrNotExist, pyexcelize.KindFileNotFound},
		{"os not exist", &fs.PathError{Op: "open", Path: "/nope", Err: fs.ErrNotExist}, pyexcelize.KindFileNotFound},
		{"os not exist wrapped", fmt.Errorf("w: %w", os.ErrNotExist), pyexcelize.KindFileNotFound},
		{"password required", pyexcelize.ErrPasswordRequired, pyexcelize.KindPassword},
		{"wrong password", pyexcelize.ErrWrongPassword, pyexcelize.KindPassword},
		{"password wrapped", fmt.Errorf("w: %w", pyexcelize.ErrWrongPassword), pyexcelize.KindPassword},
		{"excelize format", excelize.ErrWorkbookFileFormat, pyexcelize.KindFormat},
		{"excelize format wrapped", fmt.Errorf("w: %w", excelize.ErrWorkbookFileFormat), pyexcelize.KindFormat},
		{"zip format", zip.ErrFormat, pyexcelize.KindFormat},
		{"zip format wrapped", fmt.Errorf("w: %w", zip.ErrFormat), pyexcelize.KindFormat},
		{"other", errors.New("boom"), pyexcelize.KindOther},
		{"other wrapped", fmt.Errorf("w: %w", io.ErrUnexpectedEOF), pyexcelize.KindOther},
		{"joined closed", errors.Join(errors.New("x"), pyexcelize.ErrClosed), pyexcelize.KindClosed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := pyexcelize.ErrorKind(tc.err); got != tc.want {
				t.Fatalf("ErrorKind(%v) = %q, want %q", tc.err, got, tc.want)
			}
		})
	}
}

func TestErrorKindPrecedence(t *testing.T) {
	t.Parallel()
	// closed beats everything else.
	both := errors.Join(fs.ErrNotExist, pyexcelize.ErrClosed)
	if got := pyexcelize.ErrorKind(both); got != pyexcelize.KindClosed {
		t.Fatalf("got %q, want closed", got)
	}
	// sheet errors beat file-not-found.
	both = errors.Join(fs.ErrNotExist, &pyexcelize.SheetNotFoundError{Name: "s"})
	if got := pyexcelize.ErrorKind(both); got != pyexcelize.KindSheetNotFound {
		t.Fatalf("got %q, want sheet_not_found", got)
	}
}

func TestKindConstantValues(t *testing.T) {
	t.Parallel()
	want := map[string]string{
		pyexcelize.KindClosed:        "closed",
		pyexcelize.KindSheetNotFound: "sheet_not_found",
		pyexcelize.KindInvalidCell:   "invalid_cell",
		pyexcelize.KindFileNotFound:  "file_not_found",
		pyexcelize.KindFormat:        "format",
		pyexcelize.KindPassword:      "password",
		pyexcelize.KindOther:         "error",
	}
	seen := map[string]bool{}
	for got, w := range want {
		if got != w {
			t.Errorf("constant = %q, want %q", got, w)
		}
		if seen[got] {
			t.Errorf("duplicate kind %q", got)
		}
		seen[got] = true
	}
}

func TestErrorMessages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"closed", pyexcelize.ErrClosed, "pyexcelize: workbook is closed"},
		{"password required", pyexcelize.ErrPasswordRequired, "pyexcelize: workbook is encrypted, a password is required"},
		{"wrong password", pyexcelize.ErrWrongPassword, "pyexcelize: incorrect workbook password"},
		{"sheet", &pyexcelize.SheetNotFoundError{Name: "Data"}, `pyexcelize: sheet "Data" does not exist`},
		{"sheet empty", &pyexcelize.SheetNotFoundError{}, `pyexcelize: sheet "" does not exist`},
		{"sheet quoting", &pyexcelize.SheetNotFoundError{Name: `a"b`}, `pyexcelize: sheet "a\"b" does not exist`},
		{"cell", &pyexcelize.InvalidCellError{Ref: "1A"}, `pyexcelize: invalid cell reference "1A"`},
		{"cell empty", &pyexcelize.InvalidCellError{}, `pyexcelize: invalid cell reference ""`},
		{"cell unicode", &pyexcelize.InvalidCellError{Ref: "Ä1"}, `pyexcelize: invalid cell reference "Ä1"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.err.Error(); got != tc.want {
				t.Fatalf("Error() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestErrorsAsAndIs(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))

	t.Run("sheet", func(t *testing.T) {
		t.Parallel()
		_, err := wb.Rows("Missing")
		var se *pyexcelize.SheetNotFoundError
		if !errors.As(err, &se) || se.Name != "Missing" {
			t.Fatalf("errors.As failed: %#v", err)
		}
	})
	t.Run("cell", func(t *testing.T) {
		t.Parallel()
		_, err := wb.CellValue("Sheet1", "bad!")
		var ce *pyexcelize.InvalidCellError
		if !errors.As(err, &ce) || ce.Ref != "bad!" {
			t.Fatalf("errors.As failed: %#v", err)
		}
	})
	t.Run("closed", func(t *testing.T) {
		t.Parallel()
		var nilWB *pyexcelize.Workbook
		_, err := nilWB.Rows("Sheet1")
		if !errors.Is(err, pyexcelize.ErrClosed) {
			t.Fatalf("errors.Is failed: %v", err)
		}
	})
}

func TestErrorsFromOpen(t *testing.T) {
	t.Parallel()
	_, err := pyexcelize.Open("/definitely/not/here.xlsx")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected wrapped fs.ErrNotExist, got %v", err)
	}
	if !strings.Contains(err.Error(), "/definitely/not/here.xlsx") {
		t.Fatalf("error should mention path: %v", err)
	}
}

package pyexcelize_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/xuri/excelize/v2"

	pyexcelize "github.com/adhadse/py-excelize"
	"github.com/adhadse/py-excelize/internal/fixtures"
)

// ---- helpers ---------------------------------------------------------------

func itoa(i int) string { return strconv.Itoa(i) }

func mustT(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
}

func isKind(err error, kind string) bool { return pyexcelize.ErrorKind(err) == kind }

func fileBytes(tb testing.TB, f *excelize.File) []byte {
	tb.Helper()
	defer f.Close()
	buf, err := f.WriteToBuffer()
	mustT(tb, err)
	return buf.Bytes()
}

func cellsBytes(tb testing.TB) []byte    { return fileBytes(tb, fixtures.Cells()) }
func controlsBytes(tb testing.TB) []byte { return fileBytes(tb, fixtures.Controls()) }
func emptyBytes(tb testing.TB) []byte    { return fileBytes(tb, fixtures.Empty()) }

var (
	fixturesOnce sync.Once
	fixturePaths map[string]string
	fixturesErr  error
)

// writeFixtures writes every fixture once per test binary run.
func writeFixtures(tb testing.TB) map[string]string {
	tb.Helper()
	fixturesOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pyexcelize-fixtures-*")
		if err != nil {
			fixturesErr = err
			return
		}
		fixturePaths, fixturesErr = fixtures.WriteAll(dir)
	})
	mustT(tb, fixturesErr)
	return fixturePaths
}

func TestMain(m *testing.M) {
	code := m.Run()
	for _, p := range fixturePaths {
		_ = os.RemoveAll(filepath.Dir(p))
		break
	}
	os.Exit(code)
}

func openFixture(tb testing.TB, data []byte, opts ...pyexcelize.Option) *pyexcelize.Workbook {
	tb.Helper()
	wb, err := pyexcelize.OpenBytes(data, opts...)
	mustT(tb, err)
	tb.Cleanup(func() { _ = wb.Close() })
	return wb
}

func readFile(tb testing.TB, path string) []byte {
	tb.Helper()
	b, err := os.ReadFile(path)
	mustT(tb, err)
	return b
}

// corruptSheetXML rewrites the zip so that the named part is not valid XML.
func corruptSheetXML(tb testing.TB, data []byte, part string, content ...string) []byte {
	tb.Helper()
	body := "<worksheet><sheetData><row><c"
	if len(content) > 0 {
		body = content[0]
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	mustT(tb, err)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	found := false
	for _, zf := range zr.File {
		w, err := zw.Create(zf.Name)
		mustT(tb, err)
		if zf.Name == part {
			found = true
			_, err = w.Write([]byte(body))
			mustT(tb, err)
			continue
		}
		rc, err := zf.Open()
		mustT(tb, err)
		_, err = io.Copy(w, rc)
		mustT(tb, err)
		mustT(tb, rc.Close())
	}
	if !found {
		tb.Fatalf("part %s not in archive", part)
	}
	mustT(tb, zw.Close())
	return out.Bytes()
}

func rewriteZipPart(tb testing.TB, data []byte, part string, edit func([]byte) []byte) []byte {
	tb.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	mustT(tb, err)
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, zf := range zr.File {
		w, err := zw.Create(zf.Name)
		mustT(tb, err)
		rc, err := zf.Open()
		mustT(tb, err)
		b, err := io.ReadAll(rc)
		mustT(tb, err)
		mustT(tb, rc.Close())
		if zf.Name == part {
			b = edit(b)
		}
		_, err = w.Write(b)
		mustT(tb, err)
	}
	mustT(tb, zw.Close())
	return out.Bytes()
}

// ---- Open* -----------------------------------------------------------------

func TestOpenAllFixtureFiles(t *testing.T) {
	t.Parallel()
	files := writeFixtures(t)
	tests := []struct {
		file   string
		opts   []pyexcelize.Option
		sheets []string
	}{
		{fixtures.FileEmpty, nil, []string{"Sheet1"}},
		{fixtures.FileCells, nil, []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
		{fixtures.FileControls, nil, []string{"Sheet 1", "Plain"}},
		{fixtures.FileEncrypted, []pyexcelize.Option{pyexcelize.WithPassword(fixtures.Password)}, []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()
			p, ok := files[tc.file]
			if !ok {
				t.Fatalf("fixture %s not written", tc.file)
			}
			wb, err := pyexcelize.Open(p, tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer wb.Close()
			if wb.Closed() {
				t.Fatal("freshly opened workbook reports closed")
			}
			if got := wb.SheetNames(); !reflect.DeepEqual(got, tc.sheets) {
				t.Fatalf("SheetNames = %v, want %v", got, tc.sheets)
			}

			// same data through OpenBytes and OpenReader
			data := readFile(t, p)
			wb2, err := pyexcelize.OpenBytes(data, tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer wb2.Close()
			wb3, err := pyexcelize.OpenReader(bytes.NewReader(data), tc.opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer wb3.Close()
			if !reflect.DeepEqual(wb2.SheetNames(), tc.sheets) || !reflect.DeepEqual(wb3.SheetNames(), tc.sheets) {
				t.Fatalf("OpenBytes/OpenReader sheets differ: %v %v", wb2.SheetNames(), wb3.SheetNames())
			}
		})
	}
}

func TestOpenPasswordFlows(t *testing.T) {
	t.Parallel()
	files := writeFixtures(t)
	path := files[fixtures.FileEncrypted]
	data := readFile(t, path)

	opener := map[string]func(opts ...pyexcelize.Option) (*pyexcelize.Workbook, error){
		"Open":      func(o ...pyexcelize.Option) (*pyexcelize.Workbook, error) { return pyexcelize.Open(path, o...) },
		"OpenBytes": func(o ...pyexcelize.Option) (*pyexcelize.Workbook, error) { return pyexcelize.OpenBytes(data, o...) },
		"OpenReader": func(o ...pyexcelize.Option) (*pyexcelize.Workbook, error) {
			return pyexcelize.OpenReader(bytes.NewReader(data), o...)
		},
	}
	cases := []struct {
		name    string
		opts    []pyexcelize.Option
		wantErr error
	}{
		{"missing", nil, pyexcelize.ErrPasswordRequired},
		{"empty password", []pyexcelize.Option{pyexcelize.WithPassword("")}, pyexcelize.ErrPasswordRequired},
		{"wrong", []pyexcelize.Option{pyexcelize.WithPassword("nope")}, pyexcelize.ErrWrongPassword},
		{"wrong case", []pyexcelize.Option{pyexcelize.WithPassword(strings.ToUpper(fixtures.Password))}, pyexcelize.ErrWrongPassword},
		{"right", []pyexcelize.Option{pyexcelize.WithPassword(fixtures.Password)}, nil},
		{"last password wins", []pyexcelize.Option{pyexcelize.WithPassword("bad"), pyexcelize.WithPassword(fixtures.Password)}, nil},
	}
	for name, open := range opener {
		for _, tc := range cases {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				t.Parallel()
				wb, err := open(tc.opts...)
				if tc.wantErr == nil {
					if err != nil {
						t.Fatal(err)
					}
					defer wb.Close()
					v, err := wb.CellValue("Sheet1", "A2")
					if err != nil || v != "apple" {
						t.Fatalf("A2 = %q, %v", v, err)
					}
					return
				}
				if wb != nil {
					t.Fatal("workbook must be nil on error")
				}
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if !isKind(err, pyexcelize.KindPassword) {
					t.Fatalf("kind = %q", pyexcelize.ErrorKind(err))
				}
			})
		}
	}
}

func TestPasswordOnUnencryptedWorkbookIsHarmless(t *testing.T) {
	t.Parallel()
	wb, err := pyexcelize.OpenBytes(cellsBytes(t), pyexcelize.WithPassword("whatever"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := wb.CellValue("Sheet1", "A1"); v != "name" {
		t.Fatalf("A1 = %q", v)
	}
}

func TestOpenFailures(t *testing.T) {
	t.Parallel()
	valid := cellsBytes(t)
	dir := t.TempDir()
	garbagePath := filepath.Join(dir, "garbage.xlsx")
	mustT(t, os.WriteFile(garbagePath, []byte("this is not a zip file at all"), 0o600))
	emptyPath := filepath.Join(dir, "empty.bin")
	mustT(t, os.WriteFile(emptyPath, nil, 0o600))

	t.Run("bytes", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name string
			data []byte
		}{
			{"nil", nil},
			{"empty", []byte{}},
			{"one byte", []byte{0x50}},
			{"garbage text", []byte("hello, definitely not a spreadsheet")},
			{"zip magic only", []byte("PK\x03\x04")},
			{"truncated half", valid[:len(valid)/2]},
			{"truncated minus tail", valid[:len(valid)-10]},
			{"truncated head only", valid[:30]},
			{"zeros", make([]byte, 4096)},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				wb, err := pyexcelize.OpenBytes(tc.data)
				if wb != nil {
					wb.Close()
					t.Fatal("expected nil workbook")
				}
				if !isKind(err, pyexcelize.KindFormat) {
					t.Fatalf("kind = %q, err = %v", pyexcelize.ErrorKind(err), err)
				}
				if !strings.HasPrefix(err.Error(), "pyexcelize: read workbook:") {
					t.Fatalf("missing context: %v", err)
				}
			})
		}
	})

	t.Run("valid zip but not a workbook", func(t *testing.T) {
		t.Parallel()
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		w, _ := zw.Create("hello.txt")
		_, _ = w.Write([]byte("hi"))
		mustT(t, zw.Close())
		wb, err := pyexcelize.OpenBytes(buf.Bytes())
		if err == nil {
			wb.Close()
			t.Fatal("expected error")
		}
		if wb != nil {
			t.Fatal("expected nil workbook")
		}
	})

	t.Run("file missing", func(t *testing.T) {
		t.Parallel()
		wb, err := pyexcelize.Open(filepath.Join(dir, "missing.xlsx"))
		if wb != nil || !isKind(err, pyexcelize.KindFileNotFound) {
			t.Fatalf("wb=%v err=%v", wb, err)
		}
	})
	t.Run("path is a directory", func(t *testing.T) {
		t.Parallel()
		wb, err := pyexcelize.Open(dir)
		if wb != nil || err == nil {
			t.Fatalf("wb=%v err=%v", wb, err)
		}
		if isKind(err, pyexcelize.KindFileNotFound) {
			t.Fatalf("directory must not be file_not_found: %v", err)
		}
	})
	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		if _, err := pyexcelize.Open(""); !isKind(err, pyexcelize.KindFileNotFound) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("garbage file", func(t *testing.T) {
		t.Parallel()
		wb, err := pyexcelize.Open(garbagePath)
		if wb != nil || !isKind(err, pyexcelize.KindFormat) {
			t.Fatalf("wb=%v err=%v", wb, err)
		}
		if !strings.Contains(err.Error(), garbagePath) {
			t.Fatalf("error should mention path: %v", err)
		}
	})
	t.Run("empty file", func(t *testing.T) {
		t.Parallel()
		if _, err := pyexcelize.Open(emptyPath); !isKind(err, pyexcelize.KindFormat) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("reader", func(t *testing.T) {
		t.Parallel()
		boom := errors.New("disk on fire")
		cases := []struct {
			name string
			r    io.Reader
			kind string
			is   error
		}{
			{"error reader", iotest.ErrReader(boom), pyexcelize.KindOther, boom},
			{"fails midway", io.MultiReader(bytes.NewReader(valid[:100]), iotest.ErrReader(boom)), pyexcelize.KindOther, boom},
			{"timeout", iotest.TimeoutReader(bytes.NewReader(valid)), pyexcelize.KindOther, iotest.ErrTimeout},
			{"empty", strings.NewReader(""), pyexcelize.KindFormat, nil},
			{"garbage", strings.NewReader("xyz"), pyexcelize.KindFormat, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				wb, err := pyexcelize.OpenReader(tc.r)
				if wb != nil || err == nil {
					t.Fatalf("wb=%v err=%v", wb, err)
				}
				if !isKind(err, tc.kind) {
					t.Fatalf("kind = %q, err = %v", pyexcelize.ErrorKind(err), err)
				}
				if tc.is != nil && !errors.Is(err, tc.is) {
					t.Fatalf("errors.Is(%v, %v) = false", err, tc.is)
				}
			})
		}
	})

	t.Run("reader one byte at a time", func(t *testing.T) {
		t.Parallel()
		wb, err := pyexcelize.OpenReader(iotest.OneByteReader(bytes.NewReader(valid)))
		if err != nil {
			t.Fatal(err)
		}
		defer wb.Close()
		if len(wb.SheetNames()) != 3 {
			t.Fatalf("sheets = %v", wb.SheetNames())
		}
	})
}

func TestOpenBytesDoesNotAliasInput(t *testing.T) {
	t.Parallel()
	data := cellsBytes(t)
	wb := openFixture(t, data)
	// Scribbling over the input afterwards must not break already-parsed data.
	v, err := wb.CellValue("Sheet1", "A2")
	if err != nil || v != "apple" {
		t.Fatalf("A2 = %q, %v", v, err)
	}
}

// ---- WithRawCellValues -----------------------------------------------------

func TestWithRawCellValues(t *testing.T) {
	t.Parallel()
	f := excelize.NewFile()
	style, err := f.NewStyle(&excelize.Style{NumFmt: 2}) // 0.00
	mustT(t, err)
	mustT(t, f.SetCellValue("Sheet1", "A1", 1.5))
	mustT(t, f.SetCellValue("Sheet1", "A2", 3))
	mustT(t, f.SetCellStyle("Sheet1", "A1", "A2", style))
	data := fileBytes(t, f)

	formatted := openFixture(t, data)
	raw := openFixture(t, data, pyexcelize.WithRawCellValues())

	tests := []struct {
		cell, formatted, raw string
	}{
		{"A1", "1.50", "1.5"},
		{"A2", "3.00", "3"},
	}
	for _, tc := range tests {
		t.Run(tc.cell, func(t *testing.T) {
			got, err := formatted.CellValue("Sheet1", tc.cell)
			if err != nil || got != tc.formatted {
				t.Errorf("formatted = %q, %v; want %q", got, err, tc.formatted)
			}
			got, err = raw.CellValue("Sheet1", tc.cell)
			if err != nil || got != tc.raw {
				t.Errorf("raw = %q, %v; want %q", got, err, tc.raw)
			}
		})
	}

	rows, err := raw.Rows("Sheet1")
	mustT(t, err)
	if want := [][]string{{"1.5"}, {"3"}}; !reflect.DeepEqual(rows, want) {
		t.Errorf("raw rows = %v, want %v", rows, want)
	}
}

// ---- Close / Closed / nil --------------------------------------------------

func TestCloseSemantics(t *testing.T) {
	t.Parallel()

	t.Run("double close", func(t *testing.T) {
		wb := openFixture(t, cellsBytes(t))
		if wb.Closed() {
			t.Fatal("closed before Close")
		}
		for i := 0; i < 3; i++ {
			if err := wb.Close(); err != nil {
				t.Fatalf("Close #%d: %v", i+1, err)
			}
			if !wb.Closed() {
				t.Fatalf("not closed after Close #%d", i+1)
			}
		}
	})

	t.Run("nil receiver", func(t *testing.T) {
		var wb *pyexcelize.Workbook
		if err := wb.Close(); err != nil {
			t.Fatal(err)
		}
		if !wb.Closed() {
			t.Fatal("nil workbook should be closed")
		}
	})

	t.Run("zero value", func(t *testing.T) {
		var wb pyexcelize.Workbook
		if !wb.Closed() || wb.Close() != nil {
			t.Fatal("zero Workbook should behave as closed")
		}
	})

	t.Run("FromFile", func(t *testing.T) {
		f := fixtures.Cells()
		wb := pyexcelize.FromFile(f)
		if wb.Closed() {
			t.Fatal("FromFile workbook is closed")
		}
		if v, _ := wb.CellValue("Sheet1", "A1"); v != "name" {
			t.Fatalf("A1 = %q", v)
		}
		mustT(t, wb.Close())
		if !wb.Closed() {
			t.Fatal("not closed")
		}
	})

	t.Run("FromFile nil", func(t *testing.T) {
		wb := pyexcelize.FromFile(nil)
		if wb == nil || !wb.Closed() {
			t.Fatalf("FromFile(nil) = %v", wb)
		}
		if _, err := wb.Rows("Sheet1"); !isKind(err, pyexcelize.KindClosed) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestMethodsOnClosedAndNilWorkbook(t *testing.T) {
	t.Parallel()
	closed := openFixture(t, cellsBytes(t))
	mustT(t, closed.Close())
	var nilWB *pyexcelize.Workbook

	for name, wb := range map[string]*pyexcelize.Workbook{"closed": closed, "nil": nilWB} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if names := wb.SheetNames(); names == nil || len(names) != 0 {
				t.Errorf("SheetNames = %#v, want empty non-nil", names)
			}
			if wb.HasSheet("Sheet1") {
				t.Error("HasSheet = true on closed workbook")
			}
			if wb.HasSheet("") {
				t.Error("HasSheet(\"\") = true on closed workbook")
			}

			calls := map[string]func() error{
				"Dimension":       func() error { _, err := wb.Dimension("Sheet1"); return err },
				"CellValue":       func() error { _, err := wb.CellValue("Sheet1", "A1"); return err },
				"CellFormula":     func() error { _, err := wb.CellFormula("Sheet1", "D2"); return err },
				"Rows":            func() error { _, err := wb.Rows("Sheet1"); return err },
				"MergedCells":     func() error { _, err := wb.MergedCells("Sheet1"); return err },
				"FormControls":    func() error { _, err := wb.FormControls("Sheet1"); return err },
				"AllFormControls": func() error { _, err := wb.AllFormControls(); return err },
				// closed beats every other validation error
				"CellValue bad sheet+cell": func() error { _, err := wb.CellValue("Nope", "!!"); return err },
			}
			for cname, call := range calls {
				if err := call(); !errors.Is(err, pyexcelize.ErrClosed) {
					t.Errorf("%s: err = %v, want ErrClosed", cname, err)
				} else if !isKind(err, pyexcelize.KindClosed) {
					t.Errorf("%s: kind = %q", cname, pyexcelize.ErrorKind(err))
				}
			}

			// return values are zero on error
			if s, _ := wb.CellValue("Sheet1", "A1"); s != "" {
				t.Errorf("CellValue = %q", s)
			}
			if r, _ := wb.Rows("Sheet1"); r != nil {
				t.Errorf("Rows = %#v", r)
			}
			if m, _ := wb.MergedCells("Sheet1"); m != nil {
				t.Errorf("MergedCells = %#v", m)
			}
		})
	}
}

// ---- Sheets ----------------------------------------------------------------

func TestSheetNamesAndHasSheet(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))

	if got, want := wb.SheetNames(), []string{"Sheet1", "Sheet 2", "Ünï Çødé"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("SheetNames = %v, want %v", got, want)
	}
	// callers may mutate the returned slice without affecting the workbook
	names := wb.SheetNames()
	names[0] = "mutated"
	if wb.SheetNames()[0] != "Sheet1" {
		t.Error("SheetNames returned shared backing storage")
	}

	tests := []struct {
		name string
		want bool
	}{
		{"Sheet1", true},
		{"sheet1", true},
		{"SHEET1", true},
		{"Sheet 2", true},
		{"sheet 2", true},
		{"Ünï Çødé", true},
		{"ÜNÏ ÇØDÉ", true},
		{"ünï çødé", true},
		{"Sheet2", false},
		{"Sheet 3", false},
		{"", false},
		{" Sheet1", false},
		{"Sheet1 ", false},
		{"Sheet1\x00", false},
		{"Sheet", false},
		{"Sheet1:Sheet2", false},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%q", tc.name), func(t *testing.T) {
			if got := wb.HasSheet(tc.name); got != tc.want {
				t.Fatalf("HasSheet(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

func TestSheetNamesEmptyWorkbook(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, emptyBytes(t))
	if got := wb.SheetNames(); !reflect.DeepEqual(got, []string{"Sheet1"}) {
		t.Fatalf("got %v", got)
	}
}

func TestSheetNamesNilListFromExcelize(t *testing.T) {
	t.Parallel()
	// A workbook whose only sheet was deleted has no sheet list.
	wb := pyexcelize.FromFile(&excelize.File{})
	got := wb.SheetNames()
	if got == nil || len(got) != 0 {
		t.Fatalf("SheetNames = %#v, want empty non-nil", got)
	}
}

func TestMissingSheetEverywhere(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))
	for _, sheet := range []string{"", "Nope", "Sheet 3", "Sheet1 ", "Sheet1\n"} {
		t.Run(fmt.Sprintf("%q", sheet), func(t *testing.T) {
			t.Parallel()
			calls := map[string]func() error{
				"Dimension":    func() error { _, err := wb.Dimension(sheet); return err },
				"CellValue":    func() error { _, err := wb.CellValue(sheet, "A1"); return err },
				"CellFormula":  func() error { _, err := wb.CellFormula(sheet, "A1"); return err },
				"Rows":         func() error { _, err := wb.Rows(sheet); return err },
				"MergedCells":  func() error { _, err := wb.MergedCells(sheet); return err },
				"FormControls": func() error { _, err := wb.FormControls(sheet); return err },
				// a missing sheet is reported before a bad cell
				"CellValue bad cell": func() error { _, err := wb.CellValue(sheet, "###"); return err },
			}
			for name, call := range calls {
				err := call()
				var se *pyexcelize.SheetNotFoundError
				if !errors.As(err, &se) {
					t.Errorf("%s: err = %v, want SheetNotFoundError", name, err)
					continue
				}
				if se.Name != sheet {
					t.Errorf("%s: Name = %q, want %q", name, se.Name, sheet)
				}
				if !isKind(err, pyexcelize.KindSheetNotFound) {
					t.Errorf("%s: kind = %q", name, pyexcelize.ErrorKind(err))
				}
			}
		})
	}
}

// ---- Cells -----------------------------------------------------------------

func TestCellRefValidation(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))

	valid := []string{"A1", "a1", "b2", "Z9", "AA10", "aA10", "XFD1", "xfd1", "XFD1048576", "A1048576", "A9999", "D2"}
	invalid := []string{
		"", "1A", "A0", "A", "1", "AAAA1", "A-1", " A1", "A1 ", "A1:B2", "$A$1", "A$1", "$A1",
		"Ä1", "Ａ1", "A１", "日本1", "A1\n", "\tA1", "A 1", "A01", "A+1", "A1.0", "A1048577",
		"ZZZ1", "XFE1", "ZZZ1048576", "A12345678", "AAA", "--", "A1B", "\x00", "A1\x00",
	}

	for _, ref := range valid {
		t.Run("valid/"+ref, func(t *testing.T) {
			t.Parallel()
			if _, err := wb.CellValue("Sheet1", ref); err != nil {
				t.Fatalf("CellValue: unexpected error %v", err)
			}
			if _, err := wb.CellFormula("Sheet1", ref); err != nil {
				t.Fatalf("CellFormula: unexpected error %v", err)
			}
		})
	}
	for _, ref := range invalid {
		t.Run(fmt.Sprintf("invalid/%q", ref), func(t *testing.T) {
			t.Parallel()
			for name, call := range map[string]func() (string, error){
				"CellValue":   func() (string, error) { return wb.CellValue("Sheet1", ref) },
				"CellFormula": func() (string, error) { return wb.CellFormula("Sheet1", ref) },
			} {
				v, err := call()
				var ce *pyexcelize.InvalidCellError
				if !errors.As(err, &ce) {
					t.Fatalf("%s: err = %v, want InvalidCellError", name, err)
				}
				if ce.Ref != ref {
					t.Errorf("%s: Ref = %q, want %q", name, ce.Ref, ref)
				}
				if v != "" {
					t.Errorf("%s: value = %q on error", name, v)
				}
				if !isKind(err, pyexcelize.KindInvalidCell) {
					t.Errorf("%s: kind = %q", name, pyexcelize.ErrorKind(err))
				}
			}
		})
	}
}

func TestCellValues(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))
	tests := []struct {
		sheet, cell, want string
	}{
		{"Sheet1", "A1", "name"},
		{"Sheet1", "a1", "name"},
		{"Sheet1", "B1", "qty"},
		{"Sheet1", "C1", "price"},
		{"Sheet1", "A2", "apple"},
		{"Sheet1", "B2", "3"},
		{"Sheet1", "C2", "1.5"},
		{"Sheet1", "D2", ""},
		{"Sheet1", "F2", "TRUE"},
		{"Sheet1", "A3", "pear"},
		{"Sheet1", "B3", "10"},
		{"Sheet1", "C3", "0.25"},
		{"Sheet1", "A5", "gap row"},
		{"Sheet1", "A7", "merged title"},
		{"Sheet1", "B7", "merged title"},
		{"Sheet1", "A4", ""},
		{"Sheet1", "ZZ999", ""},
		{"Sheet1", "XFD1048576", ""},
		{"sheet1", "A1", "name"},
		{"SHEET1", "A2", "apple"},
		{"Sheet 2", "B2", "spaced"},
		{"sheet 2", "B2", "spaced"},
		{"Sheet 2", "A1", ""},
		{"Ünï Çødé", "A1", "héllo wörld ✓"},
		{"ünï çødé", "A1", "héllo wörld ✓"},
	}
	for _, tc := range tests {
		t.Run(tc.sheet+"!"+tc.cell, func(t *testing.T) {
			t.Parallel()
			got, err := wb.CellValue(tc.sheet, tc.cell)
			if err != nil || got != tc.want {
				t.Fatalf("CellValue = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestCellFormulas(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))
	tests := []struct {
		sheet, cell, want string
	}{
		{"Sheet1", "D2", "B2*C2"},
		{"sheet1", "d2", "B2*C2"},
		{"Sheet1", "D3", ""},
		{"Sheet1", "A1", ""},
		{"Sheet1", "B2", ""},
		{"Sheet 2", "B2", ""},
		{"Ünï Çødé", "A1", ""},
	}
	for _, tc := range tests {
		t.Run(tc.sheet+"!"+tc.cell, func(t *testing.T) {
			t.Parallel()
			got, err := wb.CellFormula(tc.sheet, tc.cell)
			if err != nil || got != tc.want {
				t.Fatalf("CellFormula = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestCellAccessOnCorruptSheet(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, corruptSheetXML(t, cellsBytes(t), "xl/worksheets/sheet1.xml"))

	// The corrupt part must surface as a plain error, never a panic.
	if _, err := wb.CellValue("Sheet1", "A1"); err == nil {
		t.Error("CellValue: expected error")
	}
	if _, err := wb.CellFormula("Sheet1", "A1"); err == nil {
		t.Error("CellFormula: expected error")
	}
	if _, err := wb.MergedCells("Sheet1"); err == nil {
		t.Error("MergedCells: expected error")
	}
	// other sheets are still readable
	if v, err := wb.CellValue("Sheet 2", "B2"); err != nil || v != "spaced" {
		t.Errorf("Sheet 2!B2 = %q, %v", v, err)
	}
}

func TestRowsAndDimensionErrorFromExcelize(t *testing.T) {
	t.Parallel()
	// excelize refuses sheet names longer than 31 characters even though a
	// hand-edited workbook.xml may contain them.
	long := strings.Repeat("x", 40)
	data := rewriteZipPart(t, cellsBytes(t), "xl/workbook.xml", func(b []byte) []byte {
		return bytes.Replace(b, []byte(`name="Sheet 2"`), []byte(`name="`+long+`"`), 1)
	})
	wb := openFixture(t, data)
	if !wb.HasSheet(long) {
		t.Fatalf("sheets = %v", wb.SheetNames())
	}
	if _, err := wb.Rows(long); err == nil {
		t.Error("Rows: expected error")
	}
	if _, err := wb.Dimension(long); err == nil {
		t.Error("Dimension: expected error")
	}
}

// ---- Rows / Dimension / Merged --------------------------------------------

func TestRows(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))

	tests := []struct {
		sheet string
		want  [][]string
	}{
		{"Sheet1", [][]string{
			{"name", "qty", "price"},
			{"apple", "3", "1.5", "", "", "TRUE"},
			{"pear", "10", "0.25"},
			{},
			{"gap row"},
			{},
			{"merged title"},
		}},
		{"sheet1", nil}, // case-insensitive, compared below to Sheet1
		{"Sheet 2", [][]string{{}, {"", "spaced"}}},
		{"Ünï Çødé", [][]string{{"héllo wörld ✓"}}},
	}
	base, err := wb.Rows("Sheet1")
	mustT(t, err)
	for _, tc := range tests {
		t.Run(tc.sheet, func(t *testing.T) {
			got, err := wb.Rows(tc.sheet)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.want
			if want == nil {
				want = base
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Rows = %#v\nwant   %#v", got, want)
			}
		})
	}
}

func TestRowsEmptySheetNonNil(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, emptyBytes(t))
	rows, err := wb.Rows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	if rows == nil || len(rows) != 0 {
		t.Fatalf("Rows = %#v, want empty non-nil", rows)
	}
}

func TestDimension(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))
	tests := []struct {
		sheet, want string
	}{
		{"Sheet1", "A1:F7"},
		{"sheet1", "A1:F7"},
		{"Sheet 2", "B2"},
		{"Ünï Çødé", "A1"},
	}
	for _, tc := range tests {
		t.Run(tc.sheet, func(t *testing.T) {
			got, err := wb.Dimension(tc.sheet)
			if err != nil || got != tc.want {
				t.Fatalf("Dimension = %q, %v; want %q", got, err, tc.want)
			}
		})
	}

	t.Run("empty sheet", func(t *testing.T) {
		e := openFixture(t, emptyBytes(t))
		defer e.Close()
		got, err := e.Dimension("Sheet1")
		if err != nil || got != "" {
			t.Fatalf("Dimension = %q, %v", got, err)
		}
	})
	t.Run("plain sheet of controls workbook", func(t *testing.T) {
		c := openFixture(t, controlsBytes(t))
		defer c.Close()
		got, err := c.Dimension("Plain")
		if err != nil || got != "" {
			t.Fatalf("Dimension = %q, %v", got, err)
		}
	})
}

func TestMergedCells(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))

	got, err := wb.MergedCells("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	byRange := map[string]pyexcelize.MergedCell{}
	for _, m := range got {
		byRange[m.Range] = m
	}
	want := map[string]pyexcelize.MergedCell{
		"A7:C7": {Range: "A7:C7", Start: "A7", End: "C7", Value: "merged title"},
		"E1:E3": {Range: "E1:E3", Start: "E1", End: "E3", Value: ""},
	}
	if len(got) != len(want) || !reflect.DeepEqual(byRange, want) {
		t.Fatalf("MergedCells = %+v, want %+v", got, want)
	}

	for _, sheet := range []string{"Sheet 2", "Ünï Çødé", "sheet 2"} {
		m, err := wb.MergedCells(sheet)
		if err != nil {
			t.Fatal(err)
		}
		if m == nil || len(m) != 0 {
			t.Errorf("%s: MergedCells = %#v, want empty non-nil", sheet, m)
		}
	}
}

func TestReadsAreRepeatable(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, cellsBytes(t))
	first, err := wb.Rows("Sheet1")
	mustT(t, err)
	for i := 0; i < 5; i++ {
		again, err := wb.Rows("Sheet1")
		mustT(t, err)
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("iteration %d differs", i)
		}
	}
}

func TestUseAfterCloseDoesNotAffectOtherWorkbooks(t *testing.T) {
	t.Parallel()
	data := cellsBytes(t)
	a := openFixture(t, data)
	b := openFixture(t, data)
	mustT(t, a.Close())
	if v, err := b.CellValue("Sheet1", "A1"); err != nil || v != "name" {
		t.Fatalf("b.A1 = %q, %v", v, err)
	}
}

// ---- Benchmarks and fuzzing -----------------------------------------------

func BenchmarkOpenBytes(b *testing.B) {
	data := cellsBytes(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb, err := pyexcelize.OpenBytes(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = wb.Close()
	}
}

func BenchmarkRows(b *testing.B) {
	wb := openFixture(b, cellsBytes(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wb.Rows("Sheet1"); err != nil {
			b.Fatal(err)
		}
	}
}

func FuzzOpenBytes(f *testing.F) {
	f.Add([]byte(nil))
	f.Add([]byte("PK\x03\x04"))
	f.Add([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1})
	f.Add([]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1, 0, 0, 0, 0})
	seed := cellsBytes(f)
	f.Add(seed)
	f.Add(seed[:len(seed)/2])
	f.Add(controlsBytes(f))

	f.Fuzz(func(t *testing.T, data []byte) {
		for _, opts := range [][]pyexcelize.Option{nil, {pyexcelize.WithPassword("x")}} {
			wb, err := pyexcelize.OpenBytes(data, opts...)
			if err != nil {
				if wb != nil {
					t.Fatal("non-nil workbook with error")
				}
				if pyexcelize.ErrorKind(err) == "" {
					t.Fatal("empty kind for non-nil error")
				}
				continue
			}
			_ = wb.SheetNames()
			for _, name := range wb.SheetNames() {
				_, _ = wb.Rows(name)
				_, _ = wb.FormControls(name)
				_, _ = wb.MergedCells(name)
			}
			_ = wb.Close()
		}
	})
}

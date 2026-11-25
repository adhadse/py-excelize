package fixtures

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	pyexcelize "github.com/adhadse/py-excelize"
	"github.com/xuri/excelize/v2"
)

func TestConstants(t *testing.T) {
	if Password == "" {
		t.Fatal("Password must not be empty")
	}
	names := []string{FileEmpty, FileCells, FileControls, FileEncrypted}
	seen := map[string]bool{}
	for _, n := range names {
		if !strings.HasSuffix(n, ".xlsx") {
			t.Errorf("%q lacks .xlsx suffix", n)
		}
		if seen[n] {
			t.Errorf("duplicate file name %q", n)
		}
		seen[n] = true
	}
}

func TestEmpty(t *testing.T) {
	f := Empty()
	defer f.Close()
	if got := f.GetSheetList(); !reflect.DeepEqual(got, []string{"Sheet1"}) {
		t.Fatalf("sheets = %v", got)
	}
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		for _, c := range r {
			if c != "" {
				t.Fatalf("unexpected content %q", c)
			}
		}
	}
	// Each call returns an independent workbook.
	g := Empty()
	defer g.Close()
	if f == g {
		t.Fatal("Empty returned a shared instance")
	}
}

func TestCells(t *testing.T) {
	f := Cells()
	defer f.Close()
	if got, want := f.GetSheetList(), []string{"Sheet1", "Sheet 2", "Ünï Çødé"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sheets = %v, want %v", got, want)
	}
	values := []struct{ sheet, cell, want string }{
		{"Sheet1", "A1", "name"},
		{"Sheet1", "B1", "qty"},
		{"Sheet1", "C1", "price"},
		{"Sheet1", "A2", "apple"},
		{"Sheet1", "B2", "3"},
		{"Sheet1", "C2", "1.5"},
		{"Sheet1", "A3", "pear"},
		{"Sheet1", "B3", "10"},
		{"Sheet1", "C3", "0.25"},
		{"Sheet1", "F2", "TRUE"},
		{"Sheet1", "A4", ""},
		{"Sheet1", "A5", "gap row"},
		{"Sheet1", "A7", "merged title"},
		{"Sheet 2", "B2", "spaced"},
		{"Sheet 2", "A1", ""},
		{"Ünï Çødé", "A1", "héllo wörld ✓"},
	}
	for _, v := range values {
		got, err := f.GetCellValue(v.sheet, v.cell)
		if err != nil {
			t.Fatalf("%s!%s: %v", v.sheet, v.cell, err)
		}
		if got != v.want {
			t.Errorf("%s!%s = %q, want %q", v.sheet, v.cell, got, v.want)
		}
	}
	formula, err := f.GetCellFormula("Sheet1", "D2")
	if err != nil || formula != "B2*C2" {
		t.Fatalf("formula = %q, %v", formula, err)
	}
	if formula, _ := f.GetCellFormula("Sheet1", "D3"); formula != "" {
		t.Fatalf("D3 formula = %q", formula)
	}

	merged, err := f.GetMergeCells("Sheet1")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, m := range merged {
		got = append(got, m.GetStartAxis()+":"+m.GetEndAxis())
	}
	sort.Strings(got)
	if want := []string{"A7:C7", "E1:E3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("merged = %v, want %v", got, want)
	}
	for _, name := range []string{"Sheet 2", "Ünï Çødé"} {
		m, err := f.GetMergeCells(name)
		if err != nil || len(m) != 0 {
			t.Fatalf("%s merged = %v, %v", name, m, err)
		}
	}
}

func TestControls(t *testing.T) {
	f := Controls()
	defer f.Close()
	if got, want := f.GetSheetList(), []string{"Sheet 1", "Plain"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("sheets = %v, want %v", got, want)
	}
	cs, err := f.GetFormControls("Sheet 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 10 {
		t.Fatalf("controls = %d, want 10", len(cs))
	}
	byCell := map[string]excelize.FormControl{}
	for _, c := range cs {
		byCell[c.Cell] = c
	}
	wantTypes := map[string]excelize.FormControlType{
		"A1":  excelize.FormControlCheckBox,
		"A3":  excelize.FormControlCheckBox,
		"A5":  excelize.FormControlOptionButton,
		"A7":  excelize.FormControlOptionButton,
		"A9":  excelize.FormControlSpinButton,
		"A11": excelize.FormControlScrollBar,
		"A13": excelize.FormControlGroupBox,
		"A15": excelize.FormControlLabel,
		"A17": excelize.FormControlButton,
		"A19": excelize.FormControlCheckBox,
	}
	for cell, typ := range wantTypes {
		c, ok := byCell[cell]
		if !ok {
			t.Errorf("no control at %s", cell)
			continue
		}
		if c.Type != typ {
			t.Errorf("%s type = %v, want %v", cell, c.Type, typ)
		}
	}
	if c := byCell["A1"]; !c.Checked || c.Text != "Agree" || c.CellLink != "$D$1" {
		t.Errorf("A1 = %+v", c)
	}
	if c := byCell["A3"]; c.Checked || c.Text != "Subscribe" {
		t.Errorf("A3 = %+v", c)
	}
	if c := byCell["A9"]; c.CurrentVal != 7 || c.MinVal != 1 || c.MaxVal != 20 || c.IncChange != 2 {
		t.Errorf("A9 = %+v", c)
	}
	if c := byCell["A11"]; c.CurrentVal != 30 || c.MaxVal != 100 || c.PageChange != 10 || !c.Horizontally {
		t.Errorf("A11 = %+v", c)
	}
	if c := byCell["A17"]; c.Macro != "Button1_Click" || c.Text != "Click" {
		t.Errorf("A17 = %+v", c)
	}
	if c := byCell["A19"]; len(c.Paragraph) != 1 || strings.TrimSpace(c.Text) != "Line one" || strings.TrimSpace(c.Paragraph[0].Text) != "Line two" {
		t.Errorf("A19 = %+v", c)
	}
	plain, err := f.GetFormControls("Plain")
	if err != nil || len(plain) != 0 {
		t.Fatalf("Plain controls = %v, %v", plain, err)
	}
}

func TestWriteAll(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "out")
	paths, err := WriteAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	wantNames := []string{FileCells, FileControls, FileEmpty, FileEncrypted}
	var gotNames []string
	for n := range paths {
		gotNames = append(gotNames, n)
	}
	sort.Strings(gotNames)
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("names = %v, want %v", gotNames, wantNames)
	}
	for name, p := range paths {
		if p != filepath.Join(dir, name) {
			t.Errorf("%s path = %q", name, p)
		}
		fi, err := os.Stat(p)
		if err != nil || fi.Size() == 0 || !fi.Mode().IsRegular() {
			t.Errorf("%s: stat = %v, %v", name, fi, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 4 {
		t.Fatalf("dir entries = %d, %v", len(entries), err)
	}

	tests := []struct {
		file     string
		password string
		sheets   []string
	}{
		{FileEmpty, "", []string{"Sheet1"}},
		{FileCells, "", []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
		{FileControls, "", []string{"Sheet 1", "Plain"}},
		{FileEncrypted, Password, []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			var opts []pyexcelize.Option
			if tc.password != "" {
				opts = append(opts, pyexcelize.WithPassword(tc.password))
			}
			wb, err := pyexcelize.Open(paths[tc.file], opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer wb.Close()
			if got := wb.SheetNames(); !reflect.DeepEqual(got, tc.sheets) {
				t.Fatalf("sheets = %v, want %v", got, tc.sheets)
			}
		})
	}

	t.Run("content survives roundtrip", func(t *testing.T) {
		wb, err := pyexcelize.Open(paths[FileCells])
		if err != nil {
			t.Fatal(err)
		}
		defer wb.Close()
		if v, err := wb.CellValue("Sheet1", "A2"); err != nil || v != "apple" {
			t.Fatalf("A2 = %q, %v", v, err)
		}
		if v, err := wb.CellFormula("Sheet1", "D2"); err != nil || v != "B2*C2" {
			t.Fatalf("D2 = %q, %v", v, err)
		}
		cw, err := pyexcelize.Open(paths[FileControls])
		if err != nil {
			t.Fatal(err)
		}
		defer cw.Close()
		cs, err := cw.AllFormControls()
		if err != nil || len(cs) != 10 {
			t.Fatalf("controls = %d, %v", len(cs), err)
		}
	})

	t.Run("encrypted needs password", func(t *testing.T) {
		if _, err := pyexcelize.Open(paths[FileEncrypted]); !errors.Is(err, pyexcelize.ErrPasswordRequired) {
			t.Fatalf("err = %v", err)
		}
		if _, err := pyexcelize.Open(paths[FileEncrypted], pyexcelize.WithPassword("nope")); !errors.Is(err, pyexcelize.ErrWrongPassword) {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestWriteAllIdempotent(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 3; i++ {
		paths, err := WriteAll(dir)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		if len(paths) != 4 {
			t.Fatalf("run %d: %d paths", i, len(paths))
		}
	}
}

func TestWriteAllErrors(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "regular")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Run("dir under a regular file", func(t *testing.T) {
		paths, err := WriteAll(filepath.Join(file, "sub"))
		if err == nil || paths != nil {
			t.Fatalf("paths = %v, err = %v", paths, err)
		}
	})
	t.Run("dir is a regular file", func(t *testing.T) {
		paths, err := WriteAll(file)
		if err == nil || paths != nil {
			t.Fatalf("paths = %v, err = %v", paths, err)
		}
	})
	t.Run("target file is a directory", func(t *testing.T) {
		dir := filepath.Join(base, "blocked")
		for _, n := range []string{FileEmpty, FileCells, FileControls, FileEncrypted} {
			if err := os.MkdirAll(filepath.Join(dir, n, "child"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		paths, err := WriteAll(dir)
		if err == nil || paths != nil {
			t.Fatalf("paths = %v, err = %v", paths, err)
		}
		if !strings.Contains(err.Error(), "write ") || !strings.Contains(err.Error(), ".xlsx") {
			t.Fatalf("error not wrapped with file name: %v", err)
		}
	})
	t.Run("read-only dir", func(t *testing.T) {
		if runtime.GOOS == "windows" || os.Geteuid() == 0 {
			t.Skip("directory permissions are not enforced here")
		}
		dir := filepath.Join(base, "ro")
		if err := os.Mkdir(dir, 0o500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
		if _, err := WriteAll(dir); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMust(t *testing.T) {
	t.Run("nil does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("unexpected panic: %v", r)
			}
		}()
		must(nil)
	})
	t.Run("error panics with that error", func(t *testing.T) {
		want := errors.New("boom")
		defer func() {
			r := recover()
			err, ok := r.(error)
			if !ok || !errors.Is(err, want) {
				t.Fatalf("recovered %v (%T)", r, r)
			}
		}()
		must(want)
		t.Fatal("must did not panic")
	})
	t.Run("wrapped excelize error", func(t *testing.T) {
		f := excelize.NewFile()
		defer f.Close()
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic from invalid cell")
			}
		}()
		must(f.SetCellValue("Sheet1", "not-a-cell", 1))
	})
}

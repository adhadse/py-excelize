package pyexcelize_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	pyexcelize "github.com/adhadse/py-excelize"
)

func TestControlTypeString(t *testing.T) {
	t.Parallel()
	tests := []struct {
		typ  pyexcelize.ControlType
		want string
	}{
		{pyexcelize.ControlNote, "note"},
		{pyexcelize.ControlButton, "button"},
		{pyexcelize.ControlOptionButton, "option_button"},
		{pyexcelize.ControlSpinButton, "spin_button"},
		{pyexcelize.ControlCheckBox, "check_box"},
		{pyexcelize.ControlGroupBox, "group_box"},
		{pyexcelize.ControlLabel, "label"},
		{pyexcelize.ControlScrollBar, "scroll_bar"},
		{pyexcelize.ControlType(-1), "unknown"},
		{pyexcelize.ControlType(99), "unknown"},
		{pyexcelize.ControlType(1 << 20), "unknown"},
	}
	for _, tc := range tests {
		t.Run(tc.want+"/"+itoa(int(tc.typ)), func(t *testing.T) {
			t.Parallel()
			if got := tc.typ.String(); got != tc.want {
				t.Fatalf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestControlTypeMirrorsExcelize(t *testing.T) {
	t.Parallel()
	pairs := map[pyexcelize.ControlType]excelize.FormControlType{
		pyexcelize.ControlNote:         excelize.FormControlNote,
		pyexcelize.ControlButton:       excelize.FormControlButton,
		pyexcelize.ControlOptionButton: excelize.FormControlOptionButton,
		pyexcelize.ControlSpinButton:   excelize.FormControlSpinButton,
		pyexcelize.ControlCheckBox:     excelize.FormControlCheckBox,
		pyexcelize.ControlGroupBox:     excelize.FormControlGroupBox,
		pyexcelize.ControlLabel:        excelize.FormControlLabel,
		pyexcelize.ControlScrollBar:    excelize.FormControlScrollBar,
	}
	names := map[string]bool{}
	for mine, theirs := range pairs {
		if int(mine) != int(theirs) {
			t.Errorf("%v: %d != excelize %d", mine, mine, theirs)
		}
		if names[mine.String()] {
			t.Errorf("duplicate name %q", mine)
		}
		names[mine.String()] = true
	}
}

// wantControl describes one expected control of the controls fixture.
type wantControl struct {
	typ        pyexcelize.ControlType
	cell       string
	checked    bool
	link       string
	cur, lo    uint
	hi, inc    uint
	page       uint
	horizontal bool
	macro      string
	text       string
	paragraphs []string
}

var fixtureControls = []wantControl{
	{typ: pyexcelize.ControlCheckBox, cell: "A1", checked: true, link: "$D$1", text: "Agree", paragraphs: []string{"Agree"}},
	{typ: pyexcelize.ControlCheckBox, cell: "A3", link: "$D$3", text: "Subscribe", paragraphs: []string{"Subscribe"}},
	{typ: pyexcelize.ControlOptionButton, cell: "A5", checked: true, text: "Option A", paragraphs: []string{"Option A"}},
	{typ: pyexcelize.ControlOptionButton, cell: "A7", text: "Option B", paragraphs: []string{"Option B"}},
	{typ: pyexcelize.ControlSpinButton, cell: "A9", link: "$D$9", cur: 7, lo: 1, hi: 20, inc: 2, paragraphs: []string{}},
	{typ: pyexcelize.ControlScrollBar, cell: "A11", link: "$D$11", cur: 30, lo: 0, hi: 100, inc: 1, page: 10, horizontal: true, paragraphs: []string{}},
	{typ: pyexcelize.ControlGroupBox, cell: "A13", text: "Group", paragraphs: []string{"Group"}},
	{typ: pyexcelize.ControlLabel, cell: "A15", text: "A label", paragraphs: []string{"A label"}},
	{typ: pyexcelize.ControlButton, cell: "A17", macro: "Button1_Click", text: "Click", paragraphs: []string{"Click"}},
	{typ: pyexcelize.ControlCheckBox, cell: "A19", text: "Line one Line two", paragraphs: []string{"Line one", "Line two"}},
}

func TestFormControlsFixtureFieldByField(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))

	got, err := wb.FormControls("Sheet 1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(fixtureControls) {
		t.Fatalf("got %d controls, want %d: %+v", len(got), len(fixtureControls), got)
	}
	byCell := map[string]pyexcelize.FormControl{}
	for _, c := range got {
		byCell[c.Cell] = c
	}
	for _, w := range fixtureControls {
		t.Run(w.typ.String()+"@"+w.cell, func(t *testing.T) {
			c, ok := byCell[w.cell]
			if !ok {
				t.Fatalf("no control at %s", w.cell)
			}
			checks := []struct {
				field     string
				got, want any
			}{
				{"Sheet", c.Sheet, "Sheet 1"},
				{"Type", c.Type, w.typ},
				{"TypeName", c.TypeName, w.typ.String()},
				{"Checked", c.Checked, w.checked},
				{"LinkedCell", c.LinkedCell, w.link},
				{"CurrentValue", c.CurrentValue, w.cur},
				{"MinValue", c.MinValue, w.lo},
				{"MaxValue", c.MaxValue, w.hi},
				{"IncChange", c.IncChange, w.inc},
				{"PageChange", c.PageChange, w.page},
				{"Horizontal", c.Horizontal, w.horizontal},
				{"Macro", c.Macro, w.macro},
				{"Text", c.Text, w.text},
			}
			for _, ck := range checks {
				if !reflect.DeepEqual(ck.got, ck.want) {
					t.Errorf("%s = %#v, want %#v", ck.field, ck.got, ck.want)
				}
			}
			if c.Paragraphs == nil {
				t.Fatal("Paragraphs is nil")
			}
			if !reflect.DeepEqual(c.Paragraphs, w.paragraphs) {
				t.Errorf("Paragraphs = %#v, want %#v", c.Paragraphs, w.paragraphs)
			}
		})
	}
}

func TestFormControlsEveryTypeCovered(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	got, err := wb.FormControls("Sheet 1")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[pyexcelize.ControlType]int{}
	for _, c := range got {
		seen[c.Type]++
		if c.TypeName == "unknown" || c.TypeName == "" {
			t.Errorf("control %s has bad type name %q", c.Cell, c.TypeName)
		}
	}
	for _, typ := range []pyexcelize.ControlType{
		pyexcelize.ControlButton, pyexcelize.ControlOptionButton, pyexcelize.ControlSpinButton,
		pyexcelize.ControlCheckBox, pyexcelize.ControlGroupBox, pyexcelize.ControlLabel,
		pyexcelize.ControlScrollBar,
	} {
		if seen[typ] == 0 {
			t.Errorf("fixture has no %v control", typ)
		}
	}
	if seen[pyexcelize.ControlNote] != 0 {
		t.Errorf("notes must be excluded, saw %d", seen[pyexcelize.ControlNote])
	}
	if seen[pyexcelize.ControlCheckBox] != 3 || seen[pyexcelize.ControlOptionButton] != 2 {
		t.Errorf("unexpected counts: %v", seen)
	}
}

func TestFormControlsPlainSheetEmptyNonNil(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	got, err := wb.FormControls("Plain")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("want empty non-nil slice, got %#v", got)
	}
	b, err := json.Marshal(got)
	if err != nil || string(b) != "[]" {
		t.Fatalf("json = %s, err %v", b, err)
	}
}

func TestFormControlsSheetNameCaseInsensitive(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	for _, name := range []string{"sheet 1", "SHEET 1", "Sheet 1"} {
		got, err := wb.FormControls(name)
		if err != nil {
			t.Fatalf("%q: %v", name, err)
		}
		if len(got) != len(fixtureControls) {
			t.Errorf("%q: got %d controls", name, len(got))
		}
	}
}

func TestFormControlsErrors(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	for _, name := range []string{"", "Nope", "Sheet1", " Plain", "Plain "} {
		t.Run("missing/"+name, func(t *testing.T) {
			got, err := wb.FormControls(name)
			if got != nil {
				t.Errorf("want nil slice, got %#v", got)
			}
			if k := pyexcelize.ErrorKind(err); k != pyexcelize.KindSheetNotFound {
				t.Errorf("kind = %q (%v)", k, err)
			}
		})
	}

	t.Run("corrupt sheet", func(t *testing.T) {
		bad := openFixture(t, corruptSheetXML(t, controlsBytes(t), "xl/worksheets/sheet1.xml"))
		defer bad.Close()
		if _, err := bad.FormControls("Sheet 1"); err == nil {
			t.Fatal("expected error")
		} else if !strings.Contains(err.Error(), `form controls of "Sheet 1"`) {
			t.Fatalf("missing context: %v", err)
		}
		if _, err := bad.AllFormControls(); err == nil {
			t.Fatal("AllFormControls should propagate the error")
		}
	})
}

func TestFormControlsClosed(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	_ = wb.Close()
	got, err := wb.FormControls("Sheet 1")
	if got != nil || !isKind(err, pyexcelize.KindClosed) {
		t.Fatalf("got %v, %v", got, err)
	}
	all, err := wb.AllFormControls()
	if all != nil || !isKind(err, pyexcelize.KindClosed) {
		t.Fatalf("got %v, %v", all, err)
	}
}

func TestAllFormControlsOrdering(t *testing.T) {
	t.Parallel()

	t.Run("fixture", func(t *testing.T) {
		wb := openFixture(t, controlsBytes(t))
		defer wb.Close()
		all, err := wb.AllFormControls()
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != len(fixtureControls) {
			t.Fatalf("got %d, want %d", len(all), len(fixtureControls))
		}
		for _, c := range all {
			if c.Sheet != "Sheet 1" {
				t.Errorf("control %s on sheet %q", c.Cell, c.Sheet)
			}
		}
	})

	t.Run("multi sheet in workbook order", func(t *testing.T) {
		f := excelize.NewFile()
		mustT(t, f.SetSheetName("Sheet1", "First"))
		for _, n := range []string{"Second", "Empty", "Third"} {
			if _, err := f.NewSheet(n); err != nil {
				t.Fatal(err)
			}
		}
		add := func(sheet, cell string, typ excelize.FormControlType, text string) {
			mustT(t, f.AddFormControl(sheet, excelize.FormControl{Cell: cell, Type: typ, Text: text}))
		}
		add("Third", "B2", excelize.FormControlLabel, "t1")
		add("First", "A1", excelize.FormControlCheckBox, "f1")
		add("Second", "C3", excelize.FormControlButton, "s1")
		add("First", "A5", excelize.FormControlLabel, "f2")
		wb := pyexcelize.FromFile(f)
		defer wb.Close()

		all, err := wb.AllFormControls()
		if err != nil {
			t.Fatal(err)
		}
		var order []string
		for _, c := range all {
			order = append(order, c.Sheet+":"+c.Text)
		}
		want := []string{"First:f1", "First:f2", "Second:s1", "Third:t1"}
		if !reflect.DeepEqual(order, want) {
			t.Fatalf("order = %v, want %v", order, want)
		}
	})

	t.Run("no controls anywhere", func(t *testing.T) {
		wb := openFixture(t, emptyBytes(t))
		defer wb.Close()
		all, err := wb.AllFormControls()
		if err != nil {
			t.Fatal(err)
		}
		if all == nil || len(all) != 0 {
			t.Fatalf("want empty non-nil, got %#v", all)
		}
	})
}

func TestFormControlJSONTags(t *testing.T) {
	t.Parallel()
	wb := openFixture(t, controlsBytes(t))
	all, err := wb.AllFormControls()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(all[0])
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"sheet", "type", "type_name", "cell", "linked_cell", "checked", "current_value",
		"min_value", "max_value", "inc_change", "page_change", "horizontal", "macro",
		"text", "paragraphs", "offset_x", "offset_y",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing json key %q in %s", key, b)
		}
	}
	var back pyexcelize.FormControl
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, all[0]) {
		t.Errorf("round trip mismatch:\n%+v\n%+v", back, all[0])
	}
}

func TestFormControlsViaFile(t *testing.T) {
	t.Parallel()
	files := writeFixtures(t)
	wb, err := pyexcelize.Open(files["controls.xlsx"])
	if err != nil {
		t.Fatal(err)
	}
	all, err := wb.AllFormControls()
	if err != nil || len(all) != len(fixtureControls) {
		t.Fatalf("len=%d err=%v", len(all), err)
	}
}

func BenchmarkFormControls(b *testing.B) {
	wb := openFixture(b, controlsBytes(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wb.FormControls("Sheet 1"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAllFormControls(b *testing.B) {
	wb := openFixture(b, controlsBytes(b))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := wb.AllFormControls(); err != nil {
			b.Fatal(err)
		}
	}
}

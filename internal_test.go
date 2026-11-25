package pyexcelize

import (
	"reflect"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestUsedRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		rows [][]string
		want string
	}{
		{"nil", nil, ""},
		{"no rows", [][]string{}, ""},
		{"only empty rows", [][]string{{}, {}}, ""},
		{"only empty strings", [][]string{{"", ""}, {""}}, ""},
		{"single cell", [][]string{{"x"}}, "A1"},
		{"single cell offset", [][]string{{}, {"", "", "x"}}, "C2"},
		{"row", [][]string{{"a", "b", "c"}}, "A1:C1"},
		{"column", [][]string{{"a"}, {"b"}, {"c"}}, "A1:A3"},
		{"box", [][]string{{"a", "b"}, {"c", "d"}}, "A1:B2"},
		{"leading blank rows and cols", [][]string{{}, {"", "x"}, {"", "", "", "y"}}, "B2:D3"},
		{"gap in the middle", [][]string{{"a"}, {}, {}, {"", "", "z"}}, "A1:C4"},
		{"min row after later rows", [][]string{{}, {}, {"", "", "q"}, {"", "p"}}, "B3:C4"},
		{"min col found on later row", [][]string{{"", "", "c"}, {"a"}}, "A1:C2"},
		{"whitespace counts as content", [][]string{{" "}}, "A1"},
		{"wide", [][]string{append(make([]string, 27), "far")}, "AB1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := usedRange(tc.rows); got != tc.want {
				t.Fatalf("usedRange = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCheckCell(t *testing.T) {
	t.Parallel()
	valid := []string{"A1", "a1", "XFD1048576", "xfd1", "AB12"}
	invalid := []string{"", "1A", "A0", "A", "AAAA1", "A-1", " A1", "A1:B2", "$A$1", "ZZZ1", "XFE1", "A1048577", "Ä1"}
	for _, ref := range valid {
		if err := checkCell(ref); err != nil {
			t.Errorf("checkCell(%q) = %v", ref, err)
		}
	}
	for _, ref := range invalid {
		err := checkCell(ref)
		ce, ok := err.(*InvalidCellError)
		if !ok || ce.Ref != ref {
			t.Errorf("checkCell(%q) = %#v, want *InvalidCellError", ref, err)
		}
	}
}

func TestFlattenRuns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"nil", nil, ""},
		{"empty", []string{}, ""},
		{"one", []string{"hi"}, "hi"},
		{"two", []string{"a", "b"}, "a b"},
		{"trims ends", []string{"  a", "b  "}, "a b"},
		{"leading empty fragment", []string{"", "a", "b"}, "a b"},
		{"all blank", []string{"", " ", "\t"}, ""},
		{"newlines", []string{"\nline\n"}, "line"},
		{"unicode", []string{"héllo", "wörld ✓"}, "héllo wörld ✓"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := flattenRuns(tc.in); got != tc.want {
				t.Fatalf("flattenRuns(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRunTexts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, []string{}},
		{"empty", []string{}, []string{}},
		{"drops blanks", []string{"", " ", "a", "\n"}, []string{"a"}},
		{"trims", []string{" a ", "\tb\n"}, []string{"a", "b"}},
		{"keeps order and dups", []string{"b", "a", "b"}, []string{"b", "a", "b"}},
		{"keeps inner spaces", []string{"a  b"}, []string{"a  b"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := runTexts(tc.in)
			if got == nil {
				t.Fatal("runTexts returned nil")
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("runTexts = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestNewFormControl(t *testing.T) {
	t.Parallel()
	in := excelize.FormControl{
		Cell:         "B4",
		Type:         excelize.FormControlScrollBar,
		Text:         " Main ",
		Paragraph:    []excelize.RichTextRun{{Text: "one"}, {Text: ""}, {Text: " two "}},
		Macro:        "M",
		CellLink:     "$A$1",
		Checked:      true,
		CurrentVal:   5,
		MinVal:       1,
		MaxVal:       9,
		IncChange:    2,
		PageChange:   3,
		Horizontally: true,
		Format:       excelize.GraphicOptions{OffsetX: 4, OffsetY: -6},
	}
	got := newFormControl("S", in)
	want := FormControl{
		Sheet: "S", Type: ControlScrollBar, TypeName: "scroll_bar", Cell: "B4", LinkedCell: "$A$1",
		Checked: true, CurrentValue: 5, MinValue: 1, MaxValue: 9, IncChange: 2, PageChange: 3,
		Horizontal: true, Macro: "M", Text: "Main one two", Paragraphs: []string{"Main", "one", "two"},
		OffsetX: 4, OffsetY: -6,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("newFormControl:\n got %+v\nwant %+v", got, want)
	}

	t.Run("unknown type", func(t *testing.T) {
		c := newFormControl("S", excelize.FormControl{Type: 99})
		if c.TypeName != "unknown" || c.Type != 99 {
			t.Fatalf("got %+v", c)
		}
		if c.Paragraphs == nil || len(c.Paragraphs) != 0 || c.Text != "" {
			t.Fatalf("text fields: %+v", c)
		}
	})
}

func TestBuildOptions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		opts         []Option
		wantPassword string
		wantRaw      bool
	}{
		{"none", nil, "", false},
		{"password", []Option{WithPassword("pw")}, "pw", false},
		{"raw", []Option{WithRawCellValues()}, "", true},
		{"both", []Option{WithRawCellValues(), WithPassword("pw")}, "pw", true},
		{"override", []Option{WithPassword("a"), WithPassword("b")}, "b", false},
		{"empty override", []Option{WithPassword("a"), WithPassword("")}, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := buildOptions(tc.opts)
			if o.Password != tc.wantPassword || o.RawCellValue != tc.wantRaw {
				t.Fatalf("got password=%q raw=%v", o.Password, o.RawCellValue)
			}
		})
	}
}

package capi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	pyexcelize "github.com/adhadse/py-excelize"
	"github.com/adhadse/py-excelize/internal/fixtures"
	"github.com/xuri/excelize/v2"
)

type resp struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *Error          `json:"error"`
}

func parse(t *testing.T, s string) resp {
	t.Helper()
	var r resp
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		t.Fatalf("response is not valid JSON: %v\n%s", err, s)
	}
	if r.OK && r.Error != nil {
		t.Fatalf("ok response carries error: %s", s)
	}
	if !r.OK && r.Error == nil {
		t.Fatalf("failed response has no error: %s", s)
	}
	return r
}

func mustOK(t *testing.T, s string) resp {
	t.Helper()
	r := parse(t, s)
	if !r.OK {
		t.Fatalf("expected ok, got error %+v", r.Error)
	}
	return r
}

func mustFail(t *testing.T, s, kind string) resp {
	t.Helper()
	r := parse(t, s)
	if r.OK {
		t.Fatalf("expected failure kind %q, got ok: %s", kind, s)
	}
	if r.Error.Kind != kind {
		t.Fatalf("kind = %q, want %q (message %q)", r.Error.Kind, kind, r.Error.Message)
	}
	if r.Error.Message == "" {
		t.Fatalf("empty error message")
	}
	if len(r.Result) != 0 {
		t.Fatalf("failed response has result %s", r.Result)
	}
	return r
}

func handleOf(t *testing.T, s string) int64 {
	t.Helper()
	r := mustOK(t, s)
	var v struct {
		Handle int64 `json:"handle"`
	}
	if err := json.Unmarshal(r.Result, &v); err != nil {
		t.Fatal(err)
	}
	if v.Handle <= 0 {
		t.Fatalf("handle = %d", v.Handle)
	}
	return v.Handle
}

func fixturePaths(t *testing.T) map[string]string {
	t.Helper()
	p, err := fixtures.WriteAll(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func decode[T any](t *testing.T, r resp) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(r.Result, &v); err != nil {
		t.Fatalf("decode %s: %v", r.Result, err)
	}
	return v
}

func call(t *testing.T, reg *Registry, h int64, method, args string) resp {
	t.Helper()
	return parse(t, reg.Call(h, method, args))
}

func TestVersion(t *testing.T) {
	if Version == "" || strings.Count(Version, ".") != 2 {
		t.Fatalf("Version = %q", Version)
	}
}

func TestNewRegistryEmpty(t *testing.T) {
	r := NewRegistry()
	if r.Len() != 0 {
		t.Fatalf("Len = %d", r.Len())
	}
	if r.next != 1 {
		t.Fatalf("next = %d", r.next)
	}
}

func TestOpen(t *testing.T) {
	paths := fixturePaths(t)
	dir := t.TempDir()
	notXlsx := filepath.Join(dir, "junk.xlsx")
	if err := os.WriteFile(notXlsx, []byte("this is not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		path     string
		password string
		kind     string // "" => success
	}{
		{"plain", paths[fixtures.FileCells], "", ""},
		{"empty workbook", paths[fixtures.FileEmpty], "", ""},
		{"controls", paths[fixtures.FileControls], "", ""},
		{"password ignored for plain", paths[fixtures.FileCells], "whatever", ""},
		{"missing file", filepath.Join(dir, "nope.xlsx"), "", "file_not_found"},
		{"directory", dir, "", "error"},
		{"junk file", notXlsx, "", "format"},
		{"encrypted without password", paths[fixtures.FileEncrypted], "", "password"},
		{"encrypted wrong password", paths[fixtures.FileEncrypted], "wrong", "password"},
		{"encrypted right password", paths[fixtures.FileEncrypted], fixtures.Password, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewRegistry()
			out := reg.Open(tc.path, tc.password)
			if tc.kind == "" {
				h := handleOf(t, out)
				if reg.Len() != 1 {
					t.Fatalf("Len = %d", reg.Len())
				}
				names := decode[[]string](t, call(t, reg, h, "sheet_names", ""))
				if len(names) == 0 {
					t.Fatal("no sheets")
				}
				return
			}
			if tc.name == "directory" {
				// kind depends on the platform; just require a failure.
				if r := parse(t, out); r.OK {
					t.Fatalf("expected failure: %s", out)
				}
			} else {
				mustFail(t, out, tc.kind)
			}
			if reg.Len() != 0 {
				t.Fatalf("failed open leaked a handle: Len = %d", reg.Len())
			}
		})
	}
}

func TestOpenBytes(t *testing.T) {
	paths := fixturePaths(t)
	read := func(name string) []byte {
		b, err := os.ReadFile(paths[name])
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	tests := []struct {
		name     string
		data     []byte
		password string
		kind     string
	}{
		{"valid", read(fixtures.FileCells), "", ""},
		{"valid controls", read(fixtures.FileControls), "", ""},
		{"junk", []byte("definitely not xlsx"), "", "format"},
		{"empty", []byte{}, "", "format"},
		{"nil", nil, "", "format"},
		{"truncated zip", read(fixtures.FileCells)[:40], "", "format"},
		{"encrypted no password", read(fixtures.FileEncrypted), "", "password"},
		{"encrypted wrong password", read(fixtures.FileEncrypted), "bad", "password"},
		{"encrypted right password", read(fixtures.FileEncrypted), fixtures.Password, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reg := NewRegistry()
			out := reg.OpenBytes(tc.data, tc.password)
			if tc.kind == "" {
				h := handleOf(t, out)
				if names := decode[[]string](t, call(t, reg, h, "sheet_names", "")); len(names) == 0 {
					t.Fatal("no sheets")
				}
				if reg.Len() != 1 {
					t.Fatalf("Len = %d", reg.Len())
				}
				return
			}
			mustFail(t, out, tc.kind)
			if reg.Len() != 0 {
				t.Fatalf("Len = %d", reg.Len())
			}
		})
	}
}

func decodeBool(t *testing.T, r resp) bool {
	t.Helper()
	if !r.OK {
		t.Fatalf("not ok: %+v", r.Error)
	}
	return decode[bool](t, r)
}

func TestClose(t *testing.T) {
	paths := fixturePaths(t)
	reg := NewRegistry()
	h := handleOf(t, reg.Open(paths[fixtures.FileCells], ""))

	r := mustOK(t, reg.Close(h))
	if !decode[bool](t, r) {
		t.Fatal("close result should be true")
	}
	if reg.Len() != 0 {
		t.Fatalf("Len = %d", reg.Len())
	}
	// Using the handle afterwards reports closed.
	mustFail(t, reg.Call(h, "sheet_names", ""), "closed")
	// Double close and unknown handles succeed.
	for _, id := range []int64{h, 0, -5, 9999} {
		r := mustOK(t, reg.Close(id))
		if !decode[bool](t, r) {
			t.Fatalf("close(%d) result should be true", id)
		}
	}
}

func TestLenAndHandleUniqueness(t *testing.T) {
	paths := fixturePaths(t)
	reg := NewRegistry()
	seen := map[int64]bool{}
	var prev int64
	const n = 25
	var hs []int64
	for i := 0; i < n; i++ {
		var out string
		if i%2 == 0 {
			out = reg.Open(paths[fixtures.FileEmpty], "")
		} else {
			b, err := os.ReadFile(paths[fixtures.FileCells])
			if err != nil {
				t.Fatal(err)
			}
			out = reg.OpenBytes(b, "")
		}
		h := handleOf(t, out)
		if seen[h] {
			t.Fatalf("duplicate handle %d", h)
		}
		if h <= prev {
			t.Fatalf("handle %d not greater than previous %d", h, prev)
		}
		seen[h], prev = true, h
		hs = append(hs, h)
		if reg.Len() != i+1 {
			t.Fatalf("Len = %d, want %d", reg.Len(), i+1)
		}
	}
	for i, h := range hs {
		mustOK(t, reg.Close(h))
		if reg.Len() != n-i-1 {
			t.Fatalf("Len after close = %d", reg.Len())
		}
	}
	// Handles are never reused after closing.
	h := handleOf(t, reg.Open(paths[fixtures.FileEmpty], ""))
	if seen[h] || h <= prev {
		t.Fatalf("handle %d reused or non-monotonic (prev %d)", h, prev)
	}
	// Failed opens do not consume handles.
	mustFail(t, reg.Open("/no/such/file.xlsx", ""), "file_not_found")
	h2 := handleOf(t, reg.Open(paths[fixtures.FileEmpty], ""))
	if h2 != h+1 {
		t.Fatalf("handle after failed open = %d, want %d", h2, h+1)
	}
}

func openFixture(t *testing.T, name string) (*Registry, int64) {
	t.Helper()
	reg := NewRegistry()
	paths := fixturePaths(t)
	h := handleOf(t, reg.Open(paths[name], ""))
	t.Cleanup(func() { reg.Close(h) })
	return reg, h
}

func TestCallCells(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileCells)

	tests := []struct {
		name   string
		method string
		args   string
		want   any
	}{
		{"sheet_names", "sheet_names", "", []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
		{"sheet_names ignores args", "sheet_names", `{"sheet":"zzz"}`, []string{"Sheet1", "Sheet 2", "Ünï Çødé"}},
		{"has_sheet yes", "has_sheet", `{"sheet":"Sheet1"}`, true},
		{"has_sheet case-insensitive", "has_sheet", `{"sheet":"sheet 2"}`, true},
		{"has_sheet unicode", "has_sheet", `{"sheet":"Ünï Çødé"}`, true},
		{"has_sheet no", "has_sheet", `{"sheet":"Nope"}`, false},
		{"has_sheet empty", "has_sheet", `{}`, false},
		{"has_sheet no args", "has_sheet", ``, false},
		{"dimension Sheet1", "dimension", `{"sheet":"Sheet1"}`, "A1:F7"},
		{"dimension Sheet 2", "dimension", `{"sheet":"Sheet 2"}`, "B2"},
		{"dimension unicode", "dimension", `{"sheet":"Ünï Çødé"}`, "A1"},
		{"cell_value text", "cell_value", `{"sheet":"Sheet1","cell":"A1"}`, "name"},
		{"cell_value lowercase ref", "cell_value", `{"sheet":"Sheet1","cell":"a2"}`, "apple"},
		{"cell_value int", "cell_value", `{"sheet":"Sheet1","cell":"B2"}`, "3"},
		{"cell_value float", "cell_value", `{"sheet":"Sheet1","cell":"C3"}`, "0.25"},
		{"cell_value bool", "cell_value", `{"sheet":"Sheet1","cell":"F2"}`, "TRUE"},
		{"cell_value empty", "cell_value", `{"sheet":"Sheet1","cell":"Z99"}`, ""},
		{"cell_value merged", "cell_value", `{"sheet":"Sheet1","cell":"A7"}`, "merged title"},
		{"cell_value other sheet", "cell_value", `{"sheet":"Sheet 2","cell":"B2"}`, "spaced"},
		{"cell_value unicode", "cell_value", `{"sheet":"Ünï Çødé","cell":"A1"}`, "héllo wörld ✓"},
		{"cell_value sheet case", "cell_value", `{"sheet":"SHEET1","cell":"A1"}`, "name"},
		{"cell_formula", "cell_formula", `{"sheet":"Sheet1","cell":"D2"}`, "B2*C2"},
		{"cell_formula none", "cell_formula", `{"sheet":"Sheet1","cell":"A1"}`, ""},
		{"rows Sheet1", "rows", `{"sheet":"Sheet1"}`, [][]string{
			{"name", "qty", "price"},
			{"apple", "3", "1.5", "", "", "TRUE"},
			{"pear", "10", "0.25"},
			{},
			{"gap row"},
			{},
			{"merged title"},
		}},
		{"rows Sheet 2", "rows", `{"sheet":"Sheet 2"}`, [][]string{{}, {"", "spaced"}}},
		{"merged_cells", "merged_cells", `{"sheet":"Sheet1"}`, []pyexcelize.MergedCell{
			{Range: "A7:C7", Start: "A7", End: "C7", Value: "merged title"},
			{Range: "E1:E3", Start: "E1", End: "E3", Value: ""},
		}},
		{"merged_cells none", "merged_cells", `{"sheet":"Sheet 2"}`, []pyexcelize.MergedCell{}},
		{"form_controls none", "form_controls", `{"sheet":"Sheet1"}`, []pyexcelize.FormControl{}},
		{"all_form_controls none", "all_form_controls", ``, []pyexcelize.FormControl{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := call(t, reg, h, tc.method, tc.args)
			if !r.OK {
				t.Fatalf("error: %+v", r.Error)
			}
			got := reflect.New(reflect.TypeOf(tc.want))
			if err := json.Unmarshal(r.Result, got.Interface()); err != nil {
				t.Fatalf("decode %s: %v", r.Result, err)
			}
			if !reflect.DeepEqual(got.Elem().Interface(), tc.want) {
				t.Fatalf("got %#v\nwant %#v", got.Elem().Interface(), tc.want)
			}
		})
	}
}

func TestCallEmptyWorkbook(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileEmpty)
	if got := decode[[]string](t, call(t, reg, h, "sheet_names", "")); !reflect.DeepEqual(got, []string{"Sheet1"}) {
		t.Fatalf("sheet_names = %v", got)
	}
	if got := decode[string](t, call(t, reg, h, "dimension", `{"sheet":"Sheet1"}`)); got != "" {
		t.Fatalf("dimension = %q", got)
	}
	if got := decode[[][]string](t, call(t, reg, h, "rows", `{"sheet":"Sheet1"}`)); len(got) != 0 {
		t.Fatalf("rows = %v", got)
	}
}

func TestCallControls(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileControls)

	if got := decode[[]string](t, call(t, reg, h, "sheet_names", "")); !reflect.DeepEqual(got, []string{"Sheet 1", "Plain"}) {
		t.Fatalf("sheet_names = %v", got)
	}

	type ctl struct {
		Sheet      string   `json:"sheet"`
		TypeName   string   `json:"type_name"`
		Cell       string   `json:"cell"`
		LinkedCell string   `json:"linked_cell"`
		Checked    bool     `json:"checked"`
		Text       string   `json:"text"`
		Macro      string   `json:"macro"`
		Paragraphs []string `json:"paragraphs"`
	}
	one := call(t, reg, h, "form_controls", `{"sheet":"Sheet 1"}`)
	all := call(t, reg, h, "all_form_controls", "")
	for name, r := range map[string]resp{"form_controls": one, "all_form_controls": all} {
		got := decode[[]ctl](t, r)
		if len(got) != 10 {
			t.Fatalf("%s: %d controls, want 10: %s", name, len(got), r.Result)
		}
		byCell := map[string]ctl{}
		for _, c := range got {
			if c.Sheet != "Sheet 1" {
				t.Fatalf("%s: sheet = %q", name, c.Sheet)
			}
			byCell[c.Cell] = c
		}
		for cell, wantText := range map[string]string{
			"A1": "Agree", "A3": "Subscribe", "A5": "Option A", "A7": "Option B",
			"A13": "Group", "A15": "A label", "A17": "Click",
		} {
			if c, ok := byCell[cell]; !ok || c.Text != wantText {
				t.Fatalf("%s: %s = %+v, want text %q", name, cell, c, wantText)
			}
		}
		if c := byCell["A1"]; !c.Checked || c.LinkedCell == "" {
			t.Fatalf("%s: A1 = %+v", name, c)
		}
		if c := byCell["A3"]; c.Checked {
			t.Fatalf("%s: A3 should be unchecked: %+v", name, c)
		}
		if c := byCell["A17"]; c.Macro != "Button1_Click" {
			t.Fatalf("%s: A17 macro = %q", name, c.Macro)
		}
		if c := byCell["A19"]; !reflect.DeepEqual(c.Paragraphs, []string{"Line one", "Line two"}) {
			t.Fatalf("%s: A19 paragraphs = %v", name, c.Paragraphs)
		}
	}
	if string(one.Result) != string(all.Result) {
		t.Fatalf("all_form_controls differs from form_controls for the only sheet with controls")
	}

	type numeric struct {
		Cell       string `json:"cell"`
		Current    uint   `json:"current_value"`
		Min        uint   `json:"min_value"`
		Max        uint   `json:"max_value"`
		Inc        uint   `json:"inc_change"`
		Page       uint   `json:"page_change"`
		Horizontal bool   `json:"horizontal"`
	}
	nums := map[string]numeric{}
	for _, n := range decode[[]numeric](t, one) {
		nums[n.Cell] = n
	}
	if got, want := nums["A9"], (numeric{"A9", 7, 1, 20, 2, 0, false}); got.Current != want.Current || got.Min != want.Min || got.Max != want.Max || got.Inc != want.Inc {
		t.Fatalf("spin button = %+v", got)
	}
	if got := nums["A11"]; got.Current != 30 || got.Max != 100 || got.Page != 10 || !got.Horizontal {
		t.Fatalf("scroll bar = %+v", got)
	}

	if got := decode[[]ctl](t, call(t, reg, h, "form_controls", `{"sheet":"Plain"}`)); got == nil || len(got) != 0 {
		t.Fatalf("Plain controls = %#v", got)
	}
	if s := call(t, reg, h, "form_controls", `{"sheet":"Plain"}`); string(s.Result) != "[]" {
		t.Fatalf("Plain result = %s", s.Result)
	}
}

func TestCallErrors(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileCells)
	tests := []struct {
		name   string
		method string
		args   string
		kind   string
	}{
		{"dimension unknown sheet", "dimension", `{"sheet":"Nope"}`, "sheet_not_found"},
		{"dimension no sheet", "dimension", `{}`, "sheet_not_found"},
		{"cell_value unknown sheet", "cell_value", `{"sheet":"Nope","cell":"A1"}`, "sheet_not_found"},
		{"cell_formula unknown sheet", "cell_formula", `{"sheet":"Nope","cell":"A1"}`, "sheet_not_found"},
		{"rows unknown sheet", "rows", `{"sheet":"Nope"}`, "sheet_not_found"},
		{"merged_cells unknown sheet", "merged_cells", `{"sheet":"Nope"}`, "sheet_not_found"},
		{"form_controls unknown sheet", "form_controls", `{"sheet":"Nope"}`, "sheet_not_found"},
		{"cell_value bad ref", "cell_value", `{"sheet":"Sheet1","cell":"1A"}`, "invalid_cell"},
		{"cell_value empty ref", "cell_value", `{"sheet":"Sheet1"}`, "invalid_cell"},
		{"cell_value zero row", "cell_value", `{"sheet":"Sheet1","cell":"A0"}`, "invalid_cell"},
		{"cell_value garbage", "cell_value", `{"sheet":"Sheet1","cell":"A1:B2"}`, "invalid_cell"},
		{"cell_value col too wide", "cell_value", `{"sheet":"Sheet1","cell":"ZZZ1"}`, "invalid_cell"},
		{"cell_formula bad ref", "cell_formula", `{"sheet":"Sheet1","cell":"$$"}`, "invalid_cell"},
		{"sheet checked before cell", "cell_value", `{"sheet":"Nope","cell":"??"}`, "sheet_not_found"},
		{"malformed json", "sheet_names", `{not json`, "error"},
		{"args wrong type", "has_sheet", `{"sheet":5}`, "error"},
		{"args not object", "has_sheet", `[1,2]`, "error"},
		{"unknown method", "explode", ``, "error"},
		{"empty method", "", ``, "error"},
		{"method case sensitive", "SHEET_NAMES", ``, "error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := mustFail(t, reg.Call(h, tc.method, tc.args), tc.kind)
			switch tc.name {
			case "malformed json", "args wrong type", "args not object":
				if !strings.Contains(r.Error.Message, "invalid arguments") {
					t.Fatalf("message = %q", r.Error.Message)
				}
			case "unknown method", "empty method", "method case sensitive":
				if !strings.Contains(r.Error.Message, "unknown method") {
					t.Fatalf("message = %q", r.Error.Message)
				}
			case "dimension unknown sheet":
				if !strings.Contains(r.Error.Message, `"Nope"`) {
					t.Fatalf("message = %q", r.Error.Message)
				}
			}
		})
	}
}

func TestCallClosedAndUnknownHandle(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileCells)
	methods := []string{"sheet_names", "has_sheet", "dimension", "cell_value", "cell_formula", "rows", "merged_cells", "form_controls", "all_form_controls", "bogus"}
	for _, id := range []int64{0, -1, h + 100} {
		for _, m := range methods {
			mustFail(t, reg.Call(id, m, `{"sheet":"Sheet1","cell":"A1"}`), "closed")
		}
	}
	mustOK(t, reg.Close(h))
	for _, m := range methods {
		mustFail(t, reg.Call(h, m, `{"sheet":"Sheet1","cell":"A1"}`), "closed")
	}
	// Unknown handle wins even over malformed args.
	mustFail(t, reg.Call(h, "rows", "{bad"), "closed")
}

func TestCallOnClosedWorkbookInRegistry(t *testing.T) {
	// A workbook closed behind the registry's back reports "closed" for
	// methods that surface the error.
	reg := NewRegistry()
	wb := pyexcelize.FromFile(excelize.NewFile())
	h := reg.add(wb)
	if err := wb.Close(); err != nil {
		t.Fatal(err)
	}
	for _, m := range []string{"dimension", "cell_value", "cell_formula", "rows", "merged_cells", "form_controls", "all_form_controls"} {
		mustFail(t, reg.Call(h, m, `{"sheet":"Sheet1","cell":"A1"}`), "closed")
	}
	if got := decode[[]string](t, mustOK(t, reg.Call(h, "sheet_names", ""))); got == nil || len(got) != 0 {
		t.Fatalf("sheet_names = %#v", got)
	}
	if decodeBool(t, call(t, reg, h, "has_sheet", `{"sheet":"Sheet1"}`)) {
		t.Fatal("closed workbook should have no sheets")
	}
	mustOK(t, reg.Close(h))
}

func TestCallPanicRecovery(t *testing.T) {
	// A corrupt package part makes the underlying library panic on a type
	// assertion; Call must convert the panic into an error response.
	reg := NewRegistry()
	f := excelize.NewFile()
	f.Sheet.Delete("xl/worksheets/sheet1.xml")
	f.Pkg.Store("xl/worksheets/sheet1.xml", 42)
	h := reg.add(pyexcelize.FromFile(f))
	r := mustFail(t, reg.Call(h, "rows", `{"sheet":"Sheet1"}`), "error")
	if !strings.Contains(r.Error.Message, "internal panic") {
		t.Fatalf("message = %q", r.Error.Message)
	}
	// The registry stays usable after a recovered panic.
	if reg.Len() != 1 {
		t.Fatalf("Len = %d", reg.Len())
	}
	mustOK(t, reg.Close(h))
}

func TestEncode(t *testing.T) {
	tests := []struct {
		name string
		v    any
		err  error
		want string
	}{
		{"nil value ok", nil, nil, `{"ok":true}`},
		{"string", "hi", nil, `{"ok":true,"result":"hi"}`},
		{"false is kept", false, nil, `{"ok":true,"result":false}`},
		{"true", true, nil, `{"ok":true,"result":true}`},
		{"error drops value", "ignored", pyexcelize.ErrClosed, `{"ok":false,"error":{"kind":"closed","message":"pyexcelize: workbook is closed"}}`},
		{"other error", nil, fmt.Errorf("boom"), `{"ok":false,"error":{"kind":"error","message":"boom"}}`},
		{"wrapped kind", nil, fmt.Errorf("ctx: %w", &pyexcelize.InvalidCellError{Ref: "x"}), `{"ok":false,"error":{"kind":"invalid_cell","message":"ctx: pyexcelize: invalid cell reference \"x\""}}`},
		{"unmarshalable", make(chan int), nil, `{"ok":false,"error":{"kind":"error","message":"encode response"}}`},
		{"unmarshalable func", func() {}, nil, `{"ok":false,"error":{"kind":"error","message":"encode response"}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := encode(tc.v, tc.err); got != tc.want {
				t.Fatalf("got  %s\nwant %s", got, tc.want)
			}
			parse(t, encode(tc.v, tc.err))
		})
	}
}

func TestOpts(t *testing.T) {
	if got := opts(""); got != nil {
		t.Fatalf("opts(\"\") = %v", got)
	}
	if got := opts("pw"); len(got) != 1 {
		t.Fatalf("opts(pw) len = %d", len(got))
	}
}

func TestResponseShapeIsPureJSON(t *testing.T) {
	reg, h := openFixture(t, fixtures.FileCells)
	for _, out := range []string{
		reg.Call(h, "rows", `{"sheet":"Sheet1"}`),
		reg.Call(h, "nope", ""),
		reg.Close(99999),
	} {
		var m map[string]any
		if err := json.Unmarshal([]byte(out), &m); err != nil {
			t.Fatalf("not an object: %s", out)
		}
		if _, ok := m["ok"].(bool); !ok {
			t.Fatalf("missing ok: %s", out)
		}
	}
}

func TestConcurrentUse(t *testing.T) {
	paths := fixturePaths(t)
	data, err := os.ReadFile(paths[fixtures.FileCells])
	if err != nil {
		t.Fatal(err)
	}
	reg := NewRegistry()
	shared := handleOf(t, reg.OpenBytes(data, ""))

	const goroutines = 32
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		handles = map[int64]bool{}
		errs    = make(chan error, goroutines*64)
	)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 4; i++ {
				var out string
				if (g+i)%2 == 0 {
					out = reg.Open(paths[fixtures.FileCells], "")
				} else {
					out = reg.OpenBytes(data, "")
				}
				var o struct {
					OK     bool `json:"ok"`
					Result struct {
						Handle int64 `json:"handle"`
					} `json:"result"`
				}
				if err := json.Unmarshal([]byte(out), &o); err != nil || !o.OK {
					errs <- fmt.Errorf("open: %v %s", err, out)
					continue
				}
				h := o.Result.Handle
				mu.Lock()
				if handles[h] {
					errs <- fmt.Errorf("duplicate handle %d", h)
				}
				handles[h] = true
				mu.Unlock()

				for _, id := range []int64{h, shared} {
					if got := reg.Call(id, "cell_value", `{"sheet":"Sheet1","cell":"A2"}`); !strings.Contains(got, `"apple"`) {
						errs <- fmt.Errorf("cell_value: %s", got)
					}
					if got := reg.Call(id, "sheet_names", ""); !strings.Contains(got, "Sheet 2") {
						errs <- fmt.Errorf("sheet_names: %s", got)
					}
					_ = reg.Len()
				}
				if got := reg.Close(h); !strings.Contains(got, `"ok":true`) {
					errs <- fmt.Errorf("close: %s", got)
				}
				if got := reg.Call(h, "sheet_names", ""); !strings.Contains(got, `"closed"`) {
					errs <- fmt.Errorf("after close: %s", got)
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if len(handles) != goroutines*4 {
		t.Fatalf("unique handles = %d, want %d", len(handles), goroutines*4)
	}
	if reg.Len() != 1 {
		t.Fatalf("Len = %d, want only the shared workbook", reg.Len())
	}
	mustOK(t, reg.Close(shared))
	if reg.Len() != 0 {
		t.Fatalf("Len = %d", reg.Len())
	}
}

func TestConcurrentCloseSameHandle(t *testing.T) {
	paths := fixturePaths(t)
	reg := NewRegistry()
	h := handleOf(t, reg.Open(paths[fixtures.FileCells], ""))
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if out := reg.Close(h); !strings.Contains(out, `"ok":true`) {
				t.Errorf("close: %s", out)
			}
		}()
	}
	wg.Wait()
	if reg.Len() != 0 {
		t.Fatalf("Len = %d", reg.Len())
	}
}

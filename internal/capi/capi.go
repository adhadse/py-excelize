// Package capi implements the JSON request/response protocol used by the
// C shared library that backs the Python bindings. It is pure Go so that it
// can be unit tested without cgo.
package capi

import (
	"encoding/json"
	"fmt"
	"sync"

	pyexcelize "github.com/adhadse/py-excelize"
)

// Version of the wire protocol / library.
const Version = "0.4.0"

// Error is the error payload of a failed response.
type Error struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type response struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result,omitempty"`
	Error  *Error `json:"error,omitempty"`
}

// Registry owns open workbooks and hands out integer handles.
type Registry struct {
	mu   sync.Mutex
	next int64
	wbs  map[int64]*pyexcelize.Workbook
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{next: 1, wbs: map[int64]*pyexcelize.Workbook{}}
}

// Len returns the number of open workbooks.
func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.wbs)
}

func encode(v any, err error) string {
	resp := response{OK: err == nil, Result: v}
	if err != nil {
		resp.Result = nil
		resp.Error = &Error{Kind: pyexcelize.ErrorKind(err), Message: err.Error()}
	}
	b, merr := json.Marshal(resp)
	if merr != nil {
		return `{"ok":false,"error":{"kind":"error","message":"encode response"}}`
	}
	return string(b)
}

func (r *Registry) add(wb *pyexcelize.Workbook) int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	h := r.next
	r.next++
	r.wbs[h] = wb
	return h
}

func opts(password string) []pyexcelize.Option {
	if password == "" {
		return nil
	}
	return []pyexcelize.Option{pyexcelize.WithPassword(password)}
}

// Open opens a workbook from a path and returns {"handle": n}.
func (r *Registry) Open(path, password string) string {
	wb, err := pyexcelize.Open(path, opts(password)...)
	if err != nil {
		return encode(nil, err)
	}
	return encode(map[string]int64{"handle": r.add(wb)}, nil)
}

// OpenBytes opens a workbook from memory and returns {"handle": n}.
func (r *Registry) OpenBytes(data []byte, password string) string {
	wb, err := pyexcelize.OpenBytes(data, opts(password)...)
	if err != nil {
		return encode(nil, err)
	}
	return encode(map[string]int64{"handle": r.add(wb)}, nil)
}

// Close closes and forgets a handle. Closing an unknown handle succeeds.
func (r *Registry) Close(h int64) string {
	r.mu.Lock()
	wb := r.wbs[h]
	delete(r.wbs, h)
	r.mu.Unlock()
	return encode(true, wb.Close())
}

type args struct {
	Sheet string `json:"sheet"`
	Cell  string `json:"cell"`
}

// Call invokes method on the workbook behind handle h with JSON arguments.
func (r *Registry) Call(h int64, method, argsJSON string) (out string) {
	defer func() {
		if p := recover(); p != nil {
			out = encode(nil, fmt.Errorf("internal panic: %v", p))
		}
	}()
	r.mu.Lock()
	wb, ok := r.wbs[h]
	r.mu.Unlock()
	if !ok {
		return encode(nil, pyexcelize.ErrClosed)
	}
	var a args
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &a); err != nil {
			return encode(nil, fmt.Errorf("invalid arguments: %w", err))
		}
	}
	switch method {
	case "sheet_names":
		return encode(wb.SheetNames(), nil)
	case "has_sheet":
		return encode(wb.HasSheet(a.Sheet), nil)
	case "dimension":
		v, err := wb.Dimension(a.Sheet)
		return encode(v, err)
	case "cell_value":
		v, err := wb.CellValue(a.Sheet, a.Cell)
		return encode(v, err)
	case "cell_formula":
		v, err := wb.CellFormula(a.Sheet, a.Cell)
		return encode(v, err)
	case "rows":
		v, err := wb.Rows(a.Sheet)
		return encode(v, err)
	case "merged_cells":
		v, err := wb.MergedCells(a.Sheet)
		return encode(v, err)
	case "form_controls":
		v, err := wb.FormControls(a.Sheet)
		return encode(v, err)
	case "all_form_controls":
		v, err := wb.AllFormControls()
		return encode(v, err)
	}
	return encode(nil, fmt.Errorf("unknown method %q", method))
}

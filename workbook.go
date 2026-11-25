package pyexcelize

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Workbook is an open XLSX workbook. A Workbook is not safe for concurrent
// use; wrap it in a mutex if it must be shared between goroutines.
type Workbook struct {
	f *excelize.File
}

// Option configures how a workbook is opened.
type Option func(*excelize.Options)

// WithPassword sets the password used to open an encrypted workbook.
func WithPassword(password string) Option {
	return func(o *excelize.Options) { o.Password = password }
}

// WithRawCellValues disables number/date formatting so that cell values are
// returned exactly as stored in the file.
func WithRawCellValues() Option {
	return func(o *excelize.Options) { o.RawCellValue = true }
}

func buildOptions(opts []Option) excelize.Options {
	var o excelize.Options
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// Open opens the workbook stored at path.
func Open(path string, opts ...Option) (*Workbook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("pyexcelize: open %q: %w", path, err)
	}
	wb, err := OpenBytes(data, opts...)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	return wb, nil
}

// OpenReader reads a workbook from r.
func OpenReader(r io.Reader, opts ...Option) (*Workbook, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("pyexcelize: read workbook: %w", err)
	}
	return OpenBytes(data, opts...)
}

// cfbMagic is the header of OLE compound files, which is how Excel stores
// password protected workbooks.
var cfbMagic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// OpenBytes reads a workbook from an in-memory buffer.
func OpenBytes(data []byte, opts ...Option) (*Workbook, error) {
	o := buildOptions(opts)
	encrypted := bytes.HasPrefix(data, cfbMagic)
	if encrypted && o.Password == "" {
		return nil, ErrPasswordRequired
	}
	f, err := excelize.OpenReader(bytes.NewReader(data), o)
	if err != nil {
		if errors.Is(err, excelize.ErrWorkbookPassword) {
			return nil, ErrWrongPassword
		}
		return nil, fmt.Errorf("pyexcelize: read workbook: %w", err)
	}
	return &Workbook{f: f}, nil
}

// FromFile wraps an already opened excelize file. The Workbook takes
// ownership and closes it on Close.
func FromFile(f *excelize.File) *Workbook {
	return &Workbook{f: f}
}

// Close releases the workbook. It is safe to call more than once and on a nil
// Workbook.
func (w *Workbook) Close() error {
	if w == nil || w.f == nil {
		return nil
	}
	err := w.f.Close()
	w.f = nil
	return err
}

// Closed reports whether the workbook has been closed.
func (w *Workbook) Closed() bool { return w == nil || w.f == nil }

// SheetNames returns all sheet names in workbook order. It returns an empty,
// non-nil slice for a closed workbook.
func (w *Workbook) SheetNames() []string {
	if w.Closed() {
		return []string{}
	}
	names := w.f.GetSheetList()
	if names == nil {
		return []string{}
	}
	return names
}

// HasSheet reports whether a sheet with the given name exists. Matching is
// case-insensitive, as in Excel.
func (w *Workbook) HasSheet(name string) bool {
	return w.checkSheet(name) == nil
}

func (w *Workbook) checkSheet(name string) error {
	if w.Closed() {
		return ErrClosed
	}
	for _, n := range w.f.GetSheetList() {
		if strings.EqualFold(n, name) {
			return nil
		}
	}
	return &SheetNotFoundError{Name: name}
}

var cellRef = regexp.MustCompile(`^[A-Za-z]{1,3}[1-9][0-9]{0,6}$`)

func checkCell(ref string) error {
	if !cellRef.MatchString(ref) {
		return &InvalidCellError{Ref: ref}
	}
	if _, _, err := excelize.CellNameToCoordinates(ref); err != nil {
		return &InvalidCellError{Ref: ref}
	}
	return nil
}

// Dimension returns the bounding box of the non-empty cells of a sheet, for
// example "A1:C10". It returns "" for a sheet without data.
func (w *Workbook) Dimension(sheet string) (string, error) {
	rows, err := w.Rows(sheet)
	if err != nil {
		return "", err
	}
	return usedRange(rows), nil
}

// usedRange computes the bounding box of non-empty cells in rows.
func usedRange(rows [][]string) string {
	minRow, maxRow, minCol, maxCol := 0, 0, 0, 0
	for r, row := range rows {
		for c, v := range row {
			if v == "" {
				continue
			}
			if minRow == 0 || r+1 < minRow {
				minRow = r + 1
			}
			if r+1 > maxRow {
				maxRow = r + 1
			}
			if minCol == 0 || c+1 < minCol {
				minCol = c + 1
			}
			if c+1 > maxCol {
				maxCol = c + 1
			}
		}
	}
	if minRow == 0 {
		return ""
	}
	start, _ := excelize.CoordinatesToCellName(minCol, minRow)
	end, _ := excelize.CoordinatesToCellName(maxCol, maxRow)
	if start == end {
		return start
	}
	return start + ":" + end
}

// CellValue returns the formatted value of a cell ("" if empty).
func (w *Workbook) CellValue(sheet, cell string) (string, error) {
	if err := w.checkSheet(sheet); err != nil {
		return "", err
	}
	if err := checkCell(cell); err != nil {
		return "", err
	}
	return w.f.GetCellValue(sheet, cell)
}

// CellFormula returns the formula stored in a cell ("" if none).
func (w *Workbook) CellFormula(sheet, cell string) (string, error) {
	if err := w.checkSheet(sheet); err != nil {
		return "", err
	}
	if err := checkCell(cell); err != nil {
		return "", err
	}
	return w.f.GetCellFormula(sheet, cell)
}

// Rows returns every row of a sheet as strings. Trailing empty cells are
// omitted by the underlying reader, so rows may have different lengths.
func (w *Workbook) Rows(sheet string) ([][]string, error) {
	if err := w.checkSheet(sheet); err != nil {
		return nil, err
	}
	rows, err := w.f.GetRows(sheet)
	if rows == nil {
		rows = [][]string{}
	}
	for i, r := range rows {
		if r == nil {
			rows[i] = []string{}
		}
	}
	return rows, err
}

// MergedCell describes a merged range.
type MergedCell struct {
	Range string `json:"range"`
	Start string `json:"start"`
	End   string `json:"end"`
	Value string `json:"value"`
}

// MergedCells returns the merged ranges of a sheet.
func (w *Workbook) MergedCells(sheet string) ([]MergedCell, error) {
	if err := w.checkSheet(sheet); err != nil {
		return nil, err
	}
	merged, err := w.f.GetMergeCells(sheet)
	if err != nil {
		return nil, err
	}
	out := make([]MergedCell, 0, len(merged))
	for _, m := range merged {
		out = append(out, MergedCell{
			Range: m.GetStartAxis() + ":" + m.GetEndAxis(),
			Start: m.GetStartAxis(),
			End:   m.GetEndAxis(),
			Value: m.GetCellValue(),
		})
	}
	return out, nil
}

// FormControls returns all legacy form controls on a sheet. Notes are
// excluded. It returns an empty, non-nil slice when there are none.
func (w *Workbook) FormControls(sheet string) ([]FormControl, error) {
	if err := w.checkSheet(sheet); err != nil {
		return nil, err
	}
	raw, err := w.f.GetFormControls(sheet)
	if err != nil {
		return nil, fmt.Errorf("pyexcelize: form controls of %q: %w", sheet, err)
	}
	out := make([]FormControl, 0, len(raw))
	for _, c := range raw {
		out = append(out, newFormControl(sheet, c))
	}
	return out, nil
}

// AllFormControls returns the form controls of every sheet in workbook order.
func (w *Workbook) AllFormControls() ([]FormControl, error) {
	if w.Closed() {
		return nil, ErrClosed
	}
	out := []FormControl{}
	for _, name := range w.SheetNames() {
		cs, err := w.FormControls(name)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	return out, nil
}

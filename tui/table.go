package tui

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode"
)

type Cell []string

func visibleLen(s string) (length int) {
	s = removeAnsiEscapes(s)
	for _, char := range s {
		if unicode.IsGraphic(char) {
			length++
		}
	}
	return
}

func removeAnsiEscapes(s string) string {
	return regexp.MustCompile("\033\\[[0-9;]+m").ReplaceAllString(s, "")
}

func (c Cell) Width() int {
	n := 0
	for _, s := range c {
		if l := visibleLen(s); l > n {
			n = l
		}
	}
	return n
}

func (c Cell) Height() int {
	return len(c)
}

func (c Cell) Line(i int) string {
	if i >= len(c) {
		return ""
	}
	return c[i]
}

func ParseCell(s string) Cell {
	l := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	c := Cell(make([]string, len(l)))
	for i, v := range l {
		c[i] = v
	}
	return c
}

type Row []Cell

func (r Row) Width() int {
	w := 0
	if l := len(r); l > 0 {
		w = (l - 1) * 2 // padding
	}
	for _, c := range r {
		w += c.Width()
	}
	return w
}

func (r Row) Height() int {
	n := 0
	for _, c := range r {
		if h := c.Height(); h > n {
			n = h
		}
	}
	return n
}

func ParseRow(ss ...string) Row {
	r := Row(make([]Cell, len(ss)))
	for i, s := range ss {
		r[i] = ParseCell(s)
	}
	return r
}

type Grid struct {
	rows    []Row
	indexed bool

	index    []int
	data     []string
	prepared bool
}

func NewGrid(header ...string) Grid {
	t := Grid{rows: make([]Row, 0)}

	t.rows = append(t.rows, ParseRow(header...))
	ll := make([]string, len(header))
	for i, s := range header {
		ll[i] = strings.Repeat("=", len(s))
	}
	t.rows = append(t.rows, ParseRow(ll...))
	return t
}

func NewIndexedGrid(header ...string) Grid {
	t := NewGrid(header...)
	t.indexed = true
	return t
}

func (t *Grid) Row(vv ...interface{}) {
	ss := make([]string, len(vv))
	for i, v := range vv {
		switch v.(type) {
		case string:
			ss[i] = v.(string)
		default:
			ss[i] = fmt.Sprintf("%v", v)
		}
	}
	t.rows = append(t.rows, ParseRow(ss...))
}

func (t Grid) Height() int {
	h := 0
	for _, r := range t.rows {
		h += r.Height()
	}
	return h
}

// prepare the internal state of the Grid object for display.
func (t *Grid) prepare() {
	if t.prepared {
		return
	}

	g := [][]string{} //Grid
	i := 0
	ww := make([]int, t.Columns())
	ix := make([]int, 0)
	for y, r := range t.rows {
		h := r.Height()
		for j := 0; j < h; j++ {
			// set up R x H new cells in the grid
			g = append(g, make([]string, len(r)))
			ix = append(ix, 0)
		}

		for x, c := range r {
			if w := c.Width(); w > ww[x] {
				ww[x] = w
			}

			if y >= 2 {
				ix[i] = y - 1 // from 1 ... n
			}
			for j := 0; j < h; j++ {
				g[i+j][x] = c.Line(j)
			}
		}
		i += h
	}

	fml := make([]string, len(g)) //Format string for each line
	//Determine the format string for each cell. Start with the expected width of
	// a column, and alter if non-printing characters are found
	var rowLineNum int
	for _, row := range t.rows {
		ffml := make([][]string, row.Height()) //Array of format strings for this row
		//Initialize inner slices to default column width values.
		// This corrects cases for multiple line cells, because it will fill in values for
		// the cells of the grid that don't actually have a cell line associated with it
		for i := range ffml {
			ffml[i] = make([]string, len(row))
			for j := range ffml[i] {
				ffml[i][j] = fmt.Sprintf("%%-%ds", ww[j])
			}
		}
		for cellNum, cell := range row {
			for lineNum, cellLine := range cell {
				offset := len(cellLine) - visibleLen(cellLine)
				ffml[lineNum][cellNum] = fmt.Sprintf("%%-%ds", ww[cellNum]+offset)
			}
		}
		for lineNum, lineFormat := range ffml { //Put each format line into fml
			fml[rowLineNum+lineNum] = strings.Join(lineFormat, "  ") //Format string for an entire line
		}
		rowLineNum += row.Height()
	}

	t.index = ix
	t.data = make([]string, len(g))
	for i, ss := range g {
		ii := make([]interface{}, len(ss))
		for n, s := range ss {
			ii[n] = s
		}
		t.data[i] = strings.TrimRight(fmt.Sprintf(fml[i], ii...), " ") + "\n"
	}

	t.prepared = true
}

func (t *Grid) Columns() int {
	n := 0
	for _, r := range t.rows {
		if l := len(r); l > n {
			n = l
		}
	}
	return n
}

func (t *Grid) Line(n int) string {
	t.prepare()

	s := t.data[n]
	if t.indexed {
		if i := t.index[n]; i != 0 {
			return fmt.Sprintf("%4d) %s", i, s)
		}
		return "      " + s
	}
	return s
}

func (t *Grid) Lines() []string {
	ll := make([]string, len(t.data))
	for i := range t.data {
		ll[i] = t.Line(i)
	}
	return ll
}

type Table struct {
	objects []interface{}
	grid    Grid
}

func NewTable(header ...string) Table {
	return Table{
		objects: make([]interface{}, 0),
		grid:    NewGrid(header...),
	}
}

func (t *Table) Rows() int {
	return len(t.objects)
}

func (t *Table) Object(i int) interface{} {
	if i < 0 || i >= len(t.objects) {
		return nil
	}
	return t.objects[i]
}

func (t *Table) Row(object interface{}, cells ...interface{}) {
	t.objects = append(t.objects, object)
	t.grid.Row(cells...)
}

func (t *Table) Output(out io.Writer) {
	t.grid.prepare()
	for _, s := range t.grid.Lines() {
		fmt.Fprintf(out, "%s", s)
	}
}

func (t *Table) OutputWithIndices(out io.Writer) {
	tf := t.grid.indexed
	t.grid.indexed = true
	t.Output(out)
	t.grid.indexed = tf
}

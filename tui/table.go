package tui

import (
	"fmt"
	"io"
	"strings"
)

type Cell []string

func (c Cell) Width() int {
	n := 0
	for _, s := range c {
		if l := len(s); l > n {
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

	g := [][]string{}
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

	ff := make([]string, len(ww))
	for i, w := range ww {
		ff[i] = fmt.Sprintf("%%-%ds", w)
	}
	f := strings.Join(ff, "  ")

	t.index = ix
	t.data = make([]string, len(g))
	for i, ss := range g {
		ii := make([]interface{}, len(ss))
		for n, s := range ss {
			ii[n] = s
		}
		t.data[i] = strings.TrimRight(fmt.Sprintf(f, ii...), " ") + "\n"
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

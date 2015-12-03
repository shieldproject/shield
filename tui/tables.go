package tui

import (
	"fmt"
	"io"
	"strings"
)

type Table struct {
	Width int
	Max   []int

	header []interface{}
	line   []interface{}
	cells  [][]interface{}
}

func NewTable(header ...string) Table {
	t := Table{
		Width: len(header),
		Max:   make([]int, len(header)),

		header: make([]interface{}, len(header)),
		line:   make([]interface{}, len(header)),
	}
	for i, s := range header {
		t.header[i] = s
		t.line[i] = strings.Repeat("=", len(s))
		t.Max[i] = len(s)
	}

	return t
}

func (t *Table) Row(cells ...interface{}) {
	if len(cells) > t.Width {
		cells = cells[0:t.Width]
	}

	row := make([]interface{}, t.Width)
	for i, v := range cells {
		s := fmt.Sprintf("%v", v)
		if t.Max[i] < len(s) {
			t.Max[i] = len(s)
		}
		row[i] = s
	}

	for i := len(cells); i < t.Width; i++ {
		row[i] = ""
	}
	t.cells = append(t.cells, row)
}

func (t *Table) Output(out io.Writer) {
	formats := make([]string, t.Width)
	for i, width := range t.Max {
		formats[i] = fmt.Sprintf("%%-%ds", width)
	}
	format := strings.Join(formats, "   ") + "\n"

	fmt.Fprintf(out, format, t.header...)
	fmt.Fprintf(out, format, t.line...)
	for _, row := range t.cells {
		fmt.Fprintf(out, format, row...)
	}
}

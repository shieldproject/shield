package tui

import (
	"fmt"
	"io"
	"strings"
)

type Report struct {
	values [][]string
	width  int
}

func NewReport() Report {
	return Report{}
}

func (r *Report) Add(key string, value string) {
	if r.width < len(key) {
		r.width = len(key)
	}

	v := strings.Split(value, "\n")
	r.values = append(r.values, []string{key, v[0]})
	for _, s := range v[1:] {
		r.values = append(r.values, []string{"", s})
	}
}

func (r *Report) Break() {
	r.values = append(r.values, []string{"", ""})
}

func (r *Report) Output(out io.Writer) {
	keyf := fmt.Sprintf("%%-%ds %%s\n", r.width+1)
	blank := strings.Repeat(" ", r.width+2)

	for _, p := range r.values {
		if len(p) != 2 {
			fmt.Printf("got the wrong num balues\n")
		}
		if p[0] != "" {
			fmt.Fprintf(out, keyf, p[0]+":", p[1])
		} else {
			fmt.Fprintf(out, "%s%s\n", blank, p[1])
		}
	}
}

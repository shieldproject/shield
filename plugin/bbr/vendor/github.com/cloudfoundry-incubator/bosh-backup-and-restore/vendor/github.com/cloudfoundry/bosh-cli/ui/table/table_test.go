package table_test

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("Table", func() {
	var (
		buf *bytes.Buffer
	)

	BeforeEach(func() {
		buf = bytes.NewBufferString("")
	})

	Describe("Print", func() {
		It("prints a table in default formatting (borders, empties, etc.)", func() {
			table := Table{
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				Notes: []string{"note1", "note2"},
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(strings.Replace(`
Header1  Header2  +
r1c1     r1c2     +
r2c1     r2c2     +

note1
note2

2 things
`, "+", "", -1)))
		})

		It("prints a table with header if Header is specified", func() {
			table := Table{
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
r1c1...|r1c2...|
r2c1...|r2c2...|

note1
note2

2 things
`))
		})

		It("prints a table without number of records if content is not specified", func() {
			table := Table{
				Content: "",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
r1c1...|r1c2...|
r2c1...|r2c2...|

note1
note2
`))
		})

		It("prints a table sorted based on SortBy", func() {
			table := Table{
				SortBy: []ColumnSort{{Column: 1}, {Column: 0, Asc: true}},

				Rows: [][]Value{
					{ValueString{"a"}, ValueInt{-1}},
					{ValueString{"b"}, ValueInt{0}},
					{ValueString{"d"}, ValueInt{20}},
					{ValueString{"c"}, ValueInt{20}},
					{ValueString{"d"}, ValueInt{100}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
d|100|
c|20.|
d|20.|
b|0..|
a|-1.|
`))
		})

		It("prints a table without a header if Header is not specified", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r2c1|r2c2|
`))
		})

		It("prints a table with a title and a header", func() {
			table := Table{
				Title:   "Title",
				Content: "things",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				Notes:         []string{"note1", "note2"},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Title

Header1|Header2|
r1c1...|r1c2...|
r2c1...|r2c2...|

note1
note2

2 things
`))
		})

		Context("when sections are provided", func() {
			It("prints a table *without* sections for now", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							Rows: [][]Value{
								{ValueString{"r1c1"}, ValueString{"r1c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{"r2c1"}, ValueString{"r2c2"}},
							},
						},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r2c1|r2c2|
`))
			})

			It("prints a table with first column set", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							FirstColumn: ValueString{"r1c1"},

							Rows: [][]Value{
								{ValueString{""}, ValueString{"r1c2"}},
								{ValueString{""}, ValueString{"r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{"r3c1"}, ValueString{"r3c2"}},
							},
						},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
~...|r2c2|
r3c1|r3c2|
`))
			})

			It("prints a table with first column filled for all rows when option is set", func() {
				table := Table{
					Content: "things",
					Sections: []Section{
						{
							FirstColumn: ValueString{"r1c1"},
							Rows: [][]Value{
								{ValueString{""}, ValueString{"r1c2"}},
								{ValueString{""}, ValueString{"r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{"r3c1"}, ValueString{"r3c2"}},
							},
						},
						{
							FirstColumn: ValueString{"r4c1"},
							Rows: [][]Value{
								{ValueString{""}, ValueString{"r4c2"}},
								{ValueString{""}, ValueString{"r5c2"}},
								{ValueString{""}, ValueString{"r6c2"}},
							},
						},
					},
					FillFirstColumn: true,
					BackgroundStr:   ".",
					BorderStr:       "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2|
r1c1|r2c2|
r3c1|r3c2|
r4c1|r4c2|
r4c1|r5c2|
r4c1|r6c2|
`))
			})

			It("prints a footer including the counts for rows in sections", func() {
				table := Table{
					Content: "things",
					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
					},
					Sections: []Section{
						{
							FirstColumn: ValueString{"s1c1"},
							Rows: [][]Value{
								{ValueString{""}, ValueString{"s1r1c2"}},
								{ValueString{""}, ValueString{"s1r2c2"}},
							},
						},
						{
							Rows: [][]Value{
								{ValueString{"r3c1"}, ValueString{"r3c2"}},
							},
						},
					},
					Rows: [][]Value{
						{ValueString{"r4c1"}, ValueString{"r4c2"}},
					},
					FillFirstColumn: true,
					BackgroundStr:   ".",
					BorderStr:       "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
s1c1...|s1r1c2.|
s1c1...|s1r2c2.|
r3c1...|r3c2...|
r4c1...|r4c2...|

4 things
`))
			})
		})

		It("prints values in table that span multiple lines", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2.1\nr1c2.2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
r1c1|r1c2.1|
....|r1c2.2|
r2c1|r2c2..|
`))
		})

		It("removes duplicate values in the first column", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{"dup"}, ValueString{"dup"}},
					{ValueString{"dup"}, ValueString{"dup"}},
					{ValueString{"dup2"}, ValueString{"dup"}},
					{ValueString{"dup2"}, ValueString{"dup"}},
					{ValueString{"other"}, ValueString{"dup"}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|dup|
~....|dup|
dup2.|dup|
~....|dup|
other|dup|
`))
		})

		It("does not removes duplicate values in the first column if FillFirstColumn is true", func() {
			table := Table{
				Content: "things",

				Rows: [][]Value{
					{ValueString{"dup"}, ValueString{"dup"}},
					{ValueString{"dup"}, ValueString{"dup"}},
					{ValueString{"dup2"}, ValueString{"dup"}},
					{ValueString{"dup2"}, ValueString{"dup"}},
					{ValueString{"other"}, ValueString{"dup"}},
				},

				FillFirstColumn: true,
				BackgroundStr:   ".",
				BorderStr:       "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|dup|
dup..|dup|
dup2.|dup|
dup2.|dup|
other|dup|
`))
		})

		It("removes duplicate values in the first column even with sections", func() {
			table := Table{
				Content: "things",

				Sections: []Section{
					{
						FirstColumn: ValueString{"dup"},
						Rows: [][]Value{
							{ValueNone{}, ValueString{"dup"}},
							{ValueNone{}, ValueString{"dup"}},
						},
					},
					{
						FirstColumn: ValueString{"dup2"},
						Rows: [][]Value{
							{ValueNone{}, ValueString{"dup"}},
							{ValueNone{}, ValueString{"dup"}},
						},
					},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup.|dup|
~...|dup|
dup2|dup|
~...|dup|
`))
		})

		It("removes duplicate values in the first column after sorting", func() {
			table := Table{
				Content: "things",

				SortBy: []ColumnSort{{Column: 1, Asc: true}},

				Rows: [][]Value{
					{ValueString{"dup"}, ValueInt{1}},
					{ValueString{"dup2"}, ValueInt{3}},
					{ValueString{"dup"}, ValueInt{2}},
					{ValueString{"dup2"}, ValueInt{4}},
					{ValueString{"other"}, ValueInt{5}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
dup..|1|
~....|2|
dup2.|3|
~....|4|
other|5|
`))
		})

		It("prints empty values as dashes", func() {
			table := Table{
				Rows: [][]Value{
					{ValueString{""}, ValueNone{}},
					{ValueString{""}, ValueNone{}},
				},

				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
-|-|
~|-|
`))
		})

		It("prints empty tables without rows and section", func() {
			table := Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}
			table.Print(buf)
			Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|

0 content
`))
		})

		Context("table has Transpose:true", func() {
			It("prints as transposed table", func() {
				table := Table{
					Content: "errands",
					Header: []Header{
						NewHeader("Header1"),
						NewHeader("OtherHeader2"),
						NewHeader("Header3"),
					},
					Rows: [][]Value{
						{ValueString{"r1c1"}, ValueString{"longr1c2"}, ValueString{"r1c3"}},
						{ValueString{"r2c1"}, ValueString{"r2c2"}, ValueString{"r2c3"}},
					},
					BackgroundStr: ".",
					BorderStr:     "|",
					Transpose:     true,
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1.....|r1c1....|
OtherHeader2|longr1c2|
Header3.....|r1c3....|

Header1.....|r2c1....|
OtherHeader2|r2c2....|
Header3.....|r2c3....|

2 errands
`))
			})

			It("prints a filtered transposed table", func() {
				nonVisibleHeader := NewHeader("Header3")
				nonVisibleHeader.Hidden = true

				table := Table{
					Content: "errands",

					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
						nonVisibleHeader,
					},
					Rows: [][]Value{
						{ValueString{"v1"}, ValueString{"v2"}, ValueString{"v3"}},
					},
					BorderStr: "|",
					Transpose: true,
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|v1|
Header2|v2|

1 errands
`))
			})

			Context("when table also has a SortBy value set", func() {
				It("prints as transposed table with sections sorted by the SortBy", func() {
					table := Table{
						Content: "errands",
						Header: []Header{
							NewHeader("Header1"),
							NewHeader("OtherHeader2"),
							NewHeader("Header3"),
						},
						Rows: [][]Value{
							{ValueString{"r1c1"}, ValueString{"longr1c2"}, ValueString{"r1c3"}},
							{ValueString{"r2c1"}, ValueString{"r2c2"}, ValueString{"r2c3"}},
						},
						SortBy: []ColumnSort{
							{Column: 0, Asc: true},
						},
						BackgroundStr: ".",
						BorderStr:     "|",
						Transpose:     true,
					}
					table.Print(buf)
					Expect("\n" + buf.String()).To(Equal(`
Header1.....|r1c1....|
OtherHeader2|longr1c2|
Header3.....|r1c3....|

Header1.....|r2c1....|
OtherHeader2|r2c2....|
Header3.....|r2c3....|

2 errands
`))
				})
			})
		})

		Context("when column filtering is used", func() {
			It("prints all non-filtered out columns", func() {
				nonVisibleHeader := NewHeader("Header3")
				nonVisibleHeader.Hidden = true

				table := Table{
					Content: "content",

					Header: []Header{
						NewHeader("Header1"),
						NewHeader("Header2"),
						nonVisibleHeader,
					},
					Rows: [][]Value{
						{ValueString{"v1"}, ValueString{"v2"}, ValueString{"v3"}},
					},
					BorderStr: "|",
				}
				table.Print(buf)
				Expect("\n" + buf.String()).To(Equal(`
Header1|Header2|
v1     |v2     |

1 content
`))
			})
		})
	})

	Describe("AddColumn", func() {
		It("returns an updated table with the new column", func() {
			table := Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
				},
				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}},
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}

			newTable := table.AddColumn("Header3", []Value{ValueString{"r1c3"}, ValueString{"r2c3"}})
			Expect(newTable).To(Equal(Table{
				Content: "content",
				Header: []Header{
					NewHeader("Header1"),
					NewHeader("Header2"),
					NewHeader("Header3"),
				},
				Rows: [][]Value{
					{ValueString{"r1c1"}, ValueString{"r1c2"}, ValueString{"r1c3"}},
					{ValueString{"r2c1"}, ValueString{"r2c2"}, ValueString{"r2c3"}},
				},
				BackgroundStr: ".",
				BorderStr:     "|",
			}))
		})
	})
})

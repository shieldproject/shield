package table_test

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-cli/ui/table"
)

var _ = Describe("Sorting", func() {
	It("sorts by single column in asc order", func() {
		sortBy := []ColumnSort{{Column: 0, Asc: true}}
		rows := [][]Value{
			{ValueString{"b"}, ValueString{"x"}},
			{ValueString{"a"}, ValueString{"y"}},
		}

		sort.Sort(Sorting{sortBy, rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{"a"}, ValueString{"y"}},
			{ValueString{"b"}, ValueString{"x"}},
		}))
	})

	It("sorts by single column in desc order", func() {
		sortBy := []ColumnSort{{Column: 0, Asc: false}}
		rows := [][]Value{
			{ValueString{"a"}, ValueString{"y"}},
			{ValueString{"b"}, ValueString{"x"}},
		}

		sort.Sort(Sorting{sortBy, rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{"b"}, ValueString{"x"}},
			{ValueString{"a"}, ValueString{"y"}},
		}))
	})

	It("sorts by multiple columns in asc order", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueString{"b"}, ValueString{"z"}, ValueString{"2"}},
			{ValueString{"a"}, ValueString{"x"}, ValueString{"1"}},
			{ValueString{"b"}, ValueString{"y"}, ValueString{"2"}},
			{ValueString{"c"}, ValueString{"t"}, ValueString{"0"}},
		}

		sort.Sort(Sorting{sortBy, rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{"a"}, ValueString{"x"}, ValueString{"1"}},
			{ValueString{"b"}, ValueString{"y"}, ValueString{"2"}},
			{ValueString{"b"}, ValueString{"z"}, ValueString{"2"}},
			{ValueString{"c"}, ValueString{"t"}, ValueString{"0"}},
		}))
	})

	It("sorts by multiple columns in asc and desc order", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: false},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueString{"b"}, ValueString{"z"}, ValueString{"2"}},
			{ValueString{"a"}, ValueString{"x"}, ValueString{"1"}},
			{ValueString{"b"}, ValueString{"y"}, ValueString{"2"}},
			{ValueString{"c"}, ValueString{"t"}, ValueString{"0"}},
		}

		sort.Sort(Sorting{sortBy, rows})

		Expect(rows).To(Equal([][]Value{
			{ValueString{"c"}, ValueString{"t"}, ValueString{"0"}},
			{ValueString{"b"}, ValueString{"y"}, ValueString{"2"}},
			{ValueString{"b"}, ValueString{"z"}, ValueString{"2"}},
			{ValueString{"a"}, ValueString{"x"}, ValueString{"1"}},
		}))
	})

	It("sorts real values (e.g. suffix does not count)", func() {
		sortBy := []ColumnSort{
			{Column: 0, Asc: true},
			{Column: 1, Asc: true},
		}

		rows := [][]Value{
			{ValueSuffix{ValueString{"a"}, "b"}, ValueString{"x"}},
			{ValueSuffix{ValueString{"a"}, "a"}, ValueString{"y"}},
		}

		sort.Sort(Sorting{sortBy, rows})

		Expect(rows).To(Equal([][]Value{
			{ValueSuffix{ValueString{"a"}, "b"}, ValueString{"x"}},
			{ValueSuffix{ValueString{"a"}, "a"}, ValueString{"y"}},
		}))
	})
})

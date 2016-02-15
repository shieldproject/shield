package tui_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/starkandwayne/shield/tui"
)

var _ = Describe("Cell Management", func() {
	Context("with the empty string", func() {
		c := tui.ParseCell("")

		It("should calculate the width as 0", func() {
			Ω(c.Width()).Should(Equal(0))
		})
		It("should calculate the height as 1", func() {
			Ω(c.Height()).Should(Equal(1))
		})
		It("should return the empty string for all line indices", func() {
			Ω(c.Line(0)).Should(Equal(""))
			Ω(c.Line(1)).Should(Equal(""))
			Ω(c.Line(9)).Should(Equal(""))
		})
	})

	Context("with a single-line string", func() {
		c := tui.ParseCell("hello")

		It("should calculate the width as the length of the string", func() {
			Ω(c.Width()).Should(Equal(len("hello")))
		})
		It("should calculate the height as 1", func() {
			Ω(c.Height()).Should(Equal(1))
		})
		It("should return the original string for line 0", func() {
			Ω(c.Line(0)).Should(Equal("hello"))
		})
		It("should return the empty string for all line indices > 0", func() {
			Ω(c.Line(1)).Should(Equal(""))
			Ω(c.Line(5)).Should(Equal(""))
			Ω(c.Line(9)).Should(Equal(""))
		})
	})

	Context("with a newline-terminated single-line string", func() {
		c := tui.ParseCell("hello\n")

		It("should calculate the width as the length of the string", func() {
			Ω(c.Width()).Should(Equal(len("hello")))
		})
		It("should calculate the height as 1", func() {
			Ω(c.Height()).Should(Equal(1))
		})
		It("should return the original string (without the newline) for line 0", func() {
			Ω(c.Line(0)).Should(Equal("hello"))
		})
		It("should return the empty string for all line indices > 0", func() {
			Ω(c.Line(1)).Should(Equal(""))
			Ω(c.Line(5)).Should(Equal(""))
			Ω(c.Line(9)).Should(Equal(""))
		})
	})

	Context("with a multi-line string", func() {
		c := tui.ParseCell("hi\n" + "hello\n" + "hiya")

		It("should calculate the width as the length of the longest line", func() {
			Ω(c.Width()).Should(Equal(len("hello")))
		})
		It("should calculate the height as the number of lines", func() {
			Ω(c.Height()).Should(Equal(3))
		})
		It("should return the the correct substring (without the newline) for indices 0 - 2", func() {
			Ω(c.Line(0)).Should(Equal("hi"))
			Ω(c.Line(1)).Should(Equal("hello"))
			Ω(c.Line(2)).Should(Equal("hiya"))
		})
		It("should return the empty string for all line indices > 2", func() {
			Ω(c.Line(3)).Should(Equal(""))
			Ω(c.Line(5)).Should(Equal(""))
			Ω(c.Line(9)).Should(Equal(""))
		})
	})

	Context("with a newline-terminated multi-line string", func() {
		c := tui.ParseCell("hi\n" + "hello\n" + "hiya\n")

		It("should calculate the width as the length of the longest line", func() {
			Ω(c.Width()).Should(Equal(len("hello")))
		})
		It("should calculate the height as the number of lines", func() {
			Ω(c.Height()).Should(Equal(3))
		})
		It("should return the the correct substring (without the newline) for indices 0 - 2", func() {
			Ω(c.Line(0)).Should(Equal("hi"))
			Ω(c.Line(1)).Should(Equal("hello"))
			Ω(c.Line(2)).Should(Equal("hiya"))
		})
		It("should return the empty string for all line indices > 2", func() {
			Ω(c.Line(3)).Should(Equal(""))
			Ω(c.Line(5)).Should(Equal(""))
			Ω(c.Line(9)).Should(Equal(""))
		})
	})
})

var _ = Describe("Row Management", func() {
	Context("with no cells", func() {
		r := tui.ParseRow()

		It("should calculate the width as 0", func() {
			Ω(r.Width()).Should(Equal(0))
		})
		It("should calculate the height as 0", func() {
			Ω(r.Height()).Should(Equal(0))
		})
	})

	Context("with a single empty string", func() {
		r := tui.ParseRow("")

		It("should calculate the width as the width of the cell", func() {
			Ω(r.Width()).Should(Equal(0))
		})
		It("should calculate the height as the height of the cell", func() {
			Ω(r.Height()).Should(Equal(1))
		})
	})

	Context("with a solitary, single-line string", func() {
		r := tui.ParseRow("hello")

		It("should calculate the width as the width of the cell", func() {
			Ω(r.Width()).Should(Equal(5))
		})
		It("should calculate the height as the height of the cell", func() {
			Ω(r.Height()).Should(Equal(1))
		})
	})

	Context("with a mix of different line-lengths and line-counts", func() {
		r := tui.ParseRow(
			"To a Mouse", // 10

			"Wee, sleekit, cowrin, tim'rous beastie,\n"+ // 39
				"O, what a panic's in thy breastie!\n",

			"Thou need na start awa sae hasty,\n"+
				"Wi' bickering brattle!\n"+
				"I wad be laith to rin an' chase thee,\n"+ // 37
				"Wi' murdering pattle!\n",
		)

		It("should calculate the width as the sum of the cell widths, plus padding", func() {
			Ω(r.Width()).Should(Equal(10 + 2 + 39 + 2 + 37))
		})
		It("should calculate the height as the height of the tallest cell", func() {
			Ω(r.Height()).Should(Equal(4))
		})
	})
})

var _ = Describe("Table Management", func() {
	Context("with a 1x1 configuration", func() {
		t := tui.NewGrid("header")
		t.Row("hello")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(3))
			Ω(t.Line(0)).Should(Equal("header\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("======\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("hello\n"), `third line`)
		})
	})
	Context("with a 1x3 configuration", func() {
		t := tui.NewGrid("a", "b", "c")
		t.Row("first", "second", "third")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(3))
			Ω(t.Line(0)).Should(Equal("a      b       c\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("=      =       =\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("first  second  third\n"), `third line`)
		})
	})
	Context("with a 4x1 configuration", func() {
		t := tui.NewGrid("Superheroes")
		t.Row("Batman")
		t.Row("Iron Man")
		t.Row("The Green Lantern")
		t.Row("Deadpool(?)")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(6))
			Ω(t.Line(0)).Should(Equal("Superheroes\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("===========\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("Batman\n"), `third line`)
			Ω(t.Line(3)).Should(Equal("Iron Man\n"), `fourth line`)
			Ω(t.Line(4)).Should(Equal("The Green Lantern\n"), `fifth line`)
			Ω(t.Line(5)).Should(Equal("Deadpool(?)\n"), `sixth line`)
		})
	})
	Context("with a 5x2 configuration", func() {
		t := tui.NewGrid("Superhero", "Secret Identity")
		t.Row("Batman", "Bruce Wayne")
		t.Row("Iron Man", "Tony Stark")
		t.Row("The Green Lantern", "Alan / Hal / Guy / Kyle / John")
		t.Row("Deadpool(?)", "Wade Winston Wilson")
		t.Row("The Spectre", "Jim Corrigan")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(7))
			Ω(t.Line(0)).Should(Equal("Superhero          Secret Identity\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("=========          ===============\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("Batman             Bruce Wayne\n"), `third line`)
			Ω(t.Line(3)).Should(Equal("Iron Man           Tony Stark\n"), `fourth line`)
			Ω(t.Line(4)).Should(Equal("The Green Lantern  Alan / Hal / Guy / Kyle / John\n"), `fifth line`)
			Ω(t.Line(5)).Should(Equal("Deadpool(?)        Wade Winston Wilson\n"), `sixth line`)
			Ω(t.Line(6)).Should(Equal("The Spectre        Jim Corrigan\n"), `seventh line`)
		})
	})
	Context("with multi-line cells", func() {
		t := tui.NewGrid("Superhero", "Origin City")
		t.Row("Batman", "Gotham City")
		t.Row("Iron Man", "New York\n(New York, NY)\nunconfirmed")
		t.Row("The Green Lantern\n(Hal Jordan)\n", "Coast City")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(8))
			Ω(t.Line(0)).Should(Equal("Superhero          Origin City\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("=========          ===========\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("Batman             Gotham City\n"), `third line`)
			Ω(t.Line(3)).Should(Equal("Iron Man           New York\n"), `fourth line`)
			Ω(t.Line(4)).Should(Equal("                   (New York, NY)\n"), `fifth line`)
			Ω(t.Line(5)).Should(Equal("                   unconfirmed\n"), `sixth line`)
			Ω(t.Line(6)).Should(Equal("The Green Lantern  Coast City\n"), `seventh line`)
			Ω(t.Line(7)).Should(Equal("(Hal Jordan)\n"), `eighth line`)
		})
	})
	Context("with single-line cells and indexing turned on", func() {
		t := tui.NewIndexedGrid("Superhero", "Origin City")
		t.Row("Batman", "Gotham City")
		t.Row("Iron Man", "New York")
		t.Row("The Green Lantern", "Coast City")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(5))
			Ω(t.Line(0)).Should(Equal("      Superhero          Origin City\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("      =========          ===========\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("   1) Batman             Gotham City\n"), `third line`)
			Ω(t.Line(3)).Should(Equal("   2) Iron Man           New York\n"), `fourth line`)
			Ω(t.Line(4)).Should(Equal("   3) The Green Lantern  Coast City\n"), `fifth line`)
		})
	})
	Context("with multi-line cells and indexing turned on", func() {
		t := tui.NewIndexedGrid("Superhero", "Origin City")
		t.Row("Batman", "Gotham City")
		t.Row("Iron Man", "New York\n(New York, NY)\nunconfirmed")
		t.Row("The Green Lantern\n(Hal Jordan)\n", "Coast City")

		It("should format data properly", func() {
			Ω(t.Height()).Should(Equal(8))
			Ω(t.Line(0)).Should(Equal("      Superhero          Origin City\n"), `first line`)
			Ω(t.Line(1)).Should(Equal("      =========          ===========\n"), `second line`)
			Ω(t.Line(2)).Should(Equal("   1) Batman             Gotham City\n"), `third line`)
			Ω(t.Line(3)).Should(Equal("   2) Iron Man           New York\n"), `fourth line`)
			Ω(t.Line(4)).Should(Equal("                         (New York, NY)\n"), `fifth line`)
			Ω(t.Line(5)).Should(Equal("                         unconfirmed\n"), `sixth line`)
			Ω(t.Line(6)).Should(Equal("   3) The Green Lantern  Coast City\n"), `seventh line`)
			Ω(t.Line(7)).Should(Equal("      (Hal Jordan)\n"), `eighth line`)
		})
	})
})

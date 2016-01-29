package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/starkandwayne/shield/api"
)

var _ = Describe("URL creation", func() {
	It("Should parse URLs properly", func() {
		u, err := ParseURL("http://test.example.com:8081/v1/path?q=string")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(u).ShouldNot(BeNil())
		Ω(u.String()).Should(Equal("http://test.example.com:8081/v1/path?q=string"))
	})

	It("Should be able to add querystring parameters piecemeal", func() {
		u, err := ParseURL("http://test.example.com:8081/v1/path")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(u).ShouldNot(BeNil())

		Ω(u.AddParameter("key", "value")).Should(Succeed())
		Ω(u.String()).Should(Equal("http://test.example.com:8081/v1/path?key=value"))
	})

	It("Should handle all standard data types", func() {
		u, err := ParseURL("http://test.example.com:8081/v1/path")
		Ω(err).ShouldNot(HaveOccurred())
		Ω(u).ShouldNot(BeNil())

		Ω(u.AddParameter("string", "value")).Should(Succeed())
		Ω(u.AddParameter("i8", int8(8))).Should(Succeed())
		Ω(u.AddParameter("i16", int16(16))).Should(Succeed())
		Ω(u.AddParameter("i32", int32(32))).Should(Succeed())
		Ω(u.AddParameter("i64", int64(64))).Should(Succeed())
		Ω(u.AddParameter("u8", uint8(8))).Should(Succeed())
		Ω(u.AddParameter("u16", uint16(16))).Should(Succeed())
		Ω(u.AddParameter("u32", uint32(32))).Should(Succeed())
		Ω(u.AddParameter("u64", uint64(64))).Should(Succeed())
		Ω(u.AddParameter("f32", float32(3.2))).Should(Succeed())
		Ω(u.AddParameter("f64", float64(6.4))).Should(Succeed())
		Ω(u.AddParameter("bool", true)).Should(Succeed())
		Ω(u.AddParameter("notbool", false)).Should(Succeed())
		Ω(u.String()).Should(Equal("http://test.example.com:8081/v1/path?bool=t&f32=3.2&f64=6.4&i16=16&i32=32&i64=64&i8=8&notbool=f&string=value&u16=16&u32=32&u64=64&u8=8"))
	})
})

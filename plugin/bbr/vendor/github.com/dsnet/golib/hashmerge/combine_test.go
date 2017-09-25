// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package hashmerge

import (
	"hash/adler32"
	"hash/crc32"
	"hash/crc64"
	"strings"
	"testing"
)

// shortString truncates long strings into something more human readable.
func shortString(s string) string {
	if len(s) > 220 {
		s = s[:100] + "..." + s[len(s)-100:]
	}
	return s
}

func TestCombineAdler32(t *testing.T) {
	var golden = []struct {
		out uint32
		in  string
	}{
		{0x00000001, ""},
		{0x00620062, "a"},
		{0x012600c4, "ab"},
		{0x024d0127, "abc"},
		{0x03d8018b, "abcd"},
		{0x05c801f0, "abcde"},
		{0x081e0256, "abcdef"},
		{0x0adb02bd, "abcdefg"},
		{0x0e000325, "abcdefgh"},
		{0x118e038e, "abcdefghi"},
		{0x158603f8, "abcdefghij"},
		{0x3f090f02, "Discard medicine more than two years old."},
		{0x46d81477, "He who has a shady past knows that nice guys finish last."},
		{0x40ee0ee1, "I wouldn't marry him with a ten foot pole."},
		{0x16661315, "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
		{0x5b2e1480, "The days of the digital watch are numbered.  -Tom Stoppard"},
		{0x8c3c09ea, "Nepal premier won't resign."},
		{0x45ac18fd, "For every action there is an equal and opposite government program."},
		{0x53c61462, "His money is twice tainted: 'taint yours and 'taint mine."},
		{0x7e511e63, "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
		{0xe4801a6a, "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
		{0x61b507df, "size:  a.out:  bad magic"},
		{0xb8631171, "The major problem is with sendmail.  -Mark Horton"},
		{0x8b5e1904, "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
		{0x7cc6102b, "If the enemy is within range, then so are you."},
		{0x700318e7, "It's well we cannot hear the screams/That we create in others' dreams."},
		{0x1e601747, "You remind me of a TV show, but that's all right: I watch it anyway."},
		{0xb55b0b09, "C is as portable as Stonehedge!!"},
		{0x39111dd0, "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
		{0x91dd304f, "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
		{0x2e5d1316, "How can you write a big system without C++?  -Paul Glick"},
		{0xd0201df6, "'Invariant assertions' is the most elegant programming technique!  -Tom Szymanski"},
		{0x211297c8, strings.Repeat("\xff", 5548) + "8"},
		{0xbaa198c8, strings.Repeat("\xff", 5549) + "9"},
		{0x553499be, strings.Repeat("\xff", 5550) + "0"},
		{0xf0c19abe, strings.Repeat("\xff", 5551) + "1"},
		{0x8d5c9bbe, strings.Repeat("\xff", 5552) + "2"},
		{0x2af69cbe, strings.Repeat("\xff", 5553) + "3"},
		{0xc9809dbe, strings.Repeat("\xff", 5554) + "4"},
		{0x69189ebe, strings.Repeat("\xff", 5555) + "5"},
		{0x86af0001, strings.Repeat("\x00", 1e5)},
		{0x79660b4d, strings.Repeat("a", 1e5)},
		{0x110588ee, strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZ", 1e4)},
	}

	for _, g := range golden {
		var splits = []int{
			0 * (len(g.in) / 1),
			1 * (len(g.in) / 4),
			2 * (len(g.in) / 4),
			3 * (len(g.in) / 4),
			1 * (len(g.in) / 1),
		}

		for _, i := range splits {
			p1, p2 := []byte(g.in[:i]), []byte(g.in[i:])
			in1, in2 := shortString(g.in[:i]), shortString(g.in[i:])
			len2 := int64(len(p2))
			if got := CombineAdler32(adler32.Checksum(p1), adler32.Checksum(p2), len2); got != g.out {
				t.Errorf("CombineAdler32(Checksum(%q), Checksum(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.out)
			}
		}
	}
}

func TestCombineCRC32(t *testing.T) {
	var golden = []struct {
		ieee, castagnoli, koopman uint32
		in                        string
	}{
		{0x00000000, 0x00000000, 0x00000000, ""},
		{0xe8b7be43, 0xc1d04330, 0x0da2aa8a, "a"},
		{0x9e83486d, 0xe2a22936, 0x31ec935a, "ab"},
		{0x352441c2, 0x364b3fb7, 0xba2322ac, "abc"},
		{0xed82cd11, 0x92c80a31, 0xe0a6bcf7, "abcd"},
		{0x8587d865, 0xc450d697, 0xac046415, "abcde"},
		{0x4b8e39ef, 0x53bceff1, 0x7589981b, "abcdef"},
		{0x312a6aa6, 0xe627f441, 0x7999acb5, "abcdefg"},
		{0xaeef2a50, 0x0a9421b7, 0xd5cc0e40, "abcdefgh"},
		{0x8da988af, 0x2ddc99fc, 0x39080d0d, "abcdefghi"},
		{0x3981703a, 0xe6599437, 0xd6205881, "abcdefghij"},
		{0x6b9cdfe7, 0xb2cc01fe, 0x418f6bac, "Discard medicine more than two years old."},
		{0xc90ef73f, 0x0e28207f, 0x847e1e04, "He who has a shady past knows that nice guys finish last."},
		{0xb902341f, 0xbe93f964, 0x606bf5a6, "I wouldn't marry him with a ten foot pole."},
		{0x042080e8, 0x9e3be0c3, 0x1521d7b7, "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
		{0x154c6d11, 0xf505ef04, 0xe238d024, "The days of the digital watch are numbered.  -Tom Stoppard"},
		{0x4c418325, 0x85d3dc82, 0x5423e28a, "Nepal premier won't resign."},
		{0x33955150, 0xc5142380, 0x97f7c3a6, "For every action there is an equal and opposite government program."},
		{0x26216a4b, 0x75eb77dd, 0xe4543ac6, "His money is twice tainted: 'taint yours and 'taint mine."},
		{0x1abbe45e, 0x91ebe9f7, 0x48ec4d9a, "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
		{0xc89a94f7, 0xf0b1168e, 0xc75afda4, "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
		{0xab3abe14, 0x572b74e2, 0x6db40154, "size:  a.out:  bad magic"},
		{0xbab102b6, 0x8a58a6d5, 0x4c148ba0, "The major problem is with sendmail.  -Mark Horton"},
		{0x999149d7, 0x9c426c50, 0x9be6c237, "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
		{0x6d52a33c, 0x735400a4, 0x52f8abfc, "If the enemy is within range, then so are you."},
		{0x90631e8d, 0xbec49c95, 0xf98e0b1d, "It's well we cannot hear the screams/That we create in others' dreams."},
		{0x78309130, 0xa95a2079, 0x6a1d5514, "You remind me of a TV show, but that's all right: I watch it anyway."},
		{0x7d0a377f, 0xde2e65c5, 0xd88bc947, "C is as portable as Stonehedge!!"},
		{0x8c79fd79, 0x297a88ed, 0x5e625378, "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
		{0xa20b7167, 0x66ed1d8b, 0xbd1004ed, "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
		{0x8e0bb443, 0xdcded527, 0xd4575591, "How can you write a big system without C++?  -Paul Glick"},
	}

	var ChecksumIEEE = func(data []byte) uint32 {
		return crc32.ChecksumIEEE(data)
	}
	var ChecksumCastagnoli = func(data []byte) uint32 {
		return crc32.Checksum(data, crc32.MakeTable(crc32.Castagnoli))
	}
	var ChecksumKoopman = func(data []byte) uint32 {
		return crc32.Checksum(data, crc32.MakeTable(crc32.Koopman))
	}

	for _, g := range golden {
		var splits = []int{
			0 * (len(g.in) / 1),
			1 * (len(g.in) / 4),
			2 * (len(g.in) / 4),
			3 * (len(g.in) / 4),
			1 * (len(g.in) / 1),
		}

		for _, i := range splits {
			p1, p2 := []byte(g.in[:i]), []byte(g.in[i:])
			in1, in2 := g.in[:i], g.in[i:]
			len2 := int64(len(p2))
			if got := CombineCRC32(crc32.IEEE, ChecksumIEEE(p1), ChecksumIEEE(p2), len2); got != g.ieee {
				t.Errorf("CombineCRC32(IEEE, ChecksumIEEE(%q), ChecksumIEEE(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.ieee)
			}
			if got := CombineCRC32(crc32.Castagnoli, ChecksumCastagnoli(p1), ChecksumCastagnoli(p2), len2); got != g.castagnoli {
				t.Errorf("CombineCRC32(Castagnoli, ChecksumCastagnoli(%q), ChecksumCastagnoli(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.castagnoli)
			}
			if got := CombineCRC32(crc32.Koopman, ChecksumKoopman(p1), ChecksumKoopman(p2), len2); got != g.koopman {
				t.Errorf("CombineCRC32(Koopman, ChecksumKoopman(%q), ChecksumKoopman(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.koopman)
			}
		}
	}
}

func TestCombineCRC64(t *testing.T) {
	var golden = []struct {
		iso, ecma uint64
		in        string
	}{
		{0x0000000000000000, 0x0000000000000000, ""},
		{0x3420000000000000, 0x330284772e652b05, "a"},
		{0x36c4200000000000, 0xbc6573200e84b046, "ab"},
		{0x3776c42000000000, 0x2cd8094a1a277627, "abc"},
		{0x336776c420000000, 0x3c9d28596e5960ba, "abcd"},
		{0x32d36776c4200000, 0x040bdf58fb0895f2, "abcde"},
		{0x3002d36776c42000, 0xd08e9f8545a700f4, "abcdef"},
		{0x31b002d36776c420, 0xec20a3a8cc710e66, "abcdefg"},
		{0x0e21b002d36776c4, 0x67b4f30a647a0c59, "abcdefgh"},
		{0x8b6e21b002d36776, 0x9966f6c89d56ef8e, "abcdefghi"},
		{0x7f5b6e21b002d367, 0x32093a2ecd5773f4, "abcdefghij"},
		{0x8ec0e7c835bf9cdf, 0x8a0825223ea6d221, "Discard medicine more than two years old."},
		{0xc7db1759e2be5ab4, 0x8562c0ac2ab9a00d, "He who has a shady past knows that nice guys finish last."},
		{0xfbf9d9603a6fa020, 0x3ee2a39c083f38b4, "I wouldn't marry him with a ten foot pole."},
		{0xeafc4211a6daa0ef, 0x1f603830353e518a, "Free! Free!/A trip/to Mars/for 900/empty jars/Burma Shave"},
		{0x3e05b21c7a4dc4da, 0x02fd681d7b2421fd, "The days of the digital watch are numbered.  -Tom Stoppard"},
		{0x5255866ad6ef28a6, 0x790ef2b16a745a41, "Nepal premier won't resign."},
		{0x8a79895be1e9c361, 0x3ef8f06daccdcddf, "For every action there is an equal and opposite government program."},
		{0x8878963a649d4916, 0x049e41b2660b106d, "His money is twice tainted: 'taint yours and 'taint mine."},
		{0xa7b9d53ea87eb82f, 0x561cc0cfa235ac68, "There is no reason for any individual to have a computer in their home. -Ken Olsen, 1977"},
		{0xdb6805c0966a2f9c, 0xd4fe9ef082e69f59, "It's a tiny change to the code and not completely disgusting. - Bob Manchek"},
		{0xf3553c65dacdadd2, 0xe3b5e46cd8d63a4d, "size:  a.out:  bad magic"},
		{0x9d5e034087a676b9, 0x865aaf6b94f2a051, "The major problem is with sendmail.  -Mark Horton"},
		{0xa6db2d7f8da96417, 0x7eca10d2f8136eb4, "Give me a rock, paper and scissors and I will move the world.  CCFestoon"},
		{0x325e00cd2fe819f9, 0xd7dd118c98e98727, "If the enemy is within range, then so are you."},
		{0x88c6600ce58ae4c6, 0x70fb33c119c29318, "It's well we cannot hear the screams/That we create in others' dreams."},
		{0x28c4a3f3b769e078, 0x57c891e39a97d9b7, "You remind me of a TV show, but that's all right: I watch it anyway."},
		{0xa698a34c9d9f1dca, 0xa1f46ba20ad06eb7, "C is as portable as Stonehedge!!"},
		{0xf6c1e2a8c26c5cfc, 0x7ad25fafa1710407, "Even if I could be Shakespeare, I think I should still choose to be Faraday. - A. Huxley"},
		{0x0d402559dfe9b70c, 0x73cef1666185c13f, "The fugacity of a constituent in a mixture of gases at a given temperature is proportional to its mole fraction.  Lewis-Randall Rule"},
		{0xdb6efff26aa94946, 0xb41858f73c389602, "How can you write a big system without C++?  -Paul Glick"},
	}

	var ChecksumISO = func(data []byte) uint64 {
		return crc64.Checksum(data, crc64.MakeTable(crc64.ISO))
	}
	var ChecksumECMA = func(data []byte) uint64 {
		return crc64.Checksum(data, crc64.MakeTable(crc64.ECMA))
	}

	for _, g := range golden {
		var splits = []int{
			0 * (len(g.in) / 1),
			1 * (len(g.in) / 4),
			2 * (len(g.in) / 4),
			3 * (len(g.in) / 4),
			1 * (len(g.in) / 1),
		}

		for _, i := range splits {
			p1, p2 := []byte(g.in[:i]), []byte(g.in[i:])
			in1, in2 := g.in[:i], g.in[i:]
			len2 := int64(len(p2))
			if got := CombineCRC64(crc64.ISO, ChecksumISO(p1), ChecksumISO(p2), len2); got != g.iso {
				t.Errorf("CombineCRC64(ISO, ChecksumISO(%q), ChecksumISO(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.iso)
			}
			if got := CombineCRC64(crc64.ECMA, ChecksumECMA(p1), ChecksumECMA(p2), len2); got != g.ecma {
				t.Errorf("CombineCRC64(ECMA, ChecksumECMA(%q), ChecksumECMA(%q), %d) = 0x%x, want 0x%x",
					in1, in2, len2, got, g.ecma)
			}
		}
	}
}

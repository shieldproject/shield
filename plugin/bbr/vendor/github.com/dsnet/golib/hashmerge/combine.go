// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

// Package hashmerge provides functionality for merging hashes.
package hashmerge

// The origin of the CombineAdler32, CombineCRC32, and CombineCRC64 functions
// in this package is the adler32_combine, crc32_combine, gf2_matrix_times,
// and gf2_matrix_square functions found in the zlib library and was translated
// from C to Go. Thanks goes to the authors of zlib:
//	Mark Adler and Jean-loup Gailly.
//
// See the following:
//	http://www.zlib.net/
//	https://github.com/madler/zlib/blob/master/adler32.c
//	https://github.com/madler/zlib/blob/master/crc32.c
//	https://stackoverflow.com/questions/23122312/crc-calculation-of-a-mostly-static-data-stream/23126768#23126768
//
// ====================================================
// Copyright (C) 1995-2013 Jean-loup Gailly and Mark Adler
//
// This software is provided 'as-is', without any express or implied
// warranty.  In no event will the authors be held liable for any damages
// arising from the use of this software.
//
// Permission is granted to anyone to use this software for any purpose,
// including commercial applications, and to alter it and redistribute it
// freely, subject to the following restrictions:
//
// 1. The origin of this software must not be misrepresented; you must not
//    claim that you wrote the original software. If you use this software
//    in a product, an acknowledgment in the product documentation would be
//    appreciated but is not required.
// 2. Altered source versions must be plainly marked as such, and must not be
//    misrepresented as being the original software.
// 3. This notice may not be removed or altered from any source distribution.
//
// Jean-loup Gailly        Mark Adler
// jloup@gzip.org          madler@alumni.caltech.edu
// ====================================================

// CombineAdler32 combines two Adler-32 checksums together.
// Let AB be the string concatenation of two strings A and B. Then Combine
// computes the checksum of AB given only the checksum of A, the checksum of B,
// and the length of B:
//	adler32.Checksum(AB) == CombineAdler32(adler32.Checksum(A), adler32.Checksum(B), len(B))
func CombineAdler32(adler1, adler2 uint32, len2 int64) uint32 {
	if len2 < 0 {
		panic("hashmerge: negative length")
	}

	const mod = 65521
	var sum1, sum2, rem uint32
	rem = uint32(len2 % mod)
	sum1 = adler1 & 0xffff
	sum2 = rem * sum1
	sum2 %= mod
	sum1 += (adler2 & 0xffff) + mod - 1
	sum2 += (adler1 >> 16) + (adler2 >> 16) + mod - rem
	if sum1 >= mod {
		sum1 -= mod
	}
	if sum1 >= mod {
		sum1 -= mod
	}
	if sum2 >= mod<<1 {
		sum2 -= mod << 1
	}
	if sum2 >= mod {
		sum2 -= mod
	}
	return sum1 | (sum2 << 16)
}

// CombineCRC32 combines two CRC-32 checksums together.
// Let AB be the string concatenation of two strings A and B. Then Combine
// computes the checksum of AB given only the checksum of A, the checksum of B,
// and the length of B:
//	tab := crc32.MakeTable(poly)
//	crc32.Checksum(AB, tab) == CombineCRC32(poly, crc32.Checksum(A, tab), crc32.Checksum(B, tab), len(B))
func CombineCRC32(poly, crc1, crc2 uint32, len2 int64) uint32 {
	if len2 < 0 {
		panic("hashmerge: negative length")
	}

	// Translation of gf2_matrix_times from zlib.
	var matrixMult = func(mat *[32]uint32, vec uint32) uint32 {
		var sum uint32
		for n := 0; n < 32 && vec > 0; n++ {
			if vec&1 > 0 {
				sum ^= mat[n]
			}
			vec >>= 1
		}
		return sum
	}

	// Translation of gf2_matrix_square from zlib.
	var matrixSquare = func(square, mat *[32]uint32) {
		for n := 0; n < 32; n++ {
			square[n] = matrixMult(mat, mat[n])
		}
	}

	// Even and odd power-of-two zeros operators.
	var even, odd [32]uint32

	// Put operator for one zero bit in odd.
	var row uint32 = 1
	odd[0] = poly
	for n := 1; n < 32; n++ {
		odd[n] = row
		row <<= 1
	}

	// Put operator for two zero bits in even.
	matrixSquare(&even, &odd)

	// Put operator for four zero bits in odd.
	matrixSquare(&odd, &even)

	// Apply len2 zeros to crc1 (first square will put the operator for one
	// zero byte, eight zero bits, in even).
	for {
		// Apply zeros operator for this bit of len2.
		matrixSquare(&even, &odd)
		if len2&1 > 0 {
			crc1 = matrixMult(&even, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}

		// Another iteration of the loop with odd and even swapped.
		matrixSquare(&odd, &even)
		if len2&1 > 0 {
			crc1 = matrixMult(&odd, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}
	}
	return crc1 ^ crc2
}

// CombineCRC64 combines two CRC-64 checksums together.
// Let AB be the string concatenation of two strings A and B. Then Combine
// computes the checksum of AB given only the checksum of A, the checksum of B,
// and the length of B:
//	tab := crc64.MakeTable(poly)
//	crc64.Checksum(AB, tab) == CombineCRC64(poly, crc64.Checksum(A, tab), crc64.Checksum(B, tab), len(B))
func CombineCRC64(poly, crc1, crc2 uint64, len2 int64) uint64 {
	if len2 < 0 {
		panic("hashmerge: negative length")
	}

	// Translation of gf2_matrix_times from zlib.
	var matrixMult = func(mat *[64]uint64, vec uint64) uint64 {
		var sum uint64
		for n := 0; n < 64 && vec > 0; n++ {
			if vec&1 > 0 {
				sum ^= mat[n]
			}
			vec >>= 1
		}
		return sum
	}

	// Translation of gf2_matrix_square from zlib.
	var matrixSquare = func(square, mat *[64]uint64) {
		for n := 0; n < 64; n++ {
			square[n] = matrixMult(mat, mat[n])
		}
	}

	// Even and odd power-of-two zeros operators.
	var even, odd [64]uint64

	// Put operator for one zero bit in odd.
	var row uint64 = 1
	odd[0] = poly
	for n := 1; n < 64; n++ {
		odd[n] = row
		row <<= 1
	}

	// Put operator for two zero bits in even.
	matrixSquare(&even, &odd)

	// Put operator for four zero bits in odd.
	matrixSquare(&odd, &even)

	// Apply len2 zeros to crc1 (first square will put the operator for one
	// zero byte, eight zero bits, in even).
	for {
		// Apply zeros operator for this bit of len2.
		matrixSquare(&even, &odd)
		if len2&1 > 0 {
			crc1 = matrixMult(&even, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}

		// Another iteration of the loop with odd and even swapped.
		matrixSquare(&odd, &even)
		if len2&1 > 0 {
			crc1 = matrixMult(&odd, crc1)
		}
		len2 >>= 1
		if len2 == 0 {
			break
		}
	}
	return crc1 ^ crc2
}

// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package unitconv_test

import (
	"fmt"

	"github.com/dsnet/golib/unitconv"
)

func ExampleAppendPrefix() {
	b1 := []byte("Distance from SF to LA: ")
	b1 = unitconv.AppendPrefix(b1, 616379, unitconv.SI, -1)
	b1 = append(b1, 'm')
	fmt.Println(string(b1))

	b2 := []byte("Capacity of a DVD: ")
	b2 = unitconv.AppendPrefix(b2, 4.7*unitconv.Giga, unitconv.IEC, 2)
	b2 = append(b2, 'B')
	fmt.Println(string(b2))

	// Output:
	// Distance from SF to LA: 616.379km
	// Capacity of a DVD: 4.38GiB
}

func ExampleFormatPrefix() {
	s1 := unitconv.FormatPrefix(unitconv.Tebi, unitconv.SI, 3)
	fmt.Printf("1 tebibyte in SI: %sB\n", s1)

	s2 := unitconv.FormatPrefix(unitconv.Tera, unitconv.IEC, 3)
	fmt.Printf("1 terabyte in IEC: %sB\n", s2)

	// Output:
	// 1 tebibyte in SI: 1.100TB
	// 1 terabyte in IEC: 931.323GiB
}

func ExampleParsePrefix() {
	if s, err := unitconv.ParsePrefix("2.99792458E8", unitconv.AutoParse); err == nil {
		fmt.Printf("Speed of light: %.0fm/s\n", s)
	}

	if s, err := unitconv.ParsePrefix("616.379k", unitconv.SI); err == nil {
		fmt.Printf("Distance from LA to SF: %.0fm\n", s)
	}

	if s, err := unitconv.ParsePrefix("32M", unitconv.Base1024); err == nil {
		fmt.Printf("Max FAT12 partition size: %.0fB\n", s)
	}

	if s, err := unitconv.ParsePrefix("1Ti", unitconv.IEC); err == nil {
		fmt.Printf("Number of bytes in tebibyte: %.0fB\n", s)
	}

	// Output:
	// Speed of light: 299792458m/s
	// Distance from LA to SF: 616379m
	// Max FAT12 partition size: 33554432B
	// Number of bytes in tebibyte: 1099511627776B
}

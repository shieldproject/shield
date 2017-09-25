// Copyright 2015, Joe Tsai. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE.md file.

package unitconv

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func (m Mode) String() string {
	switch m {
	case AutoParse:
		return "AutoParse"
	case SI:
		return "SI"
	case Base1024:
		return "Base1024"
	case IEC:
		return "IEC"
	default:
		return "<unknown>"
	}
}

// factorFloor returns the closest factor for that mode that is below v.
func (m Mode) factorFloor(v float64) float64 {
	exp := math.Log2(v) / math.Log2(m.base())
	factor := math.Pow(m.base(), math.Trunc(exp))
	switch min, max := m.factorRanges(); {
	case factor < min:
		return min
	case factor > max:
		return max
	default:
		return factor
	}
}

// factorRanges returns the minimum and maximum factors for that mode.
func (m Mode) factorRanges() (min, max float64) {
	switch m {
	case SI:
		return scaleSI[0], scaleSI[len(scaleSI)-1]
	case Base1024:
		return scaleIEC[0], scaleIEC[len(scaleIEC)-1]
	case IEC:
		return 1.0, scaleIEC[len(scaleIEC)-1]
	default:
		return math.NaN(), math.NaN()
	}
}

// parseFraction parses the fractional component out of a unit-prefixed string.
// E.g., "123.45Mi" parses as the number 123.45
func parseFraction(t *testing.T, s string) float64 {
	t.Helper()
	fraction, err := strconv.ParseFloat(strings.TrimRight(s, parsePrefixes+"i"), 64)
	if err != nil {
		t.Errorf("unexpected ParseFloat error: %v", err)
	}
	return fraction
}

// TestExact tests round-trip formatting and parsing of exact values.
func TestExact(t *testing.T) {
	t.Run(SI.String(), func(t *testing.T) {
		wantStrs := strings.Split("yzafpnμm.kMGTPEZY", "")
		for _, sign := range []float64{-1, +1} {
			for i, f := range scaleSI {
				want := sign * f

				gotStr := FormatPrefix(want, SI, -1)
				wantStr := fmt.Sprintf("%v%s", sign, strings.Trim(wantStrs[i], "."))
				if gotStr != wantStr {
					t.Errorf("FormatPrefix(%v, %v, -1):\ngot  %q\nwant %q", want, SI, gotStr, wantStr)
				}

				got, err := ParsePrefix(gotStr, SI)
				if got != want || err != nil {
					t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, nil)", gotStr, SI, got, err, want)
				}
			}
		}
	})
	t.Run(Base1024.String(), func(t *testing.T) {
		testExact(t, scaleIEC, Base1024)
	})
	t.Run(IEC.String(), func(t *testing.T) {
		testExact(t, scaleIEC[len(scaleIEC)/2:], IEC)
	})
}

func testExact(t *testing.T, scales []float64, m Mode) {
	wantStrs := strings.Split("yzafpnum.KMGTPEZY", "")
	wantStrs = wantStrs[len(wantStrs)-len(scales):]
	for _, sign := range []float64{-1, +1} {
		for i, f := range scales {
			for j := uint(0); j < 10; j++ {
				want := float64(int(sign)<<j) * f

				gotStr := FormatPrefix(want, m, -1)
				wantStr := fmt.Sprintf("%d%s", int(sign)<<j, strings.Trim(wantStrs[i], "."))
				if m == IEC && i > 0 {
					wantStr += "i"
				}
				if gotStr != wantStr {
					t.Errorf("FormatPrefix(%v, %v, -1):\ngot  %q\nwant %q", want, m, gotStr, wantStr)
				}

				got, err := ParsePrefix(gotStr, m)
				if got != want || err != nil {
					t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, nil)", gotStr, m, got, err, want)
				}
			}
		}
	}
}

// TestBoundary tests round-trip formatting and parsing at unit boundaries.
func TestBoundary(t *testing.T) {
	t.Run(SI.String(), func(t *testing.T) {
		testBoundary(t, scaleSI, prefixes, SI)
	})
	t.Run(Base1024.String(), func(t *testing.T) {
		testBoundary(t, scaleIEC, prefixes, Base1024)
	})
	t.Run(IEC.String(), func(t *testing.T) {
		idx := len(prefixes) / 2
		testBoundary(t, scaleIEC[idx:], prefixes[idx:], IEC)
	})
}

func testBoundary(t *testing.T, scales []float64, prefixes string, m Mode) {
	const errFrac = 1e-12
	for _, sign := range []float64{-1, +1} {
		for _, roundDir := range []float64{math.Inf(-1), math.Inf(+1)} {
			for i, f := range scales {
				want := math.Nextafter(sign*f, sign*roundDir)
				gotStr := FormatPrefix(want, m, -1)
				got, err := ParsePrefix(gotStr, m)

				// Check round-trip was close enough.
				opt := cmpopts.EquateApprox(errFrac, 0)
				if !cmp.Equal(got, want, opt) || err != nil {
					t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, nil)", gotStr, m, got, err, want)
				}

				// Fraction must be either >= 1 or < base.
				fraction := parseFraction(t, gotStr)
				if roundDir < 0 && i == 0 {
					fraction *= m.base()
				}
				fraction = math.Abs(fraction)
				if roundDir < 0 && !(m.base()-errFrac <= fraction && fraction < m.base()) {
					t.Errorf("Abs(ParseFraction(%q)) = %v, want (%v <= got < %v)", gotStr, fraction, m.base()-errFrac, m.base())
				}
				if roundDir > 0 && !(1 <= fraction && fraction < 1+errFrac) {
					t.Errorf("Abs(ParseFraction(%q)) = %v, want (%v <= got < %v)", gotStr, fraction, 1, 1+errFrac)
				}
			}
		}
	}
}

// TestRoundtrip tests formatting and parsing in a round-trip manner.
func TestRoundtrip(t *testing.T) {
	for _, m := range []Mode{SI, Base1024, IEC} {
		t.Run(m.String(), func(t *testing.T) {
			for _, prec := range []int{-1, 0, +1, +2} {
				testRoundtrip(t, m, prec)
			}
		})
	}
}

func testRoundtrip(t *testing.T, m Mode, prec int) {
	// Test for zero, NaN, -Inf, and +Inf.
	for _, want := range []float64{-0.0, +0.0, math.NaN(), math.Inf(-1), math.Inf(+1)} {
		gotStr := FormatPrefix(want, m, prec)
		if wantStr := strconv.FormatFloat(want, 'f', prec, 64); gotStr != wantStr {
			t.Errorf("FormatPrefix(%v, %v, %v):\ngot  %q\nwant %q", want, m, prec, gotStr, wantStr)
		}

		got, err := ParsePrefix(gotStr, m)
		if !cmp.Equal(got, want, cmpopts.EquateNaNs()) || err != nil {
			t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, nil)", gotStr, m, got, err, want)
		}
	}

	// Test for a large range of values.
	for i := -100; i <= +100; i++ {
		want := 1.234567890123456 * math.Pow(10, float64(i))
		gotStr := FormatPrefix(want, m, prec)
		got, err := ParsePrefix(gotStr, m)

		// Ensure that we maintain decent precision.
		opt := cmpopts.EquateApprox(1e-12, m.factorFloor(want)/2)
		if !cmp.Equal(got, want, opt) || err != nil {
			t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, nil)", gotStr, m, got, err, want)
		}

		// Ensure that we choose the best scale if possible.
		if min, max := m.factorRanges(); min <= want && want <= max {
			fraction := parseFraction(t, gotStr)
			if !(1.0 <= fraction && fraction < m.base()) {
				t.Errorf("ParseFraction(%q) = %v, want (1.0 <= got < %v)", gotStr, fraction, m.base())
			}
		}
	}
}

// TestParsePrefix tests parsing of various string inputs.
func TestParsePrefix(t *testing.T) {
	anyError := errors.New("any error")
	tests := []struct {
		in      string
		mode    Mode
		want    float64
		wantErr error
	}{
		{"", SI, 0, anyError},
		{"NaN1M", SI, 0, anyError},
		{"1", IEC, Unit, nil},
		{"1 ", IEC, 0, anyError},
		{"1M", IEC, 0, anyError},
		{"1Mi", SI, 0, anyError},
		{"+1M", Base1024, +Mebi, nil},
		{"-1Mi", Base1024, -Mebi, nil},
		{"+1Mi", Base1024, +Mebi, nil},
		{"1E-3", SI, 0.001, anyError},
		{"1e-3", SI, 0.001, anyError},
		{"1ki", SI, 0, anyError},
		{"1ki", IEC, 0, anyError},
		{"1ki", Base1024, Kibi, nil},
		{"+1ki", Base1024, Kibi, nil},
		{"1μi", SI, 0, anyError},
		{"1μi", IEC, 0, anyError},
		{"1μi", Base1024, 0, anyError},
		{"1k", SI, Kilo, nil},
		{"1k", IEC, 0, anyError},
		{"1k", Base1024, Kibi, nil},
		{"1μ", SI, Micro, nil},
		{"1μ ", SI, 0, anyError},
		{" 1μ", SI, 0, anyError},
		{"+1μ", SI, Micro, nil},
		{"1μ", IEC, 0, anyError},
		{"1μ", Base1024, 1.0 / Mebi, nil},
		{"+1μ", Base1024, 1.0 / Mebi, nil},
		{"1mi", IEC, 0, anyError},
		{"0.000001", SI, Micro, nil},
		{"1000000u", SI, Unit, nil},
		{"1048576", Base1024, Mebi, nil},
		{"1048576Ki", IEC, Gibi, nil},
		{"nAn", SI, math.NaN(), nil},
		{"+nan", Base1024, 0, anyError},
		{"-NAN", IEC, 0, anyError},
		{"INF", SI, math.Inf(+1), nil},
		{"+iNf", Base1024, math.Inf(+1), nil},
		{"-inF", IEC, math.Inf(-1), nil},
		{"", AutoParse, 0, anyError},
		{"123", AutoParse, 123, nil},
		{"123Ki", AutoParse, 123 * Kibi, nil},
		{"123k", AutoParse, 123 * Kilo, nil},
		{"123K", AutoParse, 123 * Kilo, nil},
		{"3Mi", AutoParse, 3 * Mebi, nil},
		{"3M", AutoParse, 3 * Mega, nil},
		{"3E-3", AutoParse, 3E-3, nil},
		{"2E2", AutoParse, 2E2, nil},
	}

	for _, tt := range tests {
		got, gotErr := ParsePrefix(tt.in, tt.mode)
		if !cmp.Equal(got, tt.want, cmpopts.EquateNaNs()) || (gotErr == nil) != (tt.wantErr == nil) {
			t.Errorf("ParsePrefix(%q, %v):\ngot  (%v, %v)\nwant (%v, %v)", tt.in, tt.mode, got, gotErr, tt.want, tt.wantErr)
		}
	}
}

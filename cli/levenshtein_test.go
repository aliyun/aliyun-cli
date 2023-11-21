// Copyright (c) 2013 Kilian Evang and contributors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//
// https://github.com/texttheater/golang-levenshtein
package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testCases = []struct {
	source   string
	target   string
	distance int
	ratio    float64
	script   EditScript
}{
	{
		source:   "",
		target:   "a",
		distance: 1,
		ratio:    0.0,
		script:   EditScript{Ins},
	},
	{
		source:   "a",
		target:   "aa",
		distance: 1,
		ratio:    0.6666666666666666,
		script:   EditScript{Match, Ins},
	},
	{
		source:   "a",
		target:   "aaa",
		distance: 2,
		ratio:    0.5,
		script:   EditScript{Match, Ins, Ins},
	},
	{
		source:   "",
		target:   "",
		distance: 0,
		ratio:    0,
		script:   EditScript{},
	},
	{
		source:   "a",
		target:   "b",
		distance: 2,
		ratio:    0,
		script:   EditScript{Ins, Del},
	},
	{
		source:   "aaa",
		target:   "aba",
		distance: 2,
		ratio:    0.6666666666666666,
		script:   EditScript{Match, Ins, Match, Del},
	},
	{
		source:   "aaa",
		target:   "ab",
		distance: 3,
		ratio:    0.4,
		script:   EditScript{Match, Ins, Del, Del},
	},
	{
		source:   "a",
		target:   "a",
		distance: 0,
		ratio:    1,
		script:   EditScript{Match},
	},
	{
		source:   "ab",
		target:   "ab",
		distance: 0,
		ratio:    1,
		script:   EditScript{Match, Match},
	},
	{
		source:   "a",
		target:   "",
		distance: 1,
		ratio:    0,
		script:   EditScript{Del},
	},
	{
		source:   "aa",
		target:   "a",
		distance: 1,
		ratio:    0.6666666666666666,
		script:   EditScript{Match, Del},
	},
	{
		source:   "aaa",
		target:   "a",
		distance: 2,
		ratio:    0.5,
		script:   EditScript{Match, Del, Del},
	},
	{
		source:   "kitten",
		target:   "sitting",
		distance: 5,
		ratio:    0.6153846153846154,
		script: EditScript{
			Ins,
			Del,
			Match,
			Match,
			Match,
			Ins,
			Del,
			Match,
			Ins,
		},
	},
}

func TestEditOpString(t *testing.T) {
	editOp1 := EditOperation(3)
	assert.Equal(t, "match", editOp1.String())
	editOp1 = EditOperation(0)
	assert.Equal(t, "ins", editOp1.String())
	editOp1 = EditOperation(2)
	assert.Equal(t, "sub", editOp1.String())
	editOp1 = EditOperation(1)
	assert.Equal(t, "del", editOp1.String())
	// editOp1 = EditOperation(4)
	// assert.Equal(t, "del", editOp1.String())
}

func TestDistanceForStrings(t *testing.T) {
	for _, testCase := range testCases {
		distance := DistanceForStrings(
			[]rune(testCase.source),
			[]rune(testCase.target),
			DefaultOptions)
		if distance != testCase.distance {
			t.Log(
				"Distance between",
				testCase.source,
				"and",
				testCase.target,
				"computed as",
				distance,
				", should be",
				testCase.distance)
			t.Fail()
		}
		// DistanceForMatrix(MatrixForStrings()) should calculate the same
		// value as DistanceForStrings.
		distance = DistanceForMatrix(MatrixForStrings(
			[]rune(testCase.source),
			[]rune(testCase.target),
			DefaultOptions))
		if distance != testCase.distance {
			t.Log(
				"Distance between",
				testCase.source,
				"and",
				testCase.target,
				"computed as",
				distance,
				", should be",
				testCase.distance)
			t.Fail()
		}
	}
}

func TestRatio(t *testing.T) {
	for _, testCase := range testCases {
		ratio := RatioForStrings(
			[]rune(testCase.source),
			[]rune(testCase.target),
			DefaultOptions)
		if ratio != testCase.ratio {
			t.Log(
				"Ratio between",
				testCase.source,
				"and",
				testCase.target,
				"computed as",
				ratio,
				", should be",
				testCase.ratio)
			t.Fail()
		}
	}
}

func TestEditScriptForStrings(t *testing.T) {
	for _, testCase := range testCases {
		script := EditScriptForStrings(
			[]rune(testCase.source),
			[]rune(testCase.target),
			DefaultOptions)
		if !equal(script, testCase.script) {
			t.Log(
				"Edit script from",
				testCase.source,
				"to",
				testCase.target,
				"computed as",
				script,
				", should be",
				testCase.script)
			t.Fail()
		}
	}
}

func equal(a, b EditScript) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func ExampleDistanceForStrings() {
	source := "a"
	target := "aa"
	distance := DistanceForStrings([]rune(source), []rune(target), DefaultOptions)
	fmt.Printf(`Distance between "%s" and "%s" computed as %d`, source, target, distance)
	// Output: Distance between "a" and "aa" computed as 1
}

func ExampleWriteMatrix() {
	source := []rune("neighbor")
	target := []rune("Neighbour")
	matrix := MatrixForStrings(source, target, DefaultOptions)
	WriteMatrix(source, target, matrix, os.Stdout)
	// Output:
	//       N  e  i  g  h  b  o  u  r
	//    0  1  2  3  4  5  6  7  8  9
	// n  1  2  3  4  5  6  7  8  9 10
	// e  2  3  2  3  4  5  6  7  8  9
	// i  3  4  3  2  3  4  5  6  7  8
	// g  4  5  4  3  2  3  4  5  6  7
	// h  5  6  5  4  3  2  3  4  5  6
	// b  6  7  6  5  4  3  2  3  4  5
	// o  7  8  7  6  5  4  3  2  3  4
	// r  8  9  8  7  6  5  4  3  4  3
}

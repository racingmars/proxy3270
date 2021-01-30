/*
 * Copyright 2020-2021 by Matthew R. Wilson <mwilson@mattwilson.org>
 *
 * This file is part of proxy3270.
 *
 * proxy3270 is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * proxy3270 is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with proxy3270. If not, see <https://www.gnu.org/licenses/>.
 */

package main

import (
	"testing"
)

func TestWrap(t *testing.T) {
	type TestCase struct {
		Input              string
		LineLength         int
		OutLine1, OutLine2 string
	}

	testCases := []TestCase{
		{"This is a short line", 80, "This is a short line", ""},
		{"", 80, "", ""},
		{"    This is a short line        ", 23, "This is a short line", ""},
		{"Once upon a time, there was a programmer.", 26, "Once upon a time, there", "was a programmer."},
		{"abcdefghijklmnopqrstuvwxyz", 16, "abcdefghijklmnop", "qrstuvwxyz"},
	}

	for i := range testCases {
		line1, line2 := wrapDisclaimer(testCases[i].Input, testCases[i].LineLength)
		if line1 != testCases[i].OutLine1 || line2 != testCases[i].OutLine2 {
			t.Errorf("Input line `%s` incorrect wrapped to `%s` and `%s`; we expected `%s` and `%s`",
				testCases[i].Input, line1, line2, testCases[i].OutLine1, testCases[i].OutLine2)
		}
	}
}

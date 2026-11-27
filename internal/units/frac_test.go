// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package units_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/units"
)

func TestFrac(t *testing.T) {
	f := units.NewFrac
	for _, tc := range []struct {
		name      string
		got, want units.Frac
	}{
		{"lowest terms", f(50, 100), f(1, 2)},
		{"negative denominator", f(3, -6), f(-1, 2)},
		{"zero is the zero value", f(0, 7), units.Frac{}},
		{"whole", units.Whole(3), f(6, 2)},
		{"add", f(1, 8).Add(f(1, 20)), f(7, 40)},
		{"add to zero", units.Frac{}.Add(f(1, 20)), f(1, 20)},
		{"sub", f(1, 8).Sub(f(1, 20)), f(3, 40)},
		{"sub to zero", f(1, 8).Sub(f(2, 16)), units.Frac{}},
		{"mul", f(3, 40).Mul(20), f(3, 2)},
		{"div", f(3, 2).Div(6), f(1, 4)},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, tc.got, tc.want)
		}
	}
	for _, tc := range []struct {
		f           units.Frac
		floor, ceil int
		sign        int
		s           string
	}{
		{units.Frac{}, 0, 0, 0, "0"},
		{f(1, 40), 0, 1, 1, "1/40"},
		{f(55, 2), 27, 28, 1, "55/2"},
		{f(-55, 2), -28, -27, -1, "-55/2"},
		{units.Whole(-3), -3, -3, -1, "-3"},
	} {
		if tc.f.Floor() != tc.floor || tc.f.Ceil() != tc.ceil || tc.f.Sign() != tc.sign || tc.f.String() != tc.s {
			t.Errorf("%v: floor %d, ceil %d, sign %d, %q; want %d, %d, %d, %q",
				tc.f, tc.f.Floor(), tc.f.Ceil(), tc.f.Sign(), tc.f.String(), tc.floor, tc.ceil, tc.sign, tc.s)
		}
	}
	if f(1, 8).Cmp(f(1, 20)) != 1 || f(1, 20).Cmp(f(1, 8)) != -1 || f(2, 16).Cmp(f(1, 8)) != 0 {
		t.Error("Cmp is wrong")
	}
}

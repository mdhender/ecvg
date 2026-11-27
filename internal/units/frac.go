// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package units

import "fmt"

// Frac is an exact amount of mass or volume: a fraction in lowest terms.
// The engine never rounds one; only a report does, when it shows it. The
// zero value is 0, and Fracs compare with ==.
type Frac struct {
	num, den int64 // den is 0 only for 0
}

// Whole returns n as a Frac.
func Whole(n int) Frac {
	return NewFrac(n, 1)
}

// NewFrac returns num/den. It panics if den is 0.
func NewFrac(num, den int) Frac {
	return frac(int64(num), int64(den))
}

func frac(num, den int64) Frac {
	if den == 0 {
		panic("units: fraction with a zero denominator")
	}
	if num == 0 {
		return Frac{}
	}
	if den < 0 {
		num, den = -num, -den
	}
	g := gcd(max(num, -num), den)
	return Frac{num / g, den / g}
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// d returns the denominator, which is 1 for the zero value.
func (f Frac) d() int64 {
	if f.den == 0 {
		return 1
	}
	return f.den
}

// Add returns f + g.
func (f Frac) Add(g Frac) Frac {
	k := gcd(f.d(), g.d())
	return frac(f.num*(g.d()/k)+g.num*(f.d()/k), f.d()/k*g.d())
}

// Sub returns f - g.
func (f Frac) Sub(g Frac) Frac {
	return f.Add(Frac{-g.num, g.den})
}

// Mul returns f x n.
func (f Frac) Mul(n int) Frac {
	return frac(f.num*int64(n), f.d())
}

// Div returns f / n. It panics if n is 0.
func (f Frac) Div(n int) Frac {
	return frac(f.num, f.d()*int64(n))
}

// Cmp returns -1, 0, or 1 as f is less than, equal to, or more than g.
func (f Frac) Cmp(g Frac) int {
	return f.Sub(g).Sign()
}

// Sign returns -1, 0, or 1 as f is negative, zero, or positive.
func (f Frac) Sign() int {
	switch {
	case f.num < 0:
		return -1
	case f.num > 0:
		return 1
	}
	return 0
}

// Floor returns f rounded down to a whole number.
func (f Frac) Floor() int {
	q := f.num / f.d()
	if f.num%f.d() < 0 {
		q--
	}
	return int(q)
}

// Ceil returns f rounded up to a whole number.
func (f Frac) Ceil() int {
	return -Frac{-f.num, f.den}.Floor()
}

// String returns f as a whole number or as num/den.
func (f Frac) String() string {
	if f.d() == 1 {
		return fmt.Sprint(f.num)
	}
	return fmt.Sprintf("%d/%d", f.num, f.den)
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package prng

import "math/bits"

// Shuffle randomizes the order of n elements using a Fisher-Yates shuffle,
// calling swap to exchange the elements at indexes i and j. It panics if n is
// negative.
//
// Unlike math/rand/v2's Shuffle, whose algorithm a Go release could change,
// this algorithm is part of the package's contract: for i from n-1 down to 1,
// it swaps i with j drawn uniformly from [0, i] by uint64n. Like the hash
// encoding, it decides every generated game and is frozen after alpha.
func (s *Stream) Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("prng: Shuffle with negative n")
	}
	for i := n - 1; i > 0; i-- {
		swap(i, int(s.uint64n(uint64(i)+1)))
	}
}

// IntN returns a value drawn uniformly from [0, n) by uint64n. It panics if n
// is not positive. Like Shuffle, its algorithm is frozen after alpha.
func (s *Stream) IntN(n int) int {
	if n <= 0 {
		panic("prng: IntN with non-positive n")
	}
	return int(s.uint64n(uint64(n)))
}

// uint64n returns a value drawn uniformly from [0, n) using Lemire's
// multiply-and-reject method: the high 64 bits of a 64-bit draw times n,
// redrawing when the low 64 bits fall below 2⁶⁴ mod n. n must be nonzero.
func (s *Stream) uint64n(n uint64) uint64 {
	hi, lo := bits.Mul64(s.Uint64(), n)
	if lo < n {
		threshold := -n % n // 2⁶⁴ mod n
		for lo < threshold {
			hi, lo = bits.Mul64(s.Uint64(), n)
		}
	}
	return hi
}

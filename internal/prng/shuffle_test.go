// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package prng_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/mdhender/ecvg/internal/prng"
)

func shuffled(s *prng.Stream, n int) []int {
	xs := make([]int, n)
	for i := range xs {
		xs[i] = i
	}
	s.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
	return xs
}

func TestShuffleMatchesGolden(t *testing.T) {
	got := shuffled(prng.New(seed1, seed2), 10)
	want := []int{0, 4, 1, 7, 5, 3, 2, 8, 9, 6}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShuffleIsAPermutation(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 100, 1000} {
		got := shuffled(prng.New(1, 2), n)
		slices.Sort(got)
		for i, v := range got {
			if v != i {
				t.Fatalf("n=%d: result is not a permutation of 0..n-1", n)
			}
		}
	}
}

func TestShuffleSmallNDoesNotSwap(t *testing.T) {
	for _, n := range []int{0, 1} {
		s := prng.New(1, 2)
		s.Shuffle(n, func(i, j int) { t.Errorf("n=%d: swap(%d, %d) called", n, i, j) })
	}
}

func TestShuffleNegativePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("did not panic")
		}
	}()
	prng.New(1, 2).Shuffle(-1, func(i, j int) {})
}

// TestShuffleIsUniform checks that all 24 orders of 4 elements turn up about
// equally often. The seed is fixed, so the test is not flaky.
func TestShuffleIsUniform(t *testing.T) {
	const n, trials = 4, 240_000
	s := prng.New(seed1, seed2)
	counts := map[[n]int]int{}
	for range trials {
		counts[[n]int(shuffled(s, n))]++
	}
	if len(counts) != 24 {
		t.Fatalf("saw %d orders, want 24", len(counts))
	}
	const want = trials / 24
	for order, got := range counts {
		// 5% is about 5 standard deviations for 10,000 expected.
		if got < want*95/100 || got > want*105/100 {
			t.Errorf("order %v: %d times, want about %d", order, got, want)
		}
	}
}

func TestIntNRangeAndUniformity(t *testing.T) {
	s := prng.New(seed1, seed2)
	for _, n := range []int{1, 2, 7, 1_000_000_000} {
		for range 1000 {
			if v := s.IntN(n); v < 0 || v >= n {
				t.Fatalf("IntN(%d) = %d, out of range", n, v)
			}
		}
	}

	const n, trials = 6, 60_000
	var counts [n]int
	for range trials {
		counts[s.IntN(n)]++
	}
	for v, got := range counts {
		// 5% is about 5 standard deviations for 10,000 expected.
		if want := trials / n; got < want*95/100 || got > want*105/100 {
			t.Errorf("IntN(%d) gave %d %d times, want about %d", n, v, got, want)
		}
	}
}

func TestIntNMatchesGolden(t *testing.T) {
	s := prng.New(seed1, seed2)
	var got []int
	for range 5 {
		got = append(got, s.IntN(100))
	}
	// The first four are ⌊draw × 100 / 2⁶⁴⌋ of TestRootStreamMatchesGolden's draws.
	if want := []int{63, 74, 83, 35, 57}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestIntNPanics(t *testing.T) {
	for _, n := range []int{0, -1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			prng.New(1, 2).IntN(n)
		})
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/prng"
)

// NumHabitable is the number of habitable planets in a game.
const NumHabitable = 125

// habitableShares gives the orbits whose planets can be habitable and the
// share of the habitable planets each orbit gets.
var habitableShares = []struct{ orbitNo, share int }{
	{2, 25},
	{3, 50},
	{4, 45},
	{5, 20},
	{6, 15},
}

// HabitableQuotas returns how many habitable planets each orbit number gets:
// NumHabitable split in proportion to the orbits' shares, rounded down, with
// the planets left over going one each to the orbits with the largest
// fractions dropped (the lower orbit first on a tie).
func HabitableQuotas() map[int]int {
	total := 0
	for _, s := range habitableShares {
		total += s.share
	}
	quotas := map[int]int{}
	remainders := make([]int, len(habitableShares)) // in units of 1/total
	left := NumHabitable
	for i, s := range habitableShares {
		quotas[s.orbitNo] = NumHabitable * s.share / total
		remainders[i] = NumHabitable * s.share % total
		left -= quotas[s.orbitNo]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range remainders {
			if remainders[i] > remainders[best] {
				best = i
			}
		}
		quotas[habitableShares[best].orbitNo]++
		remainders[best] = -1
	}
	return quotas
}

// MarkHabitable makes exactly NumHabitable of the planets habitable and the
// rest not, using the stream root.Derive(TagHabitable). For each orbit number
// in habitableShares, in order, the planets in orbits with that number are
// shuffled and the first HabitableQuotas()[orbitNo] are made habitable. If an
// orbit number has too few planets for its quota, all of them are made
// habitable, and the shortfall is made up from the other planets in those
// orbit numbers, shuffled together.
//
// MarkHabitable panics if those orbit numbers hold fewer than NumHabitable
// planets, or a planet's orbit is not in orbits.
func MarkHabitable(root *prng.Stream, planets []Planet, orbits []Orbit) {
	orbitNo := map[int]int{} // orbit ID to orbit number
	for _, o := range orbits {
		orbitNo[o.ID] = o.OrbitNo
	}
	inOrbit := map[int][]int{} // orbit number to planet indexes, in planet order
	for i := range planets {
		n, ok := orbitNo[planets[i].OrbitID]
		if !ok {
			panic(fmt.Sprintf("game: planet %d is in orbit %d, which is not an orbit", planets[i].ID, planets[i].OrbitID))
		}
		planets[i].Habitable = false
		inOrbit[n] = append(inOrbit[n], i)
	}

	s := root.Derive(prng.TagHabitable)
	shuffle := func(xs []int) {
		s.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
	}
	quotas := HabitableQuotas()
	shortfall := 0
	var spare []int
	for _, share := range habitableShares {
		candidates := inOrbit[share.orbitNo]
		shuffle(candidates)
		n := min(quotas[share.orbitNo], len(candidates))
		for _, i := range candidates[:n] {
			planets[i].Habitable = true
		}
		shortfall += quotas[share.orbitNo] - n
		spare = append(spare, candidates[n:]...)
	}
	if shortfall > len(spare) {
		panic(fmt.Sprintf("game: too few planets for %d habitable planets", NumHabitable))
	} else if shortfall > 0 {
		shuffle(spare)
		for _, i := range spare[:shortfall] {
			planets[i].Habitable = true
		}
	}
}

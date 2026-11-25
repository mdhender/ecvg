// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

func TestHabitableQuotas(t *testing.T) {
	// 125 split 25:50:45:20:15 is 20.16, 40.32, 36.29, 16.13, 12.10; rounding
	// down leaves 1, which goes to orbit 3's .32.
	want := map[int]int{2: 20, 3: 41, 4: 36, 5: 16, 6: 12}
	if got := game.HabitableQuotas(); !maps.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	total := 0
	for _, n := range want {
		total += n
	}
	if total != game.NumHabitable {
		t.Errorf("quotas total %d, want %d", total, game.NumHabitable)
	}
}

func habitableGame(root *prng.Stream) ([]game.Orbit, []game.Planet) {
	_, orbits, planets := goldenPlanets(root)
	game.MarkHabitable(root, planets, orbits)
	return orbits, planets
}

func habitableByOrbit(orbits []game.Orbit, planets []game.Planet) map[int]int {
	orbitNo := map[int]int{}
	for _, o := range orbits {
		orbitNo[o.ID] = o.OrbitNo
	}
	counts := map[int]int{}
	for _, p := range planets {
		if p.Habitable {
			counts[orbitNo[p.OrbitID]]++
		}
	}
	return counts
}

func TestMarkHabitableMeetsQuotas(t *testing.T) {
	orbits, planets := habitableGame(goldenRoot())
	// Every orbit number has enough planets, so each gets exactly its quota and
	// no planet outside orbits 2..6 is habitable.
	if got, want := habitableByOrbit(orbits, planets), game.HabitableQuotas(); !maps.Equal(got, want) {
		t.Errorf("habitable planets by orbit = %v, want %v", got, want)
	}
}

func TestMarkHabitableMatchesGolden(t *testing.T) {
	_, planets := habitableGame(goldenRoot())
	var ids []int
	for _, p := range planets {
		if p.Habitable {
			ids = append(ids, p.ID)
		}
	}
	if got, want := ids[:5], []int{9, 15, 23, 28, 32}; !slices.Equal(got, want) {
		t.Errorf("first habitable planets = %v, want %v", got, want)
	}
	if got, want := ids[len(ids)-3:], []int{595, 614, 616}; !slices.Equal(got, want) {
		t.Errorf("last habitable planets = %v, want %v", got, want)
	}
}

func TestMarkHabitableDependsOnSeeds(t *testing.T) {
	_, a := habitableGame(goldenRoot())
	root := goldenRoot()
	root.Uint64() // drawing from the root does not change the picks
	if _, b := habitableGame(root); !reflect.DeepEqual(a, b) {
		t.Error("same seeds gave different habitable planets")
	}
	if _, c := habitableGame(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same habitable planets")
	}
}

// shortGame returns one star's worth of orbits per star for n stars, with a
// planet in each of that star's orbits whose number is in orbitNos.
func shortGame(n int, orbitNos ...int) ([]game.Orbit, []game.Planet) {
	var stars []game.Star
	for id := 1; id <= n; id++ {
		stars = append(stars, game.Star{ID: id})
	}
	orbits := game.GenerateOrbits(stars)
	var planets []game.Planet
	for _, o := range orbits {
		if slices.Contains(orbitNos, o.OrbitNo) {
			planets = append(planets, game.Planet{ID: len(planets) + 1, OrbitID: o.ID, Habitable: true})
		}
	}
	return orbits, planets
}

func TestMarkHabitableMakesUpShortfall(t *testing.T) {
	// 50 stars with planets in orbits 2, 4, and 6 only: orbits 3 and 5 have no
	// planets for their quotas of 41 and 16, so the 57 they miss come from the
	// spare planets in orbits 2, 4, and 6. Planets in orbit 1 never qualify.
	orbits, planets := shortGame(50, 1, 2, 4, 6)
	game.MarkHabitable(goldenRoot(), planets, orbits)
	got := habitableByOrbit(orbits, planets)
	total := 0
	for orbitNo, n := range got {
		if orbitNo != 2 && orbitNo != 4 && orbitNo != 6 {
			t.Errorf("%d habitable planets in orbit %d", n, orbitNo)
		}
		total += n
	}
	if total != game.NumHabitable {
		t.Errorf("got %d habitable planets, want %d: %v", total, game.NumHabitable, got)
	}
	quotas := game.HabitableQuotas()
	for _, orbitNo := range []int{2, 4, 6} {
		if got[orbitNo] < quotas[orbitNo] {
			t.Errorf("orbit %d has %d habitable planets, below its quota of %d", orbitNo, got[orbitNo], quotas[orbitNo])
		}
	}
}

func TestMarkHabitablePanics(t *testing.T) {
	for name, mk := range map[string]func() ([]game.Orbit, []game.Planet){
		"too few planets in orbits 2..6": func() ([]game.Orbit, []game.Planet) { return shortGame(20, 1, 2, 3, 4, 5, 6) },
		"planet in an unknown orbit": func() ([]game.Orbit, []game.Planet) {
			orbits, planets := shortGame(50, 2, 3, 4)
			planets[0].OrbitID = 99_999
			return orbits, planets
		},
	} {
		t.Run(name, func(t *testing.T) {
			orbits, planets := mk()
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			game.MarkHabitable(goldenRoot(), planets, orbits)
		})
	}
}

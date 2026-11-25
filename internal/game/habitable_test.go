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

func viableGame(root *prng.Stream) []game.Planet {
	_, planets := habitableGame(root)
	game.SetViability(root, planets)
	return planets
}

func TestSetViabilityRules(t *testing.T) {
	planets := viableGame(goldenRoot())
	sum, mid, n := 0, 0, 0
	for _, p := range planets {
		switch {
		case !p.Habitable && p.Viability != 0:
			t.Errorf("planet %d is not habitable but has viability %d, want 0", p.ID, p.Viability)
		case p.Habitable && (p.Viability < game.MinViability || p.Viability > game.MaxViability):
			t.Errorf("planet %d is habitable with viability %d, want %d..%d", p.ID, p.Viability, game.MinViability, game.MaxViability)
		case p.Habitable:
			n++
			sum += p.Viability
			if p.Viability > 9 && p.Viability <= 17 { // the middle third of 1..25
				mid++
			}
		}
	}
	// A three-dice bell curve centres on 13 and puts about two-thirds of the
	// values in the middle third, against one-third for a flat spread.
	if mean := float64(sum) / float64(n); mean < 11.5 || mean > 14.5 {
		t.Errorf("mean viability %.1f, want about 13", mean)
	}
	if frac := float64(mid) / float64(n); frac < 0.5 {
		t.Errorf("%.2f of viabilities in the middle third, want about 0.67", frac)
	}
}

func TestSetViabilityMatchesGolden(t *testing.T) {
	planets := viableGame(goldenRoot())
	for id, want := range map[int]int{9: 17, 15: 6, 23: 14, 28: 11} {
		if got := planets[id-1].Viability; got != want {
			t.Errorf("planet %d has viability %d, want %d", id, got, want)
		}
	}
}

func TestSetViabilityDependsOnlyOnPlanet(t *testing.T) {
	a := viableGame(goldenRoot())

	// A planet's viability depends only on the seeds and its ID, not on the
	// other planets or on draws from the root.
	root := goldenRoot()
	root.Uint64()
	b := slices.Clone(a)
	for i := range b {
		b[i].Viability = -1
	}
	game.SetViability(root, b[8:9]) // planet 9 alone
	if b[8].Viability != a[8].Viability {
		t.Errorf("planet 9 alone has viability %d, want %d", b[8].Viability, a[8].Viability)
	}

	// Making a planet not habitable resets its viability to 0.
	b[8].Habitable = false
	game.SetViability(root, b[8:9])
	if b[8].Viability != 0 {
		t.Errorf("non-habitable planet has viability %d, want 0", b[8].Viability)
	}

	if c := viableGame(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same planets")
	}
}

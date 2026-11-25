// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

func goldenPlanets(root *prng.Stream) ([]game.Star, []game.Orbit, []game.Planet) {
	_, stars := game.GenerateStellia(root)
	orbits := game.GenerateOrbits(stars)
	return stars, orbits, game.GeneratePlanets(root, stars, orbits)
}

func TestGeneratePlanetsRules(t *testing.T) {
	stars, orbits, planets := goldenPlanets(goldenRoot())
	if len(planets) != game.NumPlanets {
		t.Fatalf("got %d planets, want %d", len(planets), game.NumPlanets)
	}

	starOf := map[int]int{} // orbit ID to star ID
	for _, o := range orbits {
		starOf[o.ID] = o.StarID
	}
	perStar := map[int]int{}
	for i, p := range planets {
		// Planet IDs count up from 1, so they are unique.
		if p.ID != i+1 {
			t.Errorf("planet %d has id %d, want %d", i, p.ID, i+1)
		}
		starID, ok := starOf[p.OrbitID]
		if !ok {
			t.Fatalf("planet %d has orbit_id %d, which is not an orbit", p.ID, p.OrbitID)
		}
		// Orbit IDs strictly increase, so no orbit holds two planets.
		if i > 0 && p.OrbitID <= planets[i-1].OrbitID {
			t.Errorf("planet %d has orbit_id %d, not after planet %d's %d", p.ID, p.OrbitID, planets[i-1].ID, planets[i-1].OrbitID)
		}
		perStar[starID]++
	}

	for _, star := range stars {
		if n := perStar[star.ID]; n < 1 || n > game.OrbitsPerStar {
			t.Errorf("star %d has %d planets, want 1..%d", star.ID, n, game.OrbitsPerStar)
		}
	}
	if mean := float64(len(planets)) / float64(len(stars)); mean != 4.5 {
		t.Errorf("mean planets per star = %v, want 4.5", mean)
	}
}

func TestGeneratePlanetsMatchesGolden(t *testing.T) {
	_, _, planets := goldenPlanets(goldenRoot())
	for i, want := range map[int]game.Planet{
		0:   {ID: 1, OrbitID: 2},
		1:   {ID: 2, OrbitID: 5},
		2:   {ID: 3, OrbitID: 7},
		620: {ID: 621, OrbitID: 1380},
	} {
		if planets[i] != want {
			t.Errorf("planet %d = %+v, want %+v", i, planets[i], want)
		}
	}
}

func TestGeneratePlanetsDependsOnSeeds(t *testing.T) {
	_, _, a := goldenPlanets(goldenRoot())
	root := goldenRoot()
	root.Uint64() // drawing from the root does not move the planets
	if _, _, b := goldenPlanets(root); !reflect.DeepEqual(a, b) {
		t.Error("same seeds gave different planets")
	}
	if _, _, c := goldenPlanets(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same planets")
	}
}

func TestGeneratePlanetsPanics(t *testing.T) {
	stars := []game.Star{{ID: 1}, {ID: 2}}
	for name, orbits := range map[string][]game.Orbit{
		"star without orbits": game.GenerateOrbits(stars[:1]),
		"too few orbits":      game.GenerateOrbits(stars),
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			game.GeneratePlanets(goldenRoot(), stars, orbits)
		})
	}
}

func TestCreateWritesPlanetsJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, want := goldenPlanets(goldenRoot()); !reflect.DeepEqual(d.Planets, want) {
		t.Error("Create returned planets that differ from GeneratePlanets")
	}
	loaded, err := game.LoadPlanets(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, d.Planets) {
		t.Error("LoadPlanets differs from what Create wrote")
	}

	data, err := os.ReadFile(filepath.Join(dir, game.PlanetsFileName))
	if err != nil {
		t.Fatal(err)
	}
	const wantPrefix = `[{"id":1,"orbit_id":2},`
	if len(data) < len(wantPrefix) || string(data[:len(wantPrefix)]) != wantPrefix {
		t.Errorf("planets.json starts\n%.80s\nwant\n%s", data, wantPrefix)
	}
}

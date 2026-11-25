// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"slices"

	"github.com/mdhender/ecvg/internal/prng"
)

// PlanetsFileName is the name of the file in a game's folder that holds the
// game's planets.
const PlanetsFileName = "planets.json"

// NumPlanets is the number of planets in a game: an average of 4.5 for each of
// the 138 stars.
const NumPlanets = 621

// Planet is a planet in an orbit, at coordinates (x, y, z, seq_no, orbit_no).
type Planet struct {
	// ID numbers all of the game's planets from 1, in orbit order.
	ID int `json:"id"`
	// OrbitID is the ID of the orbit the planet is in.
	OrbitID int `json:"orbit_id"`
	// Habitable is true if the planet can hold open-air colonies.
	Habitable bool `json:"habitable"`
	// Viability caps the planet's open-air farms: 0 if the planet is not
	// habitable, otherwise MinViability..MaxViability.
	Viability int `json:"viability"`
}

// GeneratePlanets places NumPlanets planets, at most one per orbit, using the
// stream root.Derive(TagPlanets). Every star gets at least one planet: for
// each star in order, its orbits are shuffled and the planet goes in the
// first. The orbits left over from every star are then shuffled together, and
// the rest of the planets go in the first of them. Planet IDs run from 1 in
// orbit ID order.
//
// GeneratePlanets panics if a star has no orbits or there are fewer orbits
// than planets.
func GeneratePlanets(root *prng.Stream, stars []Star, orbits []Orbit) []Planet {
	orbitsOf := map[int][]int{} // star ID to the IDs of its orbits, in order
	for _, o := range orbits {
		orbitsOf[o.StarID] = append(orbitsOf[o.StarID], o.ID)
	}

	s := root.Derive(prng.TagPlanets)
	chosen := make([]int, 0, NumPlanets)
	var rest []int
	for _, star := range stars {
		ids := slices.Clone(orbitsOf[star.ID])
		if len(ids) == 0 {
			panic("game: star has no orbits")
		}
		s.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		chosen = append(chosen, ids[0])
		rest = append(rest, ids[1:]...)
	}
	if more := NumPlanets - len(chosen); more > len(rest) {
		panic("game: too few orbits for the planets")
	} else if more > 0 {
		s.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
		chosen = append(chosen, rest[:more]...)
	}
	slices.Sort(chosen)

	planets := make([]Planet, len(chosen))
	for i, orbitID := range chosen {
		planets[i] = Planet{ID: i + 1, OrbitID: orbitID}
	}
	return planets
}

// LoadPlanets reads planets.json from dir.
func LoadPlanets(dir string) ([]Planet, error) {
	var planets []Planet
	if err := loadJSON(dir, PlanetsFileName, &planets); err != nil {
		return nil, err
	}
	return planets, nil
}

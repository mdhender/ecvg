// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

// PlanetsFileName is the name of the file in a game's folder that holds the
// game's planets.
const PlanetsFileName = "planets.json"

// MinViability and MaxViability bound a habitable planet's viability.
const (
	MinViability = 1
	MaxViability = 25
)

// HomeWorldOrbitNo is the orbit of a home star that holds its home world.
const HomeWorldOrbitNo = 3

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

// LoadPlanets reads planets.json from dir.
func LoadPlanets(dir string) ([]Planet, error) {
	var planets []Planet
	if err := loadJSON(dir, PlanetsFileName, &planets); err != nil {
		return nil, err
	}
	return planets, nil
}

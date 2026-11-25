// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

// OrbitsFileName is the name of the file in a game's folder that holds the
// orbits of the game's stars.
const OrbitsFileName = "orbits.json"

// OrbitsPerStar is the number of orbits around every star.
const OrbitsPerStar = 10

// Orbit is one orbit around a star, at coordinates (x, y, z, seq_no, OrbitNo).
type Orbit struct {
	// ID numbers all of the game's orbits from 1, in star order.
	ID int `json:"id"`
	// StarID is the ID of the star the orbit is around.
	StarID int `json:"star_id"`
	// OrbitNo numbers the orbits of a star from 1 to OrbitsPerStar.
	OrbitNo int `json:"orbit_no"`
}

// GenerateOrbits returns OrbitsPerStar orbits for each star, in the order of
// stars and then orbit number. Orbit IDs run from 1 in that order.
func GenerateOrbits(stars []Star) []Orbit {
	orbits := make([]Orbit, 0, len(stars)*OrbitsPerStar)
	for _, star := range stars {
		for orbitNo := 1; orbitNo <= OrbitsPerStar; orbitNo++ {
			orbits = append(orbits, Orbit{ID: len(orbits) + 1, StarID: star.ID, OrbitNo: orbitNo})
		}
	}
	return orbits
}

// LoadOrbits reads orbits.json from dir.
func LoadOrbits(dir string) ([]Orbit, error) {
	var orbits []Orbit
	if err := loadJSON(dir, OrbitsFileName, &orbits); err != nil {
		return nil, err
	}
	return orbits, nil
}

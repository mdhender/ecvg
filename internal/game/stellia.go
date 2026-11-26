// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

// StelliaFileName and StarsFileName are the names of the files in a game's
// folder that hold the game's Stellia and stars.
const (
	StelliaFileName = "stellia.json"
	StarsFileName   = "stars.json"
)

const (
	// NumStellia is the number of Stellia in a game.
	NumStellia = 100
	// StelliaRadius is the greatest distance of a Stellium from the origin.
	StelliaRadius = 15
)

// Stellium is a cluster of stars at integer coordinates (X, Y, Z).
type Stellium struct {
	// ID numbers the game's Stellia from 1, in the order they were generated.
	ID int `json:"id"`
	X  int `json:"x"`
	Y  int `json:"y"`
	Z  int `json:"z"`
	// HomeStellium is true if the Stellium holds a home star.
	HomeStellium bool `json:"homestellium"`
}

// Star is one star of a Stellium, at coordinates (x, y, z, SeqNo).
type Star struct {
	// ID numbers all of the game's stars from 1, in Stellium order.
	ID int `json:"id"`
	// StelliumID is the ID of the Stellium the star belongs to.
	StelliumID int `json:"stellium_id"`
	// SeqNo numbers the stars of a Stellium from 1.
	SeqNo int `json:"seq_no"`
	// HomeStar is true if the star is a home star: its orbits come from the
	// home star template, and the planet in orbit HomeWorldOrbitNo is a home
	// world.
	HomeStar bool `json:"homestar"`
}

// LoadStellia reads stellia.json from dir.
func LoadStellia(dir string) ([]Stellium, error) {
	var stellia []Stellium
	if err := loadJSON(dir, StelliaFileName, &stellia); err != nil {
		return nil, err
	}
	return stellia, nil
}

// LoadStars reads stars.json from dir.
func LoadStars(dir string) ([]Star, error) {
	var stars []Star
	if err := loadJSON(dir, StarsFileName, &stars); err != nil {
		return nil, err
	}
	return stars, nil
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import "github.com/mdhender/ecvg/internal/prng"

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
	// HomeStellium is true once the game master designates the Stellium as a
	// home Stellium (see DesignateHomeStar).
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
	// HomeStar is true once the game master designates the star as a home
	// star (see DesignateHomeStar).
	HomeStar bool `json:"homestar"`
}

// GenerateStellia places a game's Stellia and their stars using the stream
// root.Derive(TagStellia). Every point within StelliaRadius of the origin,
// other than the origin itself, is a candidate; the candidates are shuffled and
// the first NumStellia are taken. The first Stellium has 5 stars, the second 4,
// the next 8 have 3, the next 15 have 2, and the rest have 1. Stellium IDs run
// from 1 to NumStellia in that order, and star IDs run from 1 across all the
// Stellia in that order.
func GenerateStellia(root *prng.Stream) ([]Stellium, []Star) {
	var points [][3]int
	for x := -StelliaRadius; x <= StelliaRadius; x++ {
		for y := -StelliaRadius; y <= StelliaRadius; y++ {
			for z := -StelliaRadius; z <= StelliaRadius; z++ {
				if d2 := x*x + y*y + z*z; d2 != 0 && d2 <= StelliaRadius*StelliaRadius {
					points = append(points, [3]int{x, y, z})
				}
			}
		}
	}
	root.Derive(prng.TagStellia).Shuffle(len(points), func(i, j int) { points[i], points[j] = points[j], points[i] })

	stellia := make([]Stellium, NumStellia)
	var stars []Star
	for i := range stellia {
		p := points[i]
		stellia[i] = Stellium{ID: i + 1, X: p[0], Y: p[1], Z: p[2]}
		for seqNo := 1; seqNo <= starsInStellium(i); seqNo++ {
			stars = append(stars, Star{ID: len(stars) + 1, StelliumID: stellia[i].ID, SeqNo: seqNo})
		}
	}
	return stellia, stars
}

// starsInStellium returns the number of stars in the i'th Stellium generated.
func starsInStellium(i int) int {
	switch {
	case i == 0:
		return 5
	case i == 1:
		return 4
	case i < 10: // the next 8
		return 3
	case i < 25: // the next 15
		return 2
	default:
		return 1
	}
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

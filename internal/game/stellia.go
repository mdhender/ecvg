// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mdhender/ecvg/internal/prng"
)

// StelliaFileName is the name of the file in a game's folder that holds the
// game's Stellia.
const StelliaFileName = "stellia.json"

const (
	// NumStellia is the number of Stellia in a game.
	NumStellia = 100
	// StelliaRadius is the greatest distance of a Stellium from the origin.
	StelliaRadius = 15
)

// Stellium is a cluster of stars at integer coordinates (X, Y, Z).
type Stellium struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Z     int    `json:"z"`
	Stars []Star `json:"stars"`
}

// Star is one star of a Stellium, at coordinates (x, y, z, SeqNo).
type Star struct {
	// SeqNo numbers the stars of a Stellium from 1.
	SeqNo int `json:"seq_no"`
}

// GenerateStellia places a game's Stellia using the stream root.Derive(TagStellia).
// Every point within StelliaRadius of the origin, other than the origin itself,
// is a candidate; the candidates are shuffled and the first NumStellia are
// taken. The first Stellium has 5 stars, the second 4, the next 8 have 3, the
// next 15 have 2, and the rest have 1.
func GenerateStellia(root *prng.Stream) []Stellium {
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
	for i := range stellia {
		p := points[i]
		stellia[i] = Stellium{X: p[0], Y: p[1], Z: p[2], Stars: make([]Star, starsInStellium(i))}
		for j := range stellia[i].Stars {
			stellia[i].Stars[j].SeqNo = j + 1
		}
	}
	return stellia
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
	path := filepath.Join(dir, StelliaFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var stellia []Stellium
	if err := json.Unmarshal(data, &stellia, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return stellia, nil
}

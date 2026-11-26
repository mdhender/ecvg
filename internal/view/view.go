// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package view renders a game's data as self-contained HTML pages for a
// browser. The pages load nothing from the network.
package view

import (
	_ "embed"
	"encoding/json/v2"
	"fmt"
	"html/template"
	"io"

	"github.com/mdhender/ecvg/internal/game"
)

//go:embed stellia.html
var stelliaHTML string

var stelliaTemplate = template.Must(template.New("stellia").Parse(stelliaHTML))

// StelliumPoint is a Stellium as the map draws it.
type StelliumPoint struct {
	ID    int `json:"id"`
	X     int `json:"x"`
	Y     int `json:"y"`
	Z     int `json:"z"`
	Stars int `json:"stars"`
	// Home is true for a home Stellium.
	Home bool `json:"home"`
}

// StelliaPoints pairs each Stellium with the number of its stars, keeping the
// order of stellia. It fails if a star names a Stellium that is not in
// stellia.
func StelliaPoints(stellia []game.Stellium, stars []game.Star) ([]StelliumPoint, error) {
	index := make(map[int]int, len(stellia))
	points := make([]StelliumPoint, len(stellia))
	for i, s := range stellia {
		index[s.ID] = i
		points[i] = StelliumPoint{ID: s.ID, X: s.X, Y: s.Y, Z: s.Z, Home: s.HomeStellium}
	}
	for _, star := range stars {
		i, ok := index[star.StelliumID]
		if !ok {
			return nil, fmt.Errorf("star %d: unknown stellium %d", star.ID, star.StelliumID)
		}
		points[i].Stars++
	}
	return points, nil
}

// Stellia writes a page to w showing a rotatable 3D map of the Stellia of the
// game named code, with a table listing them. Home Stellia glow on the map
// and are starred in the table.
func Stellia(w io.Writer, code string, stellia []game.Stellium, stars []game.Star) error {
	points, err := StelliaPoints(stellia, stars)
	if err != nil {
		return err
	}
	data, err := json.Marshal(points)
	if err != nil {
		return err
	}
	homes := 0
	for _, p := range points {
		if p.Home {
			homes++
		}
	}
	return stelliaTemplate.Execute(w, struct {
		Code   string
		Homes  int
		Radius int
		Count  int
		Stars  int
		Points []StelliumPoint
		// JSON is safe to inline in a script: it holds only numbers and booleans.
		JSON template.JS
	}{
		Code:   code,
		Homes:  homes,
		Radius: game.StelliaRadius,
		Count:  len(points),
		Stars:  len(stars),
		Points: points,
		JSON:   template.JS(data),
	})
}

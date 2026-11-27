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

//go:embed empires.html
var empiresHTML string

var empiresTemplate = template.Must(template.New("empires").Parse(empiresHTML))

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

// EmpireRow is an empire as the empires page shows it: the empire, where its
// home world is, and its colonies.
type EmpireRow struct {
	game.Empire
	// StelliumID, X, Y, Z, SeqNo, and OrbitNo locate the home world.
	StelliumID int `json:"stellium_id"`
	X          int `json:"x"`
	Y          int `json:"y"`
	Z          int `json:"z"`
	SeqNo      int `json:"seq_no"`
	OrbitNo    int `json:"orbit_no"`
	// Colonies are the empire's colonies, in the order of d.Colonies.
	Colonies []ColonyRow `json:"colonies"`
}

// ColonyRow is a colony as the empires page shows it: the colony and what its
// population amounts to.
type ColonyRow struct {
	game.Colony
	People int `json:"people"`
	// Mass is in MU, Volume in VU, and LifeSupport in LU; the last is 0
	// for an open-air colony, whose population wants enclosure all the same.
	Mass        int `json:"mass"`
	Volume      int `json:"volume"`
	LifeSupport int `json:"life_support"`
	// Food is the FOOD the population eats in a turn at the colony's rations.
	Food int `json:"food"`
}

// EmpireRows pairs each empire in d with the location of its home world and
// its colonies, keeping the order of d.Empires. It fails if an empire's home
// world can't be located, or a colony names an empire that is not in d.
func EmpireRows(d game.Data) ([]EmpireRow, error) {
	stellia := make(map[int]game.Stellium, len(d.Stellia))
	for _, s := range d.Stellia {
		stellia[s.ID] = s
	}
	stars := make(map[int]game.Star, len(d.Stars))
	for _, s := range d.Stars {
		stars[s.ID] = s
	}
	orbits := make(map[int]game.Orbit, len(d.Orbits))
	for _, o := range d.Orbits {
		orbits[o.ID] = o
	}
	planets := make(map[int]game.Planet, len(d.Planets))
	for _, p := range d.Planets {
		planets[p.ID] = p
	}

	index := make(map[int]int, len(d.Empires))
	rows := make([]EmpireRow, len(d.Empires))
	for i, e := range d.Empires {
		planet, ok := planets[e.HomeWorldID]
		if !ok {
			return nil, fmt.Errorf("empire %d: unknown home world %d", e.ID, e.HomeWorldID)
		}
		orbit, ok := orbits[planet.OrbitID]
		if !ok {
			return nil, fmt.Errorf("empire %d: planet %d: unknown orbit %d", e.ID, planet.ID, planet.OrbitID)
		}
		star, ok := stars[orbit.StarID]
		if !ok {
			return nil, fmt.Errorf("empire %d: orbit %d: unknown star %d", e.ID, orbit.ID, orbit.StarID)
		}
		stellium, ok := stellia[star.StelliumID]
		if !ok {
			return nil, fmt.Errorf("empire %d: star %d: unknown stellium %d", e.ID, star.ID, star.StelliumID)
		}
		index[e.ID] = i
		rows[i] = EmpireRow{
			Empire:     e,
			StelliumID: stellium.ID,
			X:          stellium.X,
			Y:          stellium.Y,
			Z:          stellium.Z,
			SeqNo:      star.SeqNo,
			OrbitNo:    orbit.OrbitNo,
			Colonies:   []ColonyRow{},
		}
	}
	for _, c := range d.Colonies {
		i, ok := index[c.EmpireID]
		if !ok {
			return nil, fmt.Errorf("colony %d: unknown empire %d", c.ID, c.EmpireID)
		}
		p := c.Population
		rows[i].Colonies = append(rows[i].Colonies, ColonyRow{
			Colony:      c,
			People:      p.People(),
			Mass:        p.Mass(),
			Volume:      p.Volume(),
			LifeSupport: p.LifeSupport(c.Type),
			Food:        p.FoodWanted(c.Rations),
		})
	}
	return rows, nil
}

// Empires writes a page to w listing the empires of the game d, which can be
// searched, filtered by race, and sorted. Selecting an empire shows its home
// world and everything in its colonies.
func Empires(w io.Writer, d game.Data) error {
	rows, err := EmpireRows(d)
	if err != nil {
		return err
	}
	players := 0
	for _, e := range d.Empires {
		if e.Handle != "" {
			players++
		}
	}
	return empiresTemplate.Execute(w, struct {
		Code    string
		Turn    int
		Count   int
		Races   int
		Players int
		// Empires is encoded and escaped for the script by html/template.
		Empires []EmpireRow
	}{
		Code:    d.Game.Code,
		Turn:    d.Game.Turn,
		Count:   len(rows),
		Races:   len(d.Races),
		Players: players,
		Empires: rows,
	})
}

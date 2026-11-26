// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package view_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/view"
)

func TestStelliaPoints(t *testing.T) {
	stellia := []game.Stellium{{ID: 1, X: 1, Y: 2, Z: 3}, {ID: 2, X: -4, Y: 0, Z: 5, HomeStellium: true}, {ID: 3, X: 0, Y: 0, Z: -1}}
	for _, tc := range []struct {
		name    string
		stars   []game.Star
		want    []view.StelliumPoint
		wantErr bool
	}{
		{
			name:  "counts stars per stellium",
			stars: []game.Star{{ID: 1, StelliumID: 1, SeqNo: 1}, {ID: 2, StelliumID: 1, SeqNo: 2}, {ID: 3, StelliumID: 3, SeqNo: 1}},
			want: []view.StelliumPoint{
				{ID: 1, X: 1, Y: 2, Z: 3, Stars: 2},
				{ID: 2, X: -4, Y: 0, Z: 5, Stars: 0, Home: true},
				{ID: 3, X: 0, Y: 0, Z: -1, Stars: 1},
			},
		},
		{
			name:    "rejects unknown stellium",
			stars:   []game.Star{{ID: 1, StelliumID: 9, SeqNo: 1}},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := view.StelliaPoints(stellia, tc.stars)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("got nil error, want one")
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestStelliaPage(t *testing.T) {
	stellia := []game.Stellium{{ID: 7, X: -3, Y: 4, Z: 12, HomeStellium: true}, {ID: 8, X: 1, Y: 1, Z: 1}}
	stars := []game.Star{{ID: 1, StelliumID: 7, SeqNo: 1}, {ID: 2, StelliumID: 8, SeqNo: 1}}
	for _, tc := range []struct {
		name string
		code string
		want []string
	}{
		{
			name: "embeds data and title",
			code: "alpha",
			want: []string{
				"<title>Stellia of alpha</title>",
				`const POINTS = [{"id":7,"x":-3,"y":4,"z":12,"stars":1,"home":true},{"id":8,"x":1,"y":1,"z":1,"stars":1,"home":false}];`,
				"1 is a home Stellium.",
			},
		},
		{
			name: "escapes the game code",
			code: "<b>",
			want: []string{"<title>Stellia of &lt;b&gt;</title>"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder
			if err := view.Stellia(&sb, tc.code, stellia, stars); err != nil {
				t.Fatalf("got error %v", err)
			}
			for _, want := range tc.want {
				if !strings.Contains(sb.String(), want) {
					t.Errorf("page lacks %q", want)
				}
			}
		})
	}
}

// empireData returns a game with two empires on one home world; the first has
// a player and a colony.
func empireData() game.Data {
	return game.Data{
		Game:    game.Game{Code: "alpha"},
		Stellia: []game.Stellium{{ID: 4, X: -3, Y: 4, Z: 12, HomeStellium: true}},
		Stars:   []game.Star{{ID: 9, StelliumID: 4, SeqNo: 1, HomeStar: true}},
		Orbits:  []game.Orbit{{ID: 83, StarID: 9, OrbitNo: 3}},
		Planets: []game.Planet{{ID: 40, OrbitID: 83, Habitable: true, Viability: 25}},
		Races:   []game.Race{{ID: 40, HomeWorldID: 40}},
		Empires: []game.Empire{
			{ID: 1, Handle: "abcdefgh", Name: "</script><b>", RaceID: 40, HomeWorldID: 40},
			{ID: 2, Name: "Empire #2", RaceID: 40, HomeWorldID: 40},
		},
		Colonies: []game.Colony{{ID: 7, EmpireID: 1, PlanetID: 40, No: 1, Type: "COPN", TechLevel: 1,
			Population: game.Population{UEM: 6, USK: 3, PRO: 2, SLD: 1},
			Cargo:      []game.Unit{{Type: "FOOD", Qty: 20}}}},
	}
}

func TestEmpireRows(t *testing.T) {
	rows, err := view.EmpireRows(empireData())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	r := rows[0]
	if r.ID != 1 || r.StelliumID != 4 || r.X != -3 || r.Y != 4 || r.Z != 12 || r.SeqNo != 1 || r.OrbitNo != 3 {
		t.Errorf("row 1 = %+v", r)
	}
	if len(r.Colonies) != 1 || r.Colonies[0].ID != 7 || len(rows[1].Colonies) != 0 {
		t.Errorf("colonies: %+v, %+v", r.Colonies, rows[1].Colonies)
	}

	if c := r.Colonies[0]; c.People != 1200 || c.Mass != 12 || c.Volume != 0 || c.LifeSupport != 0 || c.Food != 3 {
		t.Errorf("open-air colony: %+v", c)
	}
	enclosed := empireData()
	enclosed.Colonies[0].Type = game.CENC
	if rows, err := view.EmpireRows(enclosed); err != nil {
		t.Fatal(err)
	} else if c := rows[0].Colonies[0]; c.Volume != 12 || c.LifeSupport != 12 {
		t.Errorf("enclosed colony: volume %d, life support %d; want 12, 12", c.Volume, c.LifeSupport)
	}

	for _, tc := range []struct {
		name  string
		wreck func(d *game.Data)
	}{
		{"unknown home world", func(d *game.Data) { d.Planets = nil }},
		{"unknown orbit", func(d *game.Data) { d.Orbits = nil }},
		{"unknown star", func(d *game.Data) { d.Stars = nil }},
		{"unknown stellium", func(d *game.Data) { d.Stellia = nil }},
		{"colony of unknown empire", func(d *game.Data) { d.Colonies[0].EmpireID = 99 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := empireData()
			tc.wreck(&d)
			if _, err := view.EmpireRows(d); err == nil {
				t.Error("got nil error, want one")
			}
		})
	}
}

func TestEmpiresPage(t *testing.T) {
	var sb strings.Builder
	if err := view.Empires(&sb, empireData()); err != nil {
		t.Fatal(err)
	}
	page := sb.String()
	for _, want := range []string{
		"<title>Empires of alpha</title>",
		"2 empires of 1 race; 1 has a player.",
		`"handle":"abcdefgh"`,
		`"orbit_no":3`,
		`"cargo":[{"type":"FOOD","qty":20}]`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if strings.Contains(page, "</script><b>") {
		t.Error("the empire name is not escaped in the script")
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"encoding/json/v2"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestEntityType(t *testing.T) {
	for _, tc := range []struct {
		text     string
		ok       bool
		enclosed bool
	}{
		{`"CSHP"`, true, true}, {`"COPN"`, true, false}, {`"CENC"`, true, true}, {`"CORB"`, true, true},
		{`"copn"`, false, false}, {`"SHIP"`, false, false}, {`""`, false, false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			var got game.EntityType
			err := json.Unmarshal([]byte(tc.text), &got)
			if tc.ok != (err == nil) {
				t.Fatalf("got %q, %v; want ok %v", got, err, tc.ok)
			}
			if tc.ok && got.Enclosed() != tc.enclosed {
				t.Errorf("%s: Enclosed = %v, want %v", got, got.Enclosed(), tc.enclosed)
			}
		})
	}
}

func TestPopulation(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		p                    game.Population
		units, people, mass  int
		food                 int
		enclosedVU, enclosed int // volume and life support when enclosed
	}{
		{"empty", game.Population{}, 0, 0, 0, 0, 0, 0},
		{"one unit", game.Population{SLD: 1}, 1, 100, 1, 1, 1, 1},
		{"four units", game.Population{UEM: 1, USK: 1, PRO: 1, SLD: 1}, 4, 400, 4, 1, 4, 4},
		{"five units", game.Population{UEM: 5}, 5, 500, 5, 2, 5, 5},
		{"1978 sample", game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000}, 1200000, 120000000, 1200000, 300000, 1200000, 1200000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.p
			if p.Units() != tc.units || p.People() != tc.people || p.Mass() != tc.mass || p.Food() != tc.food {
				t.Errorf("units %d, people %d, mass %d, food %d; want %d, %d, %d, %d",
					p.Units(), p.People(), p.Mass(), p.Food(), tc.units, tc.people, tc.mass, tc.food)
			}
			for _, typ := range []game.EntityType{game.CSHP, game.CENC, game.CORB} {
				if p.Volume(typ) != tc.enclosedVU || p.LifeSupport(typ) != tc.enclosed {
					t.Errorf("%s: volume %d, life support %d; want %d, %d", typ, p.Volume(typ), p.LifeSupport(typ), tc.enclosedVU, tc.enclosed)
				}
			}
			if p.Volume(game.COPN) != 0 || p.LifeSupport(game.COPN) != 0 {
				t.Errorf("COPN: volume %d, life support %d; want 0, 0", p.Volume(game.COPN), p.LifeSupport(game.COPN))
			}
		})
	}
}

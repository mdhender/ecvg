// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestFeed(t *testing.T) {
	// 400 units of each type want 400 FOOD at full rations, and each percent
	// of rations is 4 FOOD.
	p := game.Population{UEM: 400, USK: 400, PRO: 400, SLD: 400}
	all := func(n int) game.Population { return game.Population{UEM: n, USK: n, PRO: n, SLD: n} }
	for _, tc := range []struct {
		name          string
		p             game.Population
		rations, food int
		wanted, eaten int
		ration        float64
		starved       game.Population
	}{
		// The rulebook's table.
		{"100%", p, 100, 9999, 400, 400, 100, all(0)},
		{"75%", p, 75, 9999, 300, 300, 75, all(0)},
		{"50%", p, 50, 9999, 200, 200, 50, all(0)},
		{"25%", p, 25, 9999, 100, 100, 25, all(0)},
		{"20%", p, 20, 9999, 80, 80, 20, all(80)},
		{"15%", p, 15, 9999, 60, 60, 15, all(160)},
		{"10%", p, 10, 9999, 40, 40, 10, all(240)},
		{"5%", p, 5, 9999, 20, 20, 5, all(320)},
		{"0%", p, 0, 9999, 0, 0, 0, all(400)},

		{"24% is the first ration that starves", p, 24, 9999, 96, 96, 24, all(16)},
		{"1%, the least that can be ordered", p, 1, 9999, 4, 4, 1, all(384)},
		{"500%", p, 500, 9999, 2000, 2000, 500, all(0)},
		{"exactly enough food", p, 100, 400, 400, 400, 100, all(0)},
		{"short of food, but not starving", p, 100, 200, 400, 200, 50, all(0)},
		{"short of food at exactly 25%", p, 100, 100, 400, 100, 25, all(0)},
		{"short of food at 20%", p, 100, 80, 400, 80, 20, all(80)},
		{"short of food on feast rations", p, 500, 80, 2000, 80, 20, all(80)},
		{"no food", p, 100, 0, 400, 0, 0, all(400)},
		{"short at 24.75%: 1% starve, the least", p, 100, 99, 400, 99, 24.75, all(4)},
		{"short at 24.9%: still 1%", game.Population{UEM: 4000}, 100, 249, 1000, 249, 24.9, game.Population{UEM: 40}},
		{"starved units round up by type", game.Population{UEM: 7, USK: 1}, 20, 9999, 1, 1, 20, game.Population{UEM: 2, USK: 1}},
		{"food rounds up", game.Population{UEM: 5}, 100, 9999, 2, 2, 100, game.Population{}},
		{"no population", game.Population{}, 100, 50, 0, 0, 100, game.Population{}},
		{"no population and no food", game.Population{}, 100, 0, 0, 0, 100, game.Population{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := engine.Feed(tc.p, tc.rations, tc.food)
			if m.Wanted != tc.wanted || m.Eaten != tc.eaten || m.Ration() != tc.ration {
				t.Errorf("wanted %d, ate %d, ration %v; want %d, %d, %v", m.Wanted, m.Eaten, m.Ration(), tc.wanted, tc.eaten, tc.ration)
			}
			if got := m.Starved(tc.p); got != tc.starved {
				t.Errorf("starved %+v, want %+v", got, tc.starved)
			}
		})
	}
}

func TestRationsAndPopulationIncrease(t *testing.T) {
	colonies := func() []game.Colony {
		return []game.Colony{
			{ID: 1, Rations: 100, Population: game.Population{UEM: 400}, Cargo: []game.Unit{{Type: "FUEL", Qty: 9}, {Type: "FOOD", Qty: 150}}},
			{ID: 2, Rations: 20, Population: game.Population{USK: 400, SLD: 5}, Cargo: []game.Unit{{Type: "FOOD", Qty: 10}, {Type: "FOOD", Qty: 90}}},
			{ID: 3, Rations: 100, Population: game.Population{PRO: 400}, Starved: game.Population{PRO: 3}},
		}
	}
	t.Run("setup turn", func(t *testing.T) {
		d := game.Data{Colonies: colonies()}
		meals := engine.Rations(&d, 0)
		engine.PopulationIncrease(&d, meals)
		if meals != nil || d.Colonies[0].Cargo[1].Qty != 150 || d.Colonies[2].Population.PRO != 400 {
			t.Errorf("meals %v, colonies %+v; want nothing eaten and nobody starved", meals, d.Colonies)
		}
		if d.Colonies[2].Starved != (game.Population{}) {
			t.Errorf("colony 3: starved %+v, want it cleared", d.Colonies[2].Starved)
		}
	})
	t.Run("later turn", func(t *testing.T) {
		d := game.Data{Colonies: colonies()}
		meals := engine.Rations(&d, 1)
		if len(meals) != 3 || meals[0].Eaten != 100 || meals[1].Eaten != 21 || meals[2].Eaten != 0 {
			t.Fatalf("meals = %+v; want 100, 21, and 0 eaten", meals)
		}
		// Colony 2 eats from both of its FOOD entries, the first one first.
		if c := d.Colonies; c[0].Cargo[0].Qty != 9 || c[0].Cargo[1].Qty != 50 || c[1].Cargo[0].Qty != 0 || c[1].Cargo[1].Qty != 79 {
			t.Errorf("cargo = %+v, %+v", c[0].Cargo, c[1].Cargo)
		}
		engine.PopulationIncrease(&d, meals)
		for i, want := range []struct{ p, starved game.Population }{
			{game.Population{UEM: 400}, game.Population{}},
			{game.Population{USK: 320, SLD: 4}, game.Population{USK: 80, SLD: 1}},
			{game.Population{}, game.Population{PRO: 400}},
		} {
			if c := d.Colonies[i]; c.Population != want.p || c.Starved != want.starved {
				t.Errorf("colony %d: population %+v, starved %+v; want %+v, %+v", c.ID, c.Population, c.Starved, want.p, want.starved)
			}
		}
	})
}

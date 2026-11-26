// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"slices"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestFarmRules(t *testing.T) {
	for _, tc := range []struct {
		tl, qty            int
		food, fuel, perTen int // perTen: units that 10 FUEL can run
	}{
		{1, 4, 400, 2, 20}, // a whole crop: 100 a year
		{1, 3, 300, 2, 20}, // 1.5 FUEL, rounded up
		{2, 4, 40, 4, 10},  // 20 x 2 a year
		{5, 3, 75, 8, 4},   // 7.5 FUEL, rounded up
		{6, 4, 120, 24, 1}, // 1 x TL from FARM-6
		{10, 100000, 5000000, 1000000, 1},
	} {
		if got := game.FarmFood(tc.tl, tc.qty); got != tc.food {
			t.Errorf("FarmFood(%d, %d) = %d, want %d", tc.tl, tc.qty, got, tc.food)
		}
		if got := game.FarmFuel(tc.tl, tc.qty); got != tc.fuel {
			t.Errorf("FarmFuel(%d, %d) = %d, want %d", tc.tl, tc.qty, got, tc.fuel)
		}
		if got := game.FarmsFuelled(tc.tl, 10); got != tc.perTen {
			t.Errorf("FarmsFuelled(%d, 10) = %d, want %d", tc.tl, got, tc.perTen)
		}
		// The units a quantity of fuel can run never need more than it.
		for fuel := range 40 {
			if n := game.FarmsFuelled(tc.tl, fuel); game.FarmFuel(tc.tl, n) > fuel || game.FarmFuel(tc.tl, n+1) <= fuel {
				t.Errorf("FarmsFuelled(%d, %d) = %d, which needs %d FUEL", tc.tl, fuel, n, game.FarmFuel(tc.tl, n))
			}
		}
	}
	for _, tc := range []struct {
		tl    int
		t     game.EntityType
		orbit int
		want  bool
	}{
		{2, game.CORB, 1, true}, {5, game.CORB, 5, true},
		{2, game.CORB, 6, false}, {1, game.CORB, 3, false}, {6, game.CORB, 3, false},
		{3, game.COPN, 3, false}, {3, game.CENC, 3, false}, {3, game.CSHP, 3, false},
	} {
		if got := game.FarmSolar(tc.tl, tc.t, tc.orbit); got != tc.want {
			t.Errorf("FarmSolar(%d, %s, %d) = %v, want %v", tc.tl, tc.t, tc.orbit, got, tc.want)
		}
	}
}

func TestFarming(t *testing.T) {
	group := func(no int, us ...game.GroupUnits) game.Group { return game.Group{No: no, Units: us} }
	world := func(c game.Colony) game.Data {
		c.PlanetID = 1
		return game.Data{
			Orbits:   []game.Orbit{{ID: 1, OrbitNo: 3}, {ID: 2, OrbitNo: 6}},
			Planets:  []game.Planet{{ID: 1, OrbitID: 1}, {ID: 2, OrbitID: 2}},
			Colonies: []game.Colony{c},
		}
	}
	for _, tc := range []struct {
		name       string
		colony     game.Colony
		planet     int
		want       game.Harvest
		fuel, food int          // left in cargo
		workers    game.Workers // checked if not zero
	}{
		{
			name: "1978 sample",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
				Cargo:         []game.Unit{{Type: "FOOD", Qty: 4269990}, {Type: "FUEL", Qty: 1360000}},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 100000, TechLevel: 10})},
			},
			want: game.Harvest{Units: 100000, Worked: 100000, Fuel: 1000000, Food: 5000000},
			fuel: 360000, food: 9269990,
		},
		{
			name: "short of fuel",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 300000, PRO: 200000},
				Cargo:         []game.Unit{{Type: "FUEL", Qty: 360005}},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 100000, TechLevel: 10})},
			},
			want: game.Harvest{Units: 100000, Worked: 36000, Unfuelled: 64000, Fuel: 360000, Food: 1800000},
			fuel: 5, food: 1800000,
			// The unfuelled farms keep their staff.
			workers: game.Workers{PRO: 200000, USK: 300000, IdlePRO: 100000, IdleUSK: 0},
		},
		{
			name: "short of staff, and the fuel comes from two cargo entries",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{UEM: 1000, USK: 10, PRO: 9, SLD: 1000},
				Cargo:         []game.Unit{{Type: "FUEL", Qty: 2}, {Type: "FOOD", Qty: 1}, {Type: "FUEL", Qty: 100}},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 8, TechLevel: 1})},
			},
			// 10 USK staff 3 units, which burn 1.5 FUEL. A FARM-1 delivers
			// nothing in its first turn of work.
			want: game.Harvest{Units: 8, Worked: 3, Unstaffed: 5, Fuel: 2},
			fuel: 100, food: 1,
		},
		{
			name: "groups in order of seniority; an unfuelled unit keeps its staff",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 12, PRO: 4},
				Cargo: []game.Unit{{Type: "FUEL", Qty: 21}},
				FarmingGroups: []game.Group{
					group(1, game.GroupUnits{Qty: 3, TechLevel: 10}, game.GroupUnits{Qty: 2, TechLevel: 2}),
					group(2, game.GroupUnits{Qty: 5, TechLevel: 1}),
				},
			},
			// Staff: all 3 FARM-10, then 1 of the 2 FARM-2, and none for group
			// 2. Fuel: 20 runs 2 FARM-10, and the last 1 runs the FARM-2.
			want: game.Harvest{Units: 10, Worked: 3, Unstaffed: 6, Unfuelled: 1, Fuel: 21, Food: 110},
			fuel: 0, food: 110,
			workers: game.Workers{PRO: 4, USK: 12, IdlePRO: 0, IdleUSK: 0},
		},
		{
			name: "automation stands in for USK but not PRO",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 3, PRO: 5},
				Operational:   []game.Unit{{Type: "AUTO", TechLevel: 2, Qty: 2}, {Type: "AUTO", TechLevel: 1, Qty: 1}},
				Cargo:         []game.Unit{{Type: "FUEL", Qty: 1000}, {Type: "AUTO", TechLevel: 10, Qty: 50}},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 9, TechLevel: 6})},
			},
			// 3 USK and 9 from the assembled AUTO staff 4 units; the AUTO in
			// cargo does nothing.
			want: game.Harvest{Units: 9, Worked: 4, Unstaffed: 5, Fuel: 24, Food: 120},
			fuel: 976, food: 120,
			workers: game.Workers{PRO: 5, USK: 12, AutoUSK: 9, IdlePRO: 1, IdleUSK: 0},
		},
		{
			name: "solar power in an inner orbit",
			colony: game.Colony{
				Type: game.CORB, Population: game.Population{USK: 30, PRO: 10},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 4, TechLevel: 5}, game.GroupUnits{Qty: 4, TechLevel: 6})},
			},
			want: game.Harvest{Units: 8, Worked: 4, Unfuelled: 4, Food: 100},
			food: 100,
		},
		{
			name: "no solar power in an outer orbit",
			colony: game.Colony{
				Type: game.CORB, Population: game.Population{USK: 30, PRO: 10},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 4, TechLevel: 5})},
			},
			planet: 2,
			want:   game.Harvest{Units: 4, Unfuelled: 4},
		},
		{
			name:   "no farms",
			colony: game.Colony{Type: game.COPN, Population: game.Population{USK: 30, PRO: 10}, Cargo: []game.Unit{{Type: "FUEL", Qty: 7}}},
			fuel:   7,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := world(tc.colony)
			if tc.planet != 0 {
				d.Colonies[0].PlanetID = tc.planet
			}
			before := d.Colonies[0].Population
			engine.Farming(&d, 1, engine.AllocateResources(&d, 1))
			c := d.Colonies[0]
			if c.Harvest != tc.want {
				t.Errorf("harvest = %+v, want %+v", c.Harvest, tc.want)
			}
			fuel, food := 0, 0
			for _, u := range c.Cargo {
				switch u.Type {
				case "FUEL":
					fuel += u.Qty
				case "FOOD":
					food += u.Qty
				}
			}
			if fuel != tc.fuel || food != tc.food {
				t.Errorf("cargo holds %d FUEL and %d FOOD, want %d and %d", fuel, food, tc.fuel, tc.food)
			}
			if tc.workers != (game.Workers{}) && c.Workers != tc.workers {
				t.Errorf("workers = %+v, want %+v", c.Workers, tc.workers)
			}
			if c.Population != before {
				t.Errorf("population changed: %+v", c.Population)
			}
		})
	}
	t.Run("setup turn", func(t *testing.T) {
		d := world(game.Colony{
			Type: game.COPN, Population: game.Population{USK: 30, PRO: 10}, Harvest: game.Harvest{Units: 9},
			Cargo:         []game.Unit{{Type: "FUEL", Qty: 100}},
			FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 4, TechLevel: 10})},
		})
		engine.Farming(&d, 0, engine.AllocateResources(&d, 0))
		if c := d.Colonies[0]; c.Harvest != (game.Harvest{}) || c.Workers != (game.Workers{}) || c.Cargo[0].Qty != 100 || len(c.Cargo) != 1 {
			t.Errorf("harvest %+v, cargo %+v; want nothing farmed", c.Harvest, c.Cargo)
		}
	})
}

func TestAdvanceCrop(t *testing.T) {
	for _, tc := range []struct {
		name              string
		qty               int
		pipeline          []int
		worked            int
		next              []int
		delivered, ruined int
	}{
		{"first turn", 100, nil, 100, []int{100, 0, 0}, 0, 0},
		{"second turn", 100, []int{100, 0, 0}, 100, []int{0, 100, 0}, 0, 0},
		{"fourth turn delivers", 100, []int{0, 0, 100}, 100, nil, 100, 0},
		{"a turn without work ruins the crop", 100, []int{0, 0, 100}, 0, nil, 0, 100},
		{"idle units with nothing begun lose nothing", 100, nil, 0, nil, 0, 0},
		{"the units closest to delivering work first", 100, []int{10, 20, 30}, 45, []int{0, 0, 15}, 30, 15},
		{"staggered units all working", 100, []int{10, 20, 30}, 100, []int{40, 10, 20}, 30, 0},
		{"units added to the group start from nothing", 150, []int{0, 100, 0}, 150, []int{50, 0, 100}, 0, 0},
		{"units taken out of the group", 50, []int{0, 0, 100}, 50, nil, 50, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			next, delivered, ruined := game.AdvanceCrop(tc.qty, tc.pipeline, tc.worked)
			if !slices.Equal(next, tc.next) || delivered != tc.delivered || ruined != tc.ruined {
				t.Errorf("got %v, %d delivered, %d ruined; want %v, %d, %d", next, delivered, ruined, tc.next, tc.delivered, tc.ruined)
			}
		})
	}
}

// TestFarm1Crop runs a FARM-1 group through turns: it delivers a year's FOOD
// every fourth turn, and a turn without fuel loses the crop.
func TestFarm1Crop(t *testing.T) {
	d := game.Data{
		Orbits:  []game.Orbit{{ID: 1, OrbitNo: 3}},
		Planets: []game.Planet{{ID: 1, OrbitID: 1, Habitable: true, Viability: 25}},
		Colonies: []game.Colony{{
			Type: game.COPN, PlanetID: 1, Population: game.Population{USK: 300, PRO: 100},
			Cargo:         []game.Unit{{Type: "FUEL", Qty: 50 * 6}},
			FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 100, TechLevel: 1}}}},
		}},
	}
	// The fuel lasts 6 turns at 50 a turn. Turn 7 has none; it comes back in
	// turn 8.
	for turn, want := range []game.Harvest{
		1:  {Units: 100, Worked: 100, Fuel: 50},
		2:  {Units: 100, Worked: 100, Fuel: 50},
		3:  {Units: 100, Worked: 100, Fuel: 50},
		4:  {Units: 100, Worked: 100, Fuel: 50, Food: 10000},
		5:  {Units: 100, Worked: 100, Fuel: 50},
		6:  {Units: 100, Worked: 100, Fuel: 50},
		7:  {Units: 100, Unfuelled: 100, Ruined: 100},
		8:  {Units: 100, Worked: 100, Fuel: 50},
		9:  {Units: 100, Worked: 100, Fuel: 50},
		10: {Units: 100, Worked: 100, Fuel: 50},
		11: {Units: 100, Worked: 100, Fuel: 50, Food: 10000},
	} {
		if turn == 0 {
			continue
		}
		if turn == 8 {
			d.Colonies[0].Cargo[0].Qty = 200
		}
		engine.Farming(&d, turn, engine.AllocateResources(&d, turn))
		if got := d.Colonies[0].Harvest; got != want {
			t.Errorf("turn %d: harvest %+v, want %+v", turn, got, want)
		}
	}
	if c := d.Colonies[0]; c.FarmingGroups[0].Units[0].Pipeline != nil || c.Cargo[1].Qty != 20000 {
		t.Errorf("pipeline %v, cargo %+v; want an empty pipeline and 20,000 FOOD", c.FarmingGroups[0].Units[0].Pipeline, c.Cargo)
	}
}

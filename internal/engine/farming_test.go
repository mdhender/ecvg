// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestFarmRules(t *testing.T) {
	for _, tc := range []struct {
		tl, qty            int
		food, fuel, perTen int // perTen: units that 10 FUEL can run
	}{
		{1, 4, 100, 2, 20}, // 25 a turn
		{1, 3, 75, 2, 20},  // 1.5 FUEL, rounded up
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

// roomy is enough structure that a test colony never runs out of space.
var roomy = game.Unit{Type: "STRS", Qty: 50_000_000}

func TestFarming(t *testing.T) {
	group := func(no int, us ...game.GroupUnits) game.Group { return game.Group{No: no, Units: us} }
	world := func(c game.Colony) game.Data {
		c.PlanetID = 1
		if !c.Type.Enclosed() || c.Type == game.CORB {
			c.Operational = append(c.Operational, roomy)
		}
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
		idle       []game.Idle  // of each group's first units; checked if not nil
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
			// 10 USK staff 3 units, which burn 1.5 FUEL and make 25 FOOD each.
			want: game.Harvest{Units: 8, Worked: 3, Unstaffed: 5, Fuel: 2, Food: 75},
			fuel: 100, food: 76,
		},
		{
			name: "groups in order of seniority; an unfuelled unit keeps its staff",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 12, PRO: 4},
				Cargo: []game.Unit{{Type: "FUEL", Qty: 21}},
				// Listed out of order: seniority is the group number.
				FarmingGroups: []game.Group{
					group(3, game.GroupUnits{Qty: 5, TechLevel: 1}),
					group(1, game.GroupUnits{Qty: 3, TechLevel: 10}),
					group(2, game.GroupUnits{Qty: 2, TechLevel: 2}),
				},
			},
			// Staff: all 3 FARM-10, then 1 of the 2 FARM-2, and none for group
			// 3. Fuel: 20 runs 2 FARM-10, and the last 1 runs the FARM-2.
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
			name: "idle by order: the idled units ask for nothing",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
				Cargo:         []game.Unit{{Type: "FUEL", Qty: 1360000}},
				FarmingGroups: []game.Group{{No: 1, IdlePercent: 94, Units: []game.GroupUnits{{Qty: 100000, TechLevel: 10}}}},
			},
			want: game.Harvest{Units: 100000, Worked: 6000, Ordered: 94000, Fuel: 60000, Food: 300000},
			fuel: 1300000, food: 300000,
			workers: game.Workers{PRO: 200000, USK: 300000, IdlePRO: 194000, IdleUSK: 282000},
			idle:    []game.Idle{{Ordered: 94000}},
		},
		{
			name: "both levels of idle, and a group idled by order doesn't end the fuel pass",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 30, PRO: 10},
				Cargo: []game.Unit{{Type: "FUEL", Qty: 35}},
				FarmingGroups: []game.Group{
					{No: 1, IdlePercent: 100, Units: []game.GroupUnits{{Qty: 50, TechLevel: 10}}},
					// 33% of 10 is 3.3, so 4 idle by order; 6 staffed, fuel for 3.
					{No: 2, IdlePercent: 33, Units: []game.GroupUnits{{Qty: 10, TechLevel: 10}}},
					// 4 of the 9 staffed, and the last 5 FUEL runs 2 of them.
					{No: 3, Units: []game.GroupUnits{{Qty: 9, TechLevel: 5}}},
				},
			},
			want: game.Harvest{Units: 69, Worked: 5, Ordered: 54, Unstaffed: 5, Unfuelled: 5, Fuel: 35, Food: 200},
			fuel: 0, food: 200,
			idle: []game.Idle{{Ordered: 50}, {Ordered: 4, Unfuelled: 3}, {Unstaffed: 5, Unfuelled: 2}},
		},
		{
			name: "no room for the harvest",
			colony: game.Colony{
				Type: game.CENC, Population: game.Population{USK: 30, PRO: 10},
				// 1,500 STRN enclose 1,500 / 5 = 300 VU. The 40 population
				// units take 40, the 10 FARM-6 180 (18 each), the 200 STRN in
				// cargo 10, and the 20 FUEL 10 (10 once it is burned): 240
				// used and 60 free, which is room for 20 FOOD at 3 VU each.
				Operational:   []game.Unit{{Type: "STRN", Qty: 1500}, {Type: "FARM", TechLevel: 6, Qty: 10}},
				Cargo:         []game.Unit{{Type: "FUEL", Qty: 80}, {Type: "STRN", Qty: 200}},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 10, TechLevel: 6})},
			},
			want: game.Harvest{Units: 10, Worked: 10, Fuel: 60, Food: 20, Lost: 280},
			fuel: 20, food: 20,
		},
		{
			name: "solar power in an inner orbit",
			colony: game.Colony{
				Type: game.CORB, Population: game.Population{USK: 30, PRO: 10},
				FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 4, TechLevel: 5}), group(2, game.GroupUnits{Qty: 4, TechLevel: 6})},
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
			for j, want := range tc.idle {
				if got := c.FarmingGroups[j].Units[0].Idle; got != want {
					t.Errorf("group %d: idle %+v, want %+v", j+1, got, want)
				}
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
			FarmingGroups: []game.Group{group(1, game.GroupUnits{Qty: 4, TechLevel: 10, Idle: game.Idle{Unfuelled: 2}})},
		})
		engine.Farming(&d, 0, engine.AllocateResources(&d, 0))
		if c := d.Colonies[0]; c.Harvest != (game.Harvest{}) || c.Workers != (game.Workers{}) || c.FarmingGroups[0].Units[0].Idle != (game.Idle{}) || c.Cargo[0].Qty != 100 || len(c.Cargo) != 1 {
			t.Errorf("harvest %+v, cargo %+v; want nothing farmed", c.Harvest, c.Cargo)
		}
	})
}

// TestFarm1 runs a FARM-1 group through turns: it makes 25 FOOD a unit in
// every turn it works, and a turn without fuel costs only that turn's FOOD.
func TestFarm1(t *testing.T) {
	d := game.Data{
		Orbits:  []game.Orbit{{ID: 1, OrbitNo: 3}},
		Planets: []game.Planet{{ID: 1, OrbitID: 1, Habitable: true, Viability: 25}},
		Colonies: []game.Colony{{
			Type: game.COPN, PlanetID: 1, Population: game.Population{USK: 300, PRO: 100},
			Operational:   []game.Unit{roomy},
			Cargo:         []game.Unit{{Type: "FUEL", Qty: 50 * 2}},
			FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 100, TechLevel: 1}}}},
		}},
	}
	// The fuel lasts 2 turns at 50 a turn. Turn 3 has none; it comes back in
	// turn 4.
	for turn, want := range []game.Harvest{
		1: {Units: 100, Worked: 100, Fuel: 50, Food: 2500},
		2: {Units: 100, Worked: 100, Fuel: 50, Food: 2500},
		3: {Units: 100, Unfuelled: 100},
		4: {Units: 100, Worked: 100, Fuel: 50, Food: 2500},
	} {
		if turn == 0 {
			continue
		}
		if turn == 4 {
			d.Colonies[0].Cargo[0].Qty = 50
		}
		engine.Farming(&d, turn, engine.AllocateResources(&d, turn))
		if got := d.Colonies[0].Harvest; got != want {
			t.Errorf("turn %d: harvest %+v, want %+v", turn, got, want)
		}
	}
	if c := d.Colonies[0]; c.Cargo[1].Qty != 7500 {
		t.Errorf("cargo %+v; want 7,500 FOOD", c.Cargo)
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestMineRules(t *testing.T) {
	for _, tc := range []struct {
		tl, qty, yield       int
		output, fuel, perTen int // perTen: units that 10 FUEL can run
	}{
		{1, 1, 13, 3, 1, 20},   // 25 x 13% = 3.25, rounded down; 0.5 FUEL, rounded up
		{1, 4, 13, 13, 2, 20},  // the total is rounded, not each unit
		{1, 3, 1, 0, 2, 20},    // 0.75
		{3, 7, 28, 147, 11, 6}, // 10.5 FUEL
		{10, 100000, 13, 3250000, 500000, 2},
	} {
		if got := game.MineOutput(tc.tl, tc.qty, tc.yield); got != tc.output {
			t.Errorf("MineOutput(%d, %d, %d) = %d, want %d", tc.tl, tc.qty, tc.yield, got, tc.output)
		}
		if got := game.MineFuel(tc.tl, tc.qty); got != tc.fuel {
			t.Errorf("MineFuel(%d, %d) = %d, want %d", tc.tl, tc.qty, got, tc.fuel)
		}
		if got := game.MinesFuelled(tc.tl, 10); got != tc.perTen {
			t.Errorf("MinesFuelled(%d, 10) = %d, want %d", tc.tl, got, tc.perTen)
		}
	}
	for typ, ok := range map[game.EntityType]bool{game.COPN: true, game.CENC: true, game.CORB: false, game.CSHP: false} {
		if err := game.CanAssembleMines(typ); ok != (err == nil) || (err != nil && !errors.Is(err, game.ErrMineSite)) {
			t.Errorf("CanAssembleMines(%s) = %v, want ok %v", typ, err, ok)
		}
	}
}

func TestMining(t *testing.T) {
	mines := func(no, depositID, qty, tl int) game.MiningGroup {
		return game.MiningGroup{No: no, DepositID: depositID, Units: []game.GroupUnits{{Qty: qty, TechLevel: tl}}}
	}
	deposits := func() []game.Deposit {
		return []game.Deposit{
			{ID: 1, Type: game.FUEL, Size: 1_000_000, Yield: 13},
			{ID: 2, Type: game.MTL, Size: 500, Yield: 20},
			{ID: 3, Type: game.GOLD, Size: 9_000, Yield: 4},
		}
	}
	for _, tc := range []struct {
		name   string
		colony game.Colony
		// What is wanted afterwards.
		mined   []game.Mined
		idle    []game.Idle
		sizes   []int // of the deposits
		cargo   []game.Unit
		depot   []game.Unit
		workers game.Workers
	}{
		{
			name: "an open-air colony mines into its depot",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 300, PRO: 100},
				Cargo:        []game.Unit{{Type: "FUEL", Qty: 1000}},
				MiningGroups: []game.MiningGroup{mines(1, 1, 100, 1)},
			},
			// 100 MINE-1 burn 50 FUEL and extract 100 x 25 x 13% = 325.
			mined:   []game.Mined{{Worked: 100, Fuel: 50, Product: 325}},
			idle:    []game.Idle{{}},
			sizes:   []int{999_675, 500, 9_000},
			cargo:   []game.Unit{{Type: "FUEL", Qty: 950}},
			depot:   []game.Unit{{Type: "FUEL", Qty: 325}},
			workers: game.Workers{PRO: 100, USK: 300},
		},
		{
			name: "an enclosed colony keeps MTL in its external depot, and a deposit gives no more than it has",
			colony: game.Colony{
				Type: game.CENC, Population: game.Population{USK: 300, PRO: 100},
				Operational:  []game.Unit{roomy},
				Cargo:        []game.Unit{{Type: "FUEL", Qty: 1000}},
				MiningGroups: []game.MiningGroup{mines(1, 2, 100, 10)},
			},
			// 100 MINE-10 could extract 5,000 MTL; the deposit has 500.
			mined:   []game.Mined{{Worked: 100, Fuel: 500, Product: 500}},
			idle:    []game.Idle{{}},
			sizes:   []int{1_000_000, 0, 9_000},
			cargo:   []game.Unit{{Type: "FUEL", Qty: 500}},
			depot:   []game.Unit{{Type: "MTL", Qty: 500}},
			workers: game.Workers{PRO: 100, USK: 300},
		},
		{
			name: "farms come first, then mines by seniority; fuel is spent from the cargo and then the depot",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 30, PRO: 20},
				Operational: []game.Unit{roomy},
				Cargo:       []game.Unit{{Type: "FUEL", Qty: 10}},
				Depot:       []game.Unit{{Type: "FUEL", Qty: 25}, {Type: "GOLD", Qty: 7}},
				// Listed out of order.
				MiningGroups: []game.MiningGroup{
					mines(2, 1, 9, 2),
					{No: 1, DepositID: 3, IdlePercent: 50, Units: []game.GroupUnits{{Qty: 8, TechLevel: 10}}},
				},
				FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 3, TechLevel: 6}}}},
			},
			// Staff: 3 farms, 4 of group 1's mines (4 idle by order), then 3
			// of group 2's 9. Fuel: 35, less 18 for the farms, less 15 for 3
			// MINE-10 (fuel for 3.4 of them), leaves 2 for 2 MINE-2.
			mined:   []game.Mined{{Worked: 2, Fuel: 2, Product: 13}, {Worked: 3, Fuel: 15, Product: 30}},
			idle:    []game.Idle{{Unstaffed: 6, Unfuelled: 1}, {Ordered: 4, Unfuelled: 1}},
			sizes:   []int{999_987, 500, 8_970},
			cargo:   []game.Unit{{Type: "FUEL", Qty: 0}, {Type: "FOOD", Qty: 90}},
			depot:   []game.Unit{{Type: "FUEL", Qty: 13}, {Type: "GOLD", Qty: 37}},
			workers: game.Workers{PRO: 20, USK: 30, IdlePRO: 10},
		},
		{
			name: "FUEL the cargo has no room for goes back to the deposit",
			colony: game.Colony{
				Type: game.CENC, Population: game.Population{USK: 300, PRO: 100},
				// 2,125 STRC-1 enclose 425 VU, which the 400 population units
				// and the 50 FUEL fill. Burning the FUEL frees 25 VU: room
				// for 50 FUEL.
				Operational:  []game.Unit{{Type: "STRC", TechLevel: 1, Qty: 2125}},
				Cargo:        []game.Unit{{Type: "FUEL", Qty: 50}},
				MiningGroups: []game.MiningGroup{mines(1, 1, 100, 1)},
			},
			mined:   []game.Mined{{Worked: 100, Fuel: 50, Product: 50, Returned: 275}},
			idle:    []game.Idle{{}},
			sizes:   []int{999_950, 500, 9_000},
			cargo:   []game.Unit{{Type: "FUEL", Qty: 50}},
			workers: game.Workers{PRO: 100, USK: 300},
		},
		{
			name: "a missing deposit still costs the fuel",
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 300, PRO: 100},
				Cargo:        []game.Unit{{Type: "FUEL", Qty: 1000}},
				MiningGroups: []game.MiningGroup{mines(1, 99, 10, 2)},
			},
			mined:   []game.Mined{{Worked: 10, Fuel: 10}},
			idle:    []game.Idle{{}},
			sizes:   []int{1_000_000, 500, 9_000},
			cargo:   []game.Unit{{Type: "FUEL", Qty: 990}},
			workers: game.Workers{PRO: 100, USK: 300, IdlePRO: 90, IdleUSK: 270},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := game.Data{Deposits: deposits(), Colonies: []game.Colony{tc.colony}}
			allocs := engine.AllocateResources(&d, 1)
			engine.Farming(&d, 1, allocs)
			engine.Mining(&d, 1, allocs)
			c := d.Colonies[0]
			for j, g := range c.MiningGroups {
				if g.Mined != tc.mined[j] {
					t.Errorf("group %d: mined %+v, want %+v", g.No, g.Mined, tc.mined[j])
				}
				if g.Units[0].Idle != tc.idle[j] {
					t.Errorf("group %d: idle %+v, want %+v", g.No, g.Units[0].Idle, tc.idle[j])
				}
			}
			for j, want := range tc.sizes {
				if got := d.Deposits[j].Size; got != want {
					t.Errorf("deposit %d: size %d, want %d", j+1, got, want)
				}
			}
			if !reflect.DeepEqual(c.Cargo, tc.cargo) || !reflect.DeepEqual(c.Depot, tc.depot) {
				t.Errorf("cargo %+v, depot %+v; want %+v, %+v", c.Cargo, c.Depot, tc.cargo, tc.depot)
			}
			if c.Workers != tc.workers {
				t.Errorf("workers %+v, want %+v", c.Workers, tc.workers)
			}
		})
	}
	t.Run("setup turn", func(t *testing.T) {
		g := mines(1, 1, 100, 1)
		g.Mined = game.Mined{Worked: 5}
		d := game.Data{Deposits: deposits(), Colonies: []game.Colony{{
			Type: game.COPN, Population: game.Population{USK: 300, PRO: 100},
			Cargo: []game.Unit{{Type: "FUEL", Qty: 1000}}, MiningGroups: []game.MiningGroup{g},
		}}}
		allocs := engine.AllocateResources(&d, 0)
		engine.Mining(&d, 0, allocs)
		if c := d.Colonies[0]; c.MiningGroups[0].Mined != (game.Mined{}) || c.Cargo[0].Qty != 1000 || d.Deposits[0].Size != 1_000_000 {
			t.Errorf("mined %+v, cargo %+v, deposit %+v; want nothing mined", c.MiningGroups[0].Mined, c.Cargo, d.Deposits[0])
		}
	})
}

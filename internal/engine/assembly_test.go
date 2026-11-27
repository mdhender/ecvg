// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// builders returns a game for the assembly orders. Planet 1, in orbit 3, has
// room for 100,000 FARM-1 units and has deposits 1 to 3; planet 2 has deposit
// 1. Empire 1 has the open-air colony 1 and the orbital colony 2 at planet 1;
// empire 2 has colony 3 there, which works deposit 2 and has 99,950 FARM-1.
func builders() game.Data {
	type unit = game.Unit
	return game.Data{
		Empires: []game.Empire{{ID: 1}, {ID: 2}},
		Orbits:  []game.Orbit{{ID: 1, OrbitNo: 3}},
		Planets: []game.Planet{{ID: 1, OrbitID: 1, Habitable: true, Viability: 1}, {ID: 2, OrbitID: 1}},
		Deposits: []game.Deposit{
			{ID: 11, PlanetID: 1, DepositNo: 1, Type: game.FUEL, Size: 1_000_000, Yield: 10},
			{ID: 12, PlanetID: 1, DepositNo: 2, Type: game.MTL, Size: 1_000_000, Yield: 10},
			{ID: 13, PlanetID: 1, DepositNo: 3, Type: game.GOLD, Size: 1_000_000, Yield: 10},
			{ID: 14, PlanetID: 2, DepositNo: 1, Type: game.NMTL, Size: 1_000_000, Yield: 10},
		},
		Colonies: []game.Colony{
			{
				ID: 1, EmpireID: 1, PlanetID: 1, Type: game.COPN, Rations: game.DefaultRations,
				Population: game.Population{USK: 5000, PRO: 1000},
				Cadres:     game.Cadres{CWRK: 2},
				Cargo:      []unit{{Type: "FOOD", Qty: 100_000}, {Type: "FUEL", Qty: 100_000}, {Type: "MINE", TechLevel: 10, Qty: 60}, {Type: "FARM", TechLevel: 1, Qty: 500}},
				Depot: []unit{
					{Type: "MINE", TechLevel: 10, Qty: 40}, {Type: "MINE", TechLevel: 9, Qty: 40},
					{Type: "FACT", TechLevel: 1, Qty: 10}, {Type: "FACT", TechLevel: 2, Qty: 5}, {Type: "FARM", TechLevel: 10, Qty: 50},
				},
				FarmingGroups: []game.Group{{No: 4, Units: []game.GroupUnits{{Qty: 10, TechLevel: 10}}}},
			},
			{
				ID: 2, EmpireID: 1, PlanetID: 1, Type: game.CORB, Rations: game.DefaultRations,
				Cargo: []unit{{Type: "MINE", TechLevel: 1, Qty: 10}, {Type: "FARM", TechLevel: 1, Qty: 10}, {Type: "FARM", TechLevel: 2, Qty: 10}},
			},
			{
				ID: 3, EmpireID: 2, PlanetID: 1, Type: game.COPN, Rations: game.DefaultRations,
				FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 99_950, TechLevel: 1}}}},
				MiningGroups:  []game.MiningGroup{{No: 1, DepositID: 12, Units: []game.GroupUnits{{Qty: 5, TechLevel: 1}}}},
			},
		},
	}
}

func TestAssembleGroupOrders(t *testing.T) {
	type unit = game.Unit
	for _, tc := range []struct {
		name  string
		text  string // the orders; all but the last must be accepted
		err   error  // of the last
		check func(t *testing.T, c []game.Colony)
	}{
		{
			name: "a new mining group claims its deposit",
			text: "assemble mining-group on 1 with 60 MINE-10 working 1",
			check: func(t *testing.T, c []game.Colony) {
				want := []game.MiningGroup{{No: 5, DepositID: 11, Units: []game.GroupUnits{{TechLevel: 10}}}}
				work := []game.Assembly{{Kind: game.KindMining, GroupNo: 5, Unit: unit{Type: "MINE", TechLevel: 10, Qty: 60}}}
				if !reflect.DeepEqual(c[0].MiningGroups, want) || !reflect.DeepEqual(c[0].Assembly, work) {
					t.Errorf("groups %+v, work %+v", c[0].MiningGroups, c[0].Assembly)
				}
			},
		},
		{
			name: "adding to a group",
			text: "assemble mining-group on 1 with 60 MINE-10 working 1\nassemble mining-group 5 on 1 add 40 MINE-10",
			check: func(t *testing.T, c []game.Colony) {
				if len(c[0].MiningGroups) != 1 || len(c[0].MiningGroups[0].Units) != 1 || len(c[0].Assembly) != 2 {
					t.Errorf("groups %+v, work %+v", c[0].MiningGroups, c[0].Assembly)
				}
			},
		},
		{name: "units already being assembled can't be promised again", text: "assemble mining-group on 1 with 100 MINE-10 working 1\nassemble mining-group 5 on 1 add 1 MINE-10", err: engine.ErrNotEnoughUnits},
		{
			name: "an order for more units than there are takes them all",
			text: "assemble mining-group on 1 with 60 MINE-10 working 1\nassemble mining-group 5 on 1 add 41 MINE-10",
			check: func(t *testing.T, c []game.Colony) {
				if len(c[0].Assembly) != 2 || c[0].Assembly[1].Unit.Qty != 40 {
					t.Errorf("work %+v, want the second order to take the 40 left", c[0].Assembly)
				}
			},
		},
		{name: "none of the units", text: "assemble mining-group on 1 with 5 MINE-8 working 1", err: engine.ErrNotEnoughUnits},
		{name: "a mining group is of one tech level", text: "assemble mining-group on 1 with 60 MINE-10 working 1\nassemble mining-group 5 on 1 add 5 MINE-9", err: engine.ErrTechLevel},
		{name: "a farming group is of one tech level", text: "assemble mining-group on 1 with 60 MINE-10 working 1\nassemble mining-group 5 on 1 add 5 MINE-9", err: engine.ErrTechLevel},
		{name: "a deposit another empire works", text: "assemble mining-group on 1 with 60 MINE-10 working 2", err: engine.ErrDepositClaimed},
		{name: "a deposit the empire works", text: "assemble mining-group on 1 with 60 MINE-10 working 1\nassemble mining-group on 1 with 10 MINE-9 working 1", err: engine.ErrDepositClaimed},
		{name: "no such deposit on the planet", text: "assemble mining-group on 1 with 60 MINE-10 working 4", err: engine.ErrNoSuchDeposit},
		{name: "another empire's entity", text: "assemble mining-group on 3 with 60 MINE-10 working 1", err: engine.ErrNotControlled},
		{name: "no such group", text: "assemble mining-group 7 on 1 add 60 MINE-10", err: engine.ErrNoSuchGroup},
		{name: "an orbital colony can't mine", text: "assemble mining-group on 2 with 10 MINE-1 working 1", err: game.ErrMineSite},
		{name: "FARM-1 needs an open-air colony", text: "assemble farming-group on 2 with 10 FARM-1", err: game.ErrFarmSite},
		{
			name: "the planet's FARM-1 cap is shared",
			text: "assemble farming-group on 1 with 30 FARM-1\nassemble farming-group on 1 with 20 FARM-1",
			check: func(t *testing.T, c []game.Colony) {
				// The new group is junior to group 4.
				if g := c[0].FarmingGroups; len(g) != 2 || g[1].No != 5 || len(c[0].Assembly) != 2 || c[0].LastGroupNo != 5 {
					t.Errorf("groups %+v, work %+v", g, c[0].Assembly)
				}
			},
		},
		{name: "the FARM-1 cap counts units being assembled", text: "assemble farming-group on 1 with 30 FARM-1\nassemble farming-group on 1 with 21 FARM-1", err: engine.ErrFarmCap},
		{
			name: "an orbital colony farms with FARM-2",
			text: "assemble farming-group on 2 with 10 FARM-2",
			check: func(t *testing.T, c []game.Colony) {
				if len(c[1].FarmingGroups) != 1 || len(c[1].Assembly) != 1 {
					t.Errorf("groups %+v, work %+v", c[1].FarmingGroups, c[1].Assembly)
				}
			},
		},
		{
			name: "a factory group has a goal",
			text: "assemble factory-group on 1 with 10 FACT-1 producing CNGD",
			check: func(t *testing.T, c []game.Colony) {
				want := []game.Group{{No: 5, Producing: unit{Type: "CNGD"}, Units: []game.GroupUnits{{TechLevel: 1}}}}
				if !reflect.DeepEqual(c[0].FactoryGroups, want) || len(c[0].Assembly) != 1 {
					t.Errorf("groups %+v, work %+v", c[0].FactoryGroups, c[0].Assembly)
				}
			},
		},
		{name: "a factory order needs the units", text: "assemble factory-group on 1 with 11 FACT-9 producing CNGD", err: engine.ErrNotEnoughUnits},
		{
			name: "a factory order takes the units there are",
			text: "assemble factory-group on 1 with 11 FACT-1 producing CNGD",
			check: func(t *testing.T, c []game.Colony) {
				if len(c[0].Assembly) != 1 || c[0].Assembly[0].Unit.Qty != 10 {
					t.Errorf("work %+v, want 10 FACT-1", c[0].Assembly)
				}
			},
		},
		{
			name: "a factory group is of one tech level",
			text: "assemble factory-group on 1 with 10 FACT-1 producing CNGD\nassemble factory-group 5 on 1 add 5 FACT-2",
			err:  engine.ErrTechLevel,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := builders()
			result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, tc.text)})
			if err != nil {
				t.Fatal(err)
			}
			if tc.err == nil {
				if len(result.Rejected) != 0 {
					t.Fatalf("rejected %+v", result.Rejected)
				}
				tc.check(t, d.Colonies)
				return
			}
			if len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, tc.err) || result.Rejected[0].Line != result.Accepted+1 {
				t.Fatalf("rejected %+v, want the last order rejected with %v", result.Rejected, tc.err)
			}
			// A rejected order changes nothing.
			want := builders()
			if result.Accepted > 0 {
				lines := file(t, 1, tc.text)
				lines.Lines = lines.Lines[:result.Accepted]
				if _, err := engine.ApplyOrders(&want, []orders.File{lines}); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(d.Colonies, want.Colonies) {
				t.Errorf("the rejected order changed the game:\n got %+v\nwant %+v", d.Colonies, want.Colonies)
			}
		})
	}
}

// Assembly comes before production (manual B3.0): the units a group gets in
// turn N produce in turn N.
func TestAssembledGroupWorksThisTurn(t *testing.T) {
	d := builders()
	d.Game.Turn = 1
	// 2 CWRK do 1,000 EMU a turn: 33 MINE-10 at 30 MU each, with 10 over.
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "assemble mining-group on 1 with 60 MINE-10 working 1")}); err != nil {
		t.Fatal(err)
	}
	c := d.Colonies[0]
	g := c.MiningGroups[0]
	// The 33 worked: 33 x 25 x 10 x 10% = 825 FUEL for 165.
	if g.Units[0].Qty != 33 || g.Mined != (game.Mined{Worked: 33, Fuel: 165, Product: 825}) || c.Assembly[0].EMU != 10 || c.Construction != (game.Construction{Pool: 1000, Used: 1000}) {
		t.Fatalf("turn 1: group %+v, work %+v, construction %+v", g, c.Assembly, c.Construction)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	// The other 27 were assembled, 810 EMU of work, and all 60 worked.
	c = d.Colonies[0]
	g = c.MiningGroups[0]
	if g.Mined != (game.Mined{Worked: 60, Fuel: 300, Product: 1500}) || g.Units[0].Qty != 60 || c.Construction != (game.Construction{Pool: 1000, Used: 800}) {
		t.Errorf("turn 2: group %+v, construction %+v", g, c.Construction)
	}
}

// The setup turn is for setting up: assembly work is done in it.
func TestAssemblyInTheSetupTurn(t *testing.T) {
	d := builders()
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "assemble farming-group on 1 with 50 FARM-10")}); err != nil {
		t.Fatal(err)
	}
	// A FARM-10 is 26 MU: 1,000 EMU assemble 38 of them.
	if c := d.Colonies[0]; c.FarmingGroups[0].Units[0].Qty != 48 || c.Harvest != (game.Harvest{}) {
		t.Errorf("group %+v, harvest %+v", c.FarmingGroups[0].Units, c.Harvest)
	}
}

// An entity's farming, mining, and factory groups share one run of group
// numbers, which always goes up and never uses a number again.
func TestGroupNumbersAreShared(t *testing.T) {
	d := builders()
	d.Colonies[0].LastGroupNo = 8 // groups 5 to 8 have come and gone
	text := "assemble mining-group on 1 with 10 MINE-10 working 1\nassemble factory-group on 1 with 5 FACT-1 producing CNGD\nassemble farming-group on 1 with 5 FARM-1\nassemble mining-group on 1 with 10 MINE-9 working 3"
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, text)})
	if err != nil || len(result.Rejected) != 0 {
		t.Fatalf("rejected %+v, %v", result.Rejected, err)
	}
	c := d.Colonies[0]
	if c.MiningGroups[0].No != 9 || c.FactoryGroups[0].No != 10 || c.FarmingGroups[1].No != 11 || c.MiningGroups[1].No != 12 || c.LastGroupNo != 12 {
		t.Errorf("mining %+v, factory %+v, farming %+v, last %d", c.MiningGroups, c.FactoryGroups, c.FarmingGroups, c.LastGroupNo)
	}
}

// An entity may have game.MaxFactoryGroups factory groups and no more.
func TestFactoryGroupLimit(t *testing.T) {
	d := builders()
	c := &d.Colonies[0]
	c.FactoryGroups = nil
	for len(c.FactoryGroups) < game.MaxFactoryGroups-1 {
		c.FactoryGroups = append(c.FactoryGroups, game.Group{No: 100 + len(c.FactoryGroups)})
	}
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, "assemble factory-group on 1 with 1 FACT-1 producing CNGD\nassemble factory-group on 1 with 1 FACT-1 producing CNGD")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted != 1 || len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, engine.ErrTooManyGroups) {
		t.Errorf("accepted %d, rejected %+v; want the second order rejected with %v", result.Accepted, result.Rejected, engine.ErrTooManyGroups)
	}
	if got := len(d.Colonies[0].FactoryGroups); got != game.MaxFactoryGroups {
		t.Errorf("%d factory groups, want %d", got, game.MaxFactoryGroups)
	}
}

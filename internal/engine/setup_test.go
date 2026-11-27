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

// founders is an open-air colony (1) of empire 1 on a habitable planet in
// orbit 3, with deposits 1 and 2, plenty of everything in its depot, 100
// CWRK, and 10 TRNS-10, which carry 200,000 MU a turn; and an open-air colony
// (2) of empire 2 on another planet, which is not habitable.
func founders() game.Data {
	type unit = game.Unit
	return game.Data{
		Empires:  []game.Empire{{ID: 1}, {ID: 2}},
		Orbits:   []game.Orbit{{ID: 1, OrbitNo: 3}, {ID: 2, OrbitNo: 7}},
		Planets:  []game.Planet{{ID: 1, OrbitID: 1, Habitable: true, Viability: 1}, {ID: 2, OrbitID: 2}},
		Deposits: []game.Deposit{{ID: 1, PlanetID: 1, DepositNo: 1}, {ID: 2, PlanetID: 1, DepositNo: 2}},
		Colonies: []game.Colony{
			{
				ID: 1, EmpireID: 1, PlanetID: 1, No: 1, Type: game.COPN, TechLevel: 3, Rations: game.DefaultRations,
				Population: game.Population{UEM: 1000, USK: 1000, PRO: 1000, SLD: 1000},
				Cadres:     game.Cadres{CWRK: 100},
				Cargo:      []unit{{Type: "FOOD", Qty: 1000}, {Type: "LFSU", TechLevel: 1, Qty: 500}},
				Depot: []unit{
					{Type: "FUEL", Qty: 1000}, {Type: "STRN", Qty: 1_000_000}, {Type: "TRNS", TechLevel: 10, Qty: 10},
					{Type: "SDRV", TechLevel: 1, Qty: 10}, {Type: "FACT", TechLevel: 1, Qty: 100},
					{Type: "MINE", TechLevel: 1, Qty: 100}, {Type: "FARM", TechLevel: 6, Qty: 100},
					{Type: "LAB", TechLevel: 1, Qty: 100}, {Type: "MTL", Qty: 500},
				},
			},
			{ID: 2, EmpireID: 2, PlanetID: 2, No: 1, Type: game.COPN, TechLevel: 1},
		},
	}
}

func TestSetupShip(t *testing.T) {
	type unit = game.Unit
	d := founders()
	// 20,000 STRN enclose 2,000 VU and take 2,000 EMU. The 3 CWRK are 6
	// people; with the 10 PRO that is 16 VU. The SDRV-1 is 25 EMU and 25 VU,
	// and the TRNS-10, which moves itself, 40 VU. That leaves 1,919 VU, which
	// is 239 LFSU-1 (8 EMU and 8 VU each), and then 7 VU, which is 2 FOOD.
	errs := apply(t, &d, "setup CSHP from 1\n20,000 STRN\n3 CWRK\n10 PRO\n1 SDRV-1\n1 TRNS-10\n500 LFSU-1\n5 FOOD\nend")
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	if len(d.Colonies) != 3 {
		t.Fatalf("%d colonies, want 3", len(d.Colonies))
	}
	ship := d.Colonies[2]
	if ship.ID != 3 || ship.EmpireID != 1 || ship.PlanetID != 1 || ship.No != 2 || ship.Type != game.CSHP || ship.TechLevel != 3 {
		t.Errorf("ship %+v", ship)
	}
	if ship.Rations != game.DefaultRations || ship.DeathRate != game.DefaultDeathRate {
		t.Errorf("ship rations %d, death rate %v", ship.Rations, ship.DeathRate)
	}
	if want := (game.Population{USK: 3, PRO: 13}); ship.Population != want {
		t.Errorf("ship population %+v, want %+v", ship.Population, want)
	}
	if want := (game.Cadres{CWRK: 3}); ship.Cadres != want {
		t.Errorf("ship cadres %+v, want %+v", ship.Cadres, want)
	}
	wantOp := []unit{{Type: "STRN", Qty: 20000}, {Type: "SDRV", TechLevel: 1, Qty: 1}, {Type: "LFSU", TechLevel: 1, Qty: 239}}
	if !reflect.DeepEqual(ship.Operational, wantOp) {
		t.Errorf("ship operational %+v, want %+v", ship.Operational, wantOp)
	}
	if got := ship.Stored("TRNS", 10); got != 1 {
		t.Errorf("ship has %d TRNS-10 in storage, want 1", got)
	}
	if got := ship.Stored("FOOD", 0); got != 2 {
		t.Errorf("ship has %d FOOD, want 2: it has no room for more", got)
	}
	if free := ship.SpaceFree(); free.Sign() < 0 {
		t.Errorf("ship is over its capacity by %s VU", free)
	}

	home := d.Colonies[0]
	if want := (game.Population{UEM: 1000, USK: 997, PRO: 987, SLD: 1000}); home.Population != want {
		t.Errorf("home population %+v, want %+v", home.Population, want)
	}
	if home.Cadres.CWRK != 97 {
		t.Errorf("home has %d CWRK, want 97", home.Cadres.CWRK)
	}
	if got := home.Stored("STRN", 0); got != 980_000 {
		t.Errorf("home has %d STRN, want 980,000", got)
	}
	if got := home.Stored("LFSU", 1); got != 261 {
		t.Errorf("home has %d LFSU-1, want 261", got)
	}
	// 2,000 + 25 + 1,912 = 3,937 EMU: the ship's 3 CWRK did 1,500 of it, and
	// the helpers the rest.
	if ship.Construction.Setup != 1500 || home.Construction.Setup != 2437 {
		t.Errorf("setup work: ship %d, home %d; want 1500 and 2437", ship.Construction.Setup, home.Construction.Setup)
	}
	// 2,000 + 6 + 10 + 25 + 1,912 + 12 MU carried, with 1 PRO.
	if tr := home.Transport; tr.Used != 3965 || tr.Crew != 1 || len(tr.Transfers) != 7 {
		t.Errorf("home transport %+v", tr)
	}
	if x := home.Transport.Transfers[5]; x.To != 3 || x.Ordered != 500 || x.Unit != (unit{Type: "LFSU", TechLevel: 1, Qty: 239}) {
		t.Errorf("transfer record %+v", x)
	}

	// The work the CWRK did for the set up comes out of the turn's pool.
	engine.Disassembly(&d)
	if k := d.Colonies[0].Construction; k.Pool != 48500 || k.Used != 2437 {
		t.Errorf("home construction %+v, want a pool of 48,500 with 2,437 used", k)
	}
	if k := d.Colonies[2].Construction; k.Pool != 1500 || k.Used != 1500 {
		t.Errorf("ship construction %+v, want a pool of 1,500, all used", k)
	}
}

func TestSetupColonyWithGroups(t *testing.T) {
	d := founders()
	text := "setup CENC from 1\n100,000 STRN\n500 PRO\n1500 USK\n" +
		"10 FACT-1 producing CNGD\n10 FACT-1 stowed\n5 MINE-1 working 2\n5 FARM-6\n5 FARM-6\n3 LAB-1\n40 MTL\n4 SPY\nend"
	if errs := apply(t, &d, text); errs[0] != nil {
		t.Fatal(errs[0])
	}
	c := d.Colonies[2]
	if want := (game.Population{USK: 900, PRO: 500}); c.Population != want {
		t.Errorf("population %+v, want %+v: the CWRK hold 100 of the 1,000 USK", c.Population, want)
	}
	if len(c.FactoryGroups) != 1 || c.FactoryGroups[0].No != 1 || c.FactoryGroups[0].Producing.Type != "CNGD" ||
		game.GroupQty(c.FactoryGroups[0].Units) != 10 {
		t.Errorf("factory groups %+v", c.FactoryGroups)
	}
	if len(c.MiningGroups) != 1 || c.MiningGroups[0].No != 2 || c.MiningGroups[0].DepositID != 2 || game.GroupQty(c.MiningGroups[0].Units) != 5 {
		t.Errorf("mining groups %+v", c.MiningGroups)
	}
	if len(c.FarmingGroups) != 1 || c.FarmingGroups[0].No != 3 || game.GroupQty(c.FarmingGroups[0].Units) != 10 {
		t.Errorf("farming groups %+v: both lines join one group", c.FarmingGroups)
	}
	if len(c.LaboratoryGroups) != 1 || c.LaboratoryGroups[0].No != 4 || game.GroupQty(c.LaboratoryGroups[0].Units) != 3 {
		t.Errorf("laboratory groups %+v", c.LaboratoryGroups)
	}
	if c.InUse("FACT", 1) != 10 || c.Stored("FACT", 1) != 10 {
		t.Errorf("FACT-1: %d in use and %d stored, want 10 and 10", c.InUse("FACT", 1), c.Stored("FACT", 1))
	}
	// An enclosed surface colony keeps MTL in its external depot.
	if !reflect.DeepEqual(c.Depot, []game.Unit{{Type: "MTL", Qty: 40}}) {
		t.Errorf("depot %+v", c.Depot)
	}
	// The colony has no spies, so none go.
	if c.Cadres != (game.Cadres{}) {
		t.Errorf("cadres %+v", c.Cadres)
	}
	// The deposit is taken now.
	if errs := apply(t, &d, "assemble mining-group on 1 with 5 MINE-1 working 2"); !errors.Is(errs[0], engine.ErrDepositClaimed) {
		t.Errorf("assembling on the deposit: %v, want ErrDepositClaimed", errs[0])
	}
}

func TestSetupShortages(t *testing.T) {
	t.Run("the CWRK run out, in the order listed", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Cadres.CWRK = 1
		// 500 EMU: 4,000 STRN take 400, and 4 SDRV-1 the other 100.
		if errs := apply(t, &d, "setup CSHP from 1; 4,000 STRN; 10 SDRV-1; 1 UEM; 10 LFSU-1; end"); errs[0] != nil {
			t.Fatal(errs[0])
		}
		ship := d.Colonies[2]
		if ship.InUse("STRN", 0) != 4000 || ship.InUse("SDRV", 1) != 4 || ship.InUse("LFSU", 1) != 0 || ship.Population.UEM != 1 {
			t.Errorf("ship %+v", ship)
		}
		if got := d.Colonies[0].Stored("SDRV", 1); got != 6 {
			t.Errorf("home keeps %d SDRV-1, want 6", got)
		}
		if got := d.Colonies[0].Construction.Setup; got != 500 {
			t.Errorf("home did %d EMU, want 500", got)
		}
	})
	t.Run("using caps the helpers, for the order or for each ship", func(t *testing.T) {
		for _, tc := range []struct {
			clause string
			strn   [2]int
		}{
			{"using 1 CWRK", [2]int{5000, 0}}, // 500 EMU for both: the second ship gets nothing
			{"using 1 CWRK each", [2]int{5000, 5000}},
		} {
			d := founders()
			apply(t, &d, "setup 2 CSHP from 1 "+tc.clause+"; 6,000 STRN; 1 UEM; 1 SDRV-1 stowed; end")
			var got [2]int
			for i, c := range d.Colonies[2:] {
				got[i] = c.InUse("STRN", 0)
			}
			if got != tc.strn {
				t.Errorf("%s: STRN assembled %v, want %v", tc.clause, got, tc.strn)
			}
		}
	})
	t.Run("the TRNS run out", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[2] = game.Unit{Type: "TRNS", TechLevel: 1, Qty: 1} // 200 MU
		if errs := apply(t, &d, "setup CORB from 1; 1,600 STRN; 100 UEM; 5 FOOD; end"); errs[0] != nil {
			t.Fatal(errs[0])
		}
		// 160 MU of STRN, which enclose 200 VU; the TRNS carries 40 UEM
		// more, and no FOOD.
		c := d.Colonies[2]
		if c.InUse("STRN", 0) != 1600 || c.Population.UEM != 40 || c.Stored("FOOD", 0) != 0 {
			t.Errorf("colony %+v", c)
		}
	})
	t.Run("ships are set up until one fails", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[3].Qty = 2 // SDRV-1
		if errs := apply(t, &d, "setup 5 CSHP from 1; 1,000 STRN; 1 UEM; 1 SDRV-1 stowed; end"); errs[0] != nil {
			t.Fatal(errs[0])
		}
		if len(d.Colonies) != 4 || d.Colonies[3].ID != 4 || d.Colonies[3].No != 3 {
			t.Errorf("%d colonies, want 4: there were only 2 SDRV", len(d.Colonies))
		}
		if got := d.Colonies[0].Stored("STRN", 0); got != 998_000 {
			t.Errorf("home has %d STRN, want 998,000: the third ship took nothing", got)
		}
	})
}

func TestSetupRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*game.Data)
		text string
		err  error
	}{
		{name: "not the empire's entity", text: "setup CSHP from 2; 1 UEM; end", err: engine.ErrNotControlled},
		{name: "no population", text: "setup CSHP from 1; 1,000 STRN; 1 SDRV-1; end", err: engine.ErrNoPopulation},
		{name: "no room for the population", text: "setup CSHP from 1; 1 UEM; 1,000 STRN; 1 SDRV-1; end", err: engine.ErrNoPopulation},
		{name: "a ship with no SDRV", text: "setup CSHP from 1; 1,000 STRN; 1 UEM; end", err: engine.ErrNoSpaceDrive},
		{name: "a second colony of the type", text: "setup COPN from 1; 1 UEM; end", err: engine.ErrOneColonyOfAType},
		{name: "an open-air colony on a planet that isn't habitable", text: "setup COPN from 1; 1 UEM; end", err: engine.ErrNotHabitable,
			edit: func(d *game.Data) { d.Colonies[0].Type = game.CORB; d.Planets[0].Habitable = false }},
		{name: "another empire's planet", text: "setup CORB from 1; 1 UEM; end", err: engine.ErrNeedsPermission,
			edit: func(d *game.Data) { d.Colonies[0].Type = game.CSHP; d.Colonies[1].PlanetID = 1 }},
		{name: "99 ships", text: "setup CSHP from 1; 1 UEM; end", err: engine.ErrTooManyShips,
			edit: func(d *game.Data) {
				for i := 0; i < 99; i++ {
					d.Colonies = append(d.Colonies, game.Colony{ID: 3 + i, EmpireID: 1, PlanetID: 1, Type: game.CSHP})
				}
			}},
		{name: "RSCH", text: "setup CORB from 1; 1 UEM; 5 RSCH; end", err: engine.ErrResearchNeedsPrototype},
		{name: "a factory group in a ship", text: "setup CSHP from 1; 1 UEM; 5 FACT-1 producing CNGD; end", err: engine.ErrShipFactory},
		{name: "mines in orbit", text: "setup CORB from 1; 1 UEM; 5 MINE-1 working 1; end", err: game.ErrMineSite},
		{name: "no such deposit", text: "setup CENC from 1; 1 UEM; 5 MINE-1 working 9; end", err: engine.ErrNoSuchDeposit},
		{name: "one deposit twice", text: "setup CENC from 1; 1 UEM; 5 MINE-1 working 1; 5 MINE-1 working 1; end", err: engine.ErrDepositClaimed},
		{name: "laboratories in a ship", text: "setup CSHP from 1; 1 UEM; 5 LAB-1; end", err: game.ErrLabSite},
		{name: "FARM-1 in an enclosed colony", text: "setup CENC from 1; 1 UEM; 5 FARM-1; end", err: game.ErrFarmSite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := founders()
			if tc.edit != nil {
				tc.edit(&d)
			}
			before := founders()
			if tc.edit != nil {
				tc.edit(&before)
			}
			errs := apply(t, &d, tc.text)
			if !errors.Is(errs[0], tc.err) {
				t.Fatalf("error = %v, want %v", errs[0], tc.err)
			}
			// Only the turn's transport is started; nothing else changes.
			for i := range d.Colonies {
				d.Colonies[i].Transport = game.Transport{}
			}
			if !reflect.DeepEqual(d.Colonies, before.Colonies) {
				t.Errorf("the rejected order changed the colonies:\n%+v\nwant\n%+v", d.Colonies, before.Colonies)
			}
		})
	}
}

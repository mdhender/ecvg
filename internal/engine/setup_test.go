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
					{Type: "SDRV", TechLevel: 1, Qty: 100}, {Type: "FACT", TechLevel: 1, Qty: 100},
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
	// 20,000 STRN enclose 2,000 VU. The 3 SDRV-1 take 75 VU, the TRNS-10 in
	// storage 40, the 200 LFSU-1 1,600, and the 5 FOOD 15; the people come
	// last, whatever the order says, and take 16 of the 270 VU left. The
	// structure, the LFSU, and one SDRV go up with the ship, so the work is
	// the other two SDRV: 50 EMU, which the ship's own CWRK do.
	errs := apply(t, &d, "setup CSHP from 1\n20,000 STRN\n3 CWRK\n10 PRO\n3 SDRV-1\n1 TRNS-10\n200 LFSU-1\n5 FOOD\nend")
	if errs[0] != nil {
		t.Fatal(errs[0])
	}
	// The order reserves the units: the colony still has them.
	home := d.Colonies[0]
	if len(d.Colonies) != 2 || len(home.Setups) != 1 {
		t.Fatalf("%d colonies and %d set ups, want 2 and 1", len(d.Colonies), len(home.Setups))
	}
	if s := home.Setups[0]; s.Need != 50 || s.EMU != 0 || s.Entity.ID != 3 {
		t.Errorf("set up %+v", s)
	}
	if home.Stored("STRN", 0) != 1_000_000 || home.Claimed("STRN", 0) != 20_000 || home.Stock("FOOD") != 995 {
		t.Errorf("home has %d STRN, %d of them claimed, and %d FOOD to use", home.Stored("STRN", 0), home.Claimed("STRN", 0), home.Stock("FOOD"))
	}
	if home.Population.PRO != 1000 || home.FreePRO() != 890 || home.FreeCWRK() != 97 {
		t.Errorf("home has %d PRO, %d of them free, and %d free CWRK", home.Population.PRO, home.FreePRO(), home.FreeCWRK())
	}

	engine.Setups(&d)
	if len(d.Colonies) != 3 || len(d.Colonies[0].Setups) != 0 {
		t.Fatalf("%d colonies and %d set ups, want 3 and 0", len(d.Colonies), len(d.Colonies[0].Setups))
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
	wantOp := []unit{{Type: "STRN", Qty: 20000}, {Type: "SDRV", TechLevel: 1, Qty: 3}, {Type: "LFSU", TechLevel: 1, Qty: 200}}
	if !reflect.DeepEqual(ship.Operational, wantOp) {
		t.Errorf("ship operational %+v, want %+v", ship.Operational, wantOp)
	}
	wantCargo := []unit{{Type: "TRNS", TechLevel: 10, Qty: 1}, {Type: "FOOD", Qty: 5}}
	if !reflect.DeepEqual(ship.Cargo, wantCargo) || len(ship.Depot) != 0 {
		t.Errorf("ship cargo %+v and depot %+v, want %+v and none", ship.Cargo, ship.Depot, wantCargo)
	}
	home = d.Colonies[0]
	if want := (game.Population{UEM: 1000, USK: 997, PRO: 987, SLD: 1000}); home.Population != want {
		t.Errorf("home population %+v, want %+v", home.Population, want)
	}
	if home.Cadres.CWRK != 97 || home.Stored("STRN", 0) != 980_000 || home.Stored("LFSU", 1) != 300 || home.Stock("FOOD") != 995 {
		t.Errorf("home %+v", home)
	}
	// Nobody was shuttled, so the TRNS did nothing and the CWRK are free.
	if home.Transport.Used != 0 || home.Construction.Setup != 0 {
		t.Errorf("home carried %d MU and did %d EMU, want 0 and 0", home.Transport.Used, home.Construction.Setup)
	}
}

// A set up takes as many turns as the crew needs, and the colony keeps what
// it reserved until the work is done.
func TestSetupOverTurns(t *testing.T) {
	d := founders()
	// 8,000 STRN enclose 1,000 VU; one of the 30 SDRV-1 goes up with the
	// colony and the others take 725 EMU: two turns for one CWRK.
	if errs := apply(t, &d, "setup CORB from 1 using 1 CWRK; 100 PRO; 8,000 STRN; 30 SDRV-1; end"); errs[0] != nil {
		t.Fatal(errs[0])
	}
	engine.Setups(&d)
	home := d.Colonies[0]
	if len(d.Colonies) != 2 || len(home.Setups) != 1 {
		t.Fatalf("%d colonies and %d set ups, want 2 and 1", len(d.Colonies), len(home.Setups))
	}
	if s := home.Setups[0]; s.Need != 725 || s.EMU != 500 || s.Crew != 1 || s.Waiting != "work" {
		t.Errorf("set up %+v", s)
	}
	// The CWRK's round trips take twice its mass of the TRNS.
	if home.Transport.Used != 4 || home.Transport.Fuel != 1 || home.Construction.Setup != 500 {
		t.Errorf("home carried %d MU on %d FUEL and did %d EMU", home.Transport.Used, home.Transport.Fuel, home.Construction.Setup)
	}
	if home.Population.PRO != 1000 || home.Stored("SDRV", 1) != 100 {
		t.Errorf("home has %d PRO and %d SDRV-1: it keeps them until the work is done", home.Population.PRO, home.Stored("SDRV", 1))
	}
	engine.Disassembly(&d)
	if k := d.Colonies[0].Construction; k.Pool != 50000 || k.Used != 500 {
		t.Errorf("home construction %+v, want a pool of 50,000 with 500 used", k)
	}

	apply(t, &d, "")
	engine.Setups(&d)
	if len(d.Colonies) != 3 || len(d.Colonies[0].Setups) != 0 {
		t.Fatalf("%d colonies and %d set ups, want 3 and 0", len(d.Colonies), len(d.Colonies[0].Setups))
	}
	if c := d.Colonies[2]; c.ID != 3 || c.Type != game.CORB || c.Population.PRO != 100 || c.InUse("SDRV", 1) != 30 {
		t.Errorf("colony %+v", c)
	}
	if home := d.Colonies[0]; home.Population.PRO != 900 || home.Stored("STRN", 0) != 992_000 {
		t.Errorf("home has %d PRO and %d STRN", home.Population.PRO, home.Stored("STRN", 0))
	}
}

func TestSetupColonyWithGroups(t *testing.T) {
	d := founders()
	text := "setup CENC from 1\n100,000 STRN\n500 PRO\n900 USK\n" +
		"10 FACT-1 producing CNGD\n10 FACT-1 stowed\n5 MINE-1 working 2\n5 FARM-6\n5 FARM-6\n3 LAB-1\n40 MTL\nend"
	if errs := apply(t, &d, text); errs[0] != nil {
		t.Fatal(errs[0])
	}
	// The deposit is taken from the start.
	if errs := apply(t, &d, "assemble mining-group on 1 with 5 MINE-1 working 2"); !errors.Is(errs[0], engine.ErrDepositClaimed) {
		t.Errorf("assembling on the deposit: %v, want ErrDepositClaimed", errs[0])
	}
	engine.Setups(&d)
	if len(d.Colonies) != 3 {
		t.Fatalf("%d colonies, want 3", len(d.Colonies))
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
	// Nothing goes to a depot.
	if len(c.Depot) != 0 || len(c.DepotOperational) != 0 || c.Stock("MTL") != 40 {
		t.Errorf("depot %+v and %+v, with %d MTL", c.Depot, c.DepotOperational, c.Stock("MTL"))
	}
}

func TestSetupCrews(t *testing.T) {
	// 10,000 STRN enclose 1,000 VU; 20 of the 21 SDRV-1 take 500 EMU, which
	// is a turn for one CWRK.
	const lines = "; 10,000 STRN; 21 SDRV-1; 1 UEM; end"
	t.Run("ships share the crew, or have one each", func(t *testing.T) {
		for _, tc := range []struct {
			clause string
			done   int // ships done after one turn
			used   int // MU the TRNS carried
		}{
			{"using 1 CWRK", 1, 4},
			{"using 1 CWRK each", 2, 8},
			{"", 2, 8},
		} {
			d := founders()
			if errs := apply(t, &d, "setup 2 CSHP from 1 "+tc.clause+lines); errs[0] != nil {
				t.Fatal(errs[0])
			}
			engine.Setups(&d)
			if got := len(d.Colonies) - 2; got != tc.done {
				t.Errorf("%q: %d ships done, want %d", tc.clause, got, tc.done)
			}
			if got := d.Colonies[0].Transport.Used; got != tc.used {
				t.Errorf("%q: the TRNS carried %d MU, want %d", tc.clause, got, tc.used)
			}
			// The other ship is done a turn later.
			apply(t, &d, "")
			engine.Setups(&d)
			if len(d.Colonies) != 4 || d.Colonies[3].ID != 4 || d.Colonies[3].No != 3 {
				t.Errorf("%q: %d colonies after two turns, want 4", tc.clause, len(d.Colonies))
			}
		}
	})
	t.Run("a crew with no TRNS stays home", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[2].Qty = 0 // TRNS-10
		apply(t, &d, "setup CSHP from 1"+lines)
		engine.Setups(&d)
		if s := d.Colonies[0].Setups; len(d.Colonies) != 2 || len(s) != 1 || s[0].EMU != 0 || s[0].Crew != 0 || s[0].Waiting != "work" {
			t.Errorf("%d colonies, set ups %+v", len(d.Colonies), s)
		}
	})
	t.Run("the CWRK of the order need no TRNS", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[2].Qty = 0
		apply(t, &d, "setup CSHP from 1; 1 CWRK"+lines)
		engine.Setups(&d)
		if len(d.Colonies) != 3 || d.Colonies[2].Cadres.CWRK != 1 {
			t.Errorf("%d colonies, want 3", len(d.Colonies))
		}
	})
	t.Run("the order's CWRK count towards the crew", func(t *testing.T) {
		d := founders()
		// 60 of the 61 SDRV-1 take 1,500 EMU. The ship's 2 CWRK are the
		// whole crew of 2, so nobody is shuttled.
		apply(t, &d, "setup CSHP from 1 using 2 CWRK; 2 CWRK; 20,000 STRN; 61 SDRV-1; end")
		engine.Setups(&d)
		home := d.Colonies[0]
		if s := home.Setups; len(s) != 1 || s[0].EMU != 1000 || s[0].Crew != 0 || s[0].Waiting != "work" || home.Transport.Used != 0 {
			t.Errorf("set ups %+v, with %d MU carried", s, home.Transport.Used)
		}
	})
	t.Run("a shortage is made up later, and then the TRNS carry it", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[3].Qty = 5 // SDRV-1
		if errs := apply(t, &d, "setup 2 CSHP from 1"+lines); errs[0] != nil {
			t.Fatal(errs[0])
		}
		engine.Setups(&d)
		// The first ship has the 5 SDRV; the work on both is done.
		home := d.Colonies[0]
		if len(d.Colonies) != 2 || len(home.Setups) != 2 {
			t.Fatalf("%d colonies and %d set ups, want 2 and 2", len(d.Colonies), len(home.Setups))
		}
		for i, s := range home.Setups {
			if !s.Built || s.EMU != 500 || s.Waiting != "units" {
				t.Errorf("set up %d: %+v", i, s)
			}
		}
		if short := home.Setups[0].Short(); !reflect.DeepEqual(short, []game.Unit{{Type: "SDRV", TechLevel: 1, Qty: 16}}) {
			t.Errorf("the first ship is short %+v", short)
		}
		// 20 more: the first ship gets its 16, which are 400 MU for the TRNS.
		d.Colonies[0].Depot[3].Qty += 20
		apply(t, &d, "")
		engine.Setups(&d)
		home = d.Colonies[0]
		if len(d.Colonies) != 3 || len(home.Setups) != 1 || home.Transport.Used != 500 {
			t.Fatalf("%d colonies, %d set ups, %d MU carried; want 3, 1, and 500", len(d.Colonies), len(home.Setups), home.Transport.Used)
		}
		if s := home.Setups[0]; s.Hauled != 100 || home.Reserved("SDRV", 1) != 4 || d.Colonies[2].InUse("SDRV", 1) != 21 {
			t.Errorf("set up %+v", s)
		}
	})
	t.Run("nothing is assembled until the structure is up", func(t *testing.T) {
		d := founders()
		d.Colonies[0].Depot[1].Qty = 5000 // STRN
		apply(t, &d, "setup CSHP from 1"+lines)
		engine.Setups(&d)
		if s := d.Colonies[0].Setups; len(s) != 1 || s[0].Built || s[0].EMU != 0 || s[0].Crew != 0 || s[0].Waiting != "structure" {
			t.Fatalf("set ups %+v", s)
		}
		// Supplies are not late while the structure is going up.
		d.Colonies[0].Depot[1].Qty += 5000
		apply(t, &d, "")
		engine.Setups(&d)
		if len(d.Colonies) != 3 || d.Colonies[0].Transport.Used != 4 {
			t.Errorf("%d colonies and %d MU carried, want 3 and 4", len(d.Colonies), d.Colonies[0].Transport.Used)
		}
	})
	t.Run("a set up can be cancelled", func(t *testing.T) {
		d := founders()
		errs := apply(t, &d, "setup CSHP from 1 using 1 CWRK; 100 PRO; 10,000 STRN; 31 SDRV-1; end\ncancel setup 9\ncancel setup 3")
		if errs[0] != nil || !errors.Is(errs[1], engine.ErrNoSuchSetup) || errs[2] != nil {
			t.Fatalf("errors %v", errs)
		}
		home := d.Colonies[0]
		if len(home.Setups) != 0 || home.FreePRO() != 900 || home.Claimed("STRN", 0) != 0 {
			t.Errorf("home %+v", home)
		}
		engine.Setups(&d)
		if len(d.Colonies) != 2 {
			t.Errorf("%d colonies, want 2", len(d.Colonies))
		}
	})
	t.Run("the people who are left go", func(t *testing.T) {
		d := founders()
		apply(t, &d, "setup CSHP from 1 using 1 CWRK; 10,000 STRN; 38 SDRV-1; 5 UEM; end")
		engine.Setups(&d)
		d.Colonies[0].Population.UEM = 2
		apply(t, &d, "")
		engine.Setups(&d)
		if len(d.Colonies) != 3 || d.Colonies[2].Population.UEM != 2 || d.Colonies[0].Population.UEM != 0 {
			t.Errorf("colonies %+v", d.Colonies)
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
		{name: "no room for the population", text: "setup CSHP from 1; 1 UEM; 1,000 STRN; 8 SDRV-1 stowed; end", err: engine.ErrNoPopulation},
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

// helpers is founders with a ship (5) of empire 1 at the home colony's
// planet: 50 SDRV-1, a little STRN, FOOD and FUEL, one TRNS-1, which carries
// 200 MU a turn, and 10 CWRK.
func helpers() game.Data {
	d := founders()
	d.Colonies = append(d.Colonies, game.Colony{
		ID: 5, EmpireID: 1, PlanetID: 1, No: 2, Type: game.CSHP, TechLevel: 1,
		Population: game.Population{UEM: 50, USK: 20, PRO: 100},
		Cadres:     game.Cadres{CWRK: 10},
		Cargo: []game.Unit{
			{Type: "SDRV", TechLevel: 1, Qty: 50}, {Type: "STRN", Qty: 10}, {Type: "FOOD", Qty: 10},
			{Type: "FUEL", Qty: 10}, {Type: "TRNS", TechLevel: 1, Qty: 1},
		},
	})
	return d
}

func TestAddOn(t *testing.T) {
	d := helpers()
	d.Colonies[0].Depot[3].Qty = 5 // SDRV-1
	d.Colonies[0].Population.UEM = 10
	d.Colonies[0].Cadres.CWRK = 1
	// The set up is short 16 SDRV-1 and 20 UEM. The add-on pledges 10 of the
	// SDRV and, of its 25 UEM, the 20 that are wanted; nobody wants its FOOD.
	errs := apply(t, &d, "setup CSHP from 1; 10,000 STRN; 21 SDRV-1; 30 UEM; end\nadd-on to 6 from 5\n10 SDRV-1\n25 UEM\n5 FOOD\nend")
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("errors %v", errs)
	}
	type unit = game.Unit
	pledged := []unit{{Type: "SDRV", TechLevel: 1, Qty: 10}, {Type: "UEM", Qty: 20}}
	ship := &d.Colonies[2]
	if a := ship.AddOns; len(a) != 1 || a[0].To != 6 || !reflect.DeepEqual(a[0].Want, pledged) || !reflect.DeepEqual(a[0].BOM, pledged) {
		t.Fatalf("add-ons %+v", a)
	}
	if ship.Claimed("SDRV", 1) != 10 || ship.Stock("FOOD") != 10 {
		t.Errorf("the ship has %d SDRV-1 claimed and %d FOOD to use", ship.Claimed("SDRV", 1), ship.Stock("FOOD"))
	}

	// Nothing is delivered in the set up's first turn.
	engine.Setups(&d)
	job := &d.Colonies[0].Setups[0]
	if a := d.Colonies[2].AddOns[0]; a.Waiting != "start" || a.Hauled != 0 || !reflect.DeepEqual(job.Pledged, pledged) || len(job.Delivered) != 0 {
		t.Fatalf("add-on %+v, set up %+v", a, job)
	}

	// The TRNS-1 carries 8 SDRV-1 a turn. The set up's own entity does not
	// supply what is pledged.
	d.Colonies[0].Depot[3].Qty += 6
	apply(t, &d, "")
	engine.Setups(&d)
	job, ship = &d.Colonies[0].Setups[0], &d.Colonies[2]
	if a := ship.AddOns[0]; a.Waiting != "transport" || a.Hauled != 200 || ship.Stored("SDRV", 1) != 42 || ship.Transport.Used != 200 {
		t.Errorf("add-on %+v; the ship has %d SDRV-1", a, ship.Stored("SDRV", 1))
	}
	if !reflect.DeepEqual(job.Delivered, []unit{{Type: "SDRV", TechLevel: 1, Qty: 8}}) ||
		!reflect.DeepEqual(job.Pledged, []unit{{Type: "SDRV", TechLevel: 1, Qty: 2}, {Type: "UEM", Qty: 20}}) {
		t.Errorf("delivered %+v, pledged %+v", job.Delivered, job.Pledged)
	}
	if d.Colonies[0].Reserved("SDRV", 1) != 11 || job.Hauled != 150 {
		t.Errorf("home has %d SDRV-1 reserved and carried %d MU for them, want 11 and 150", d.Colonies[0].Reserved("SDRV", 1), job.Hauled)
	}

	// The rest arrives, the people join the builder until the ship is done,
	// and the add-on is over.
	apply(t, &d, "")
	engine.Setups(&d)
	if len(d.Colonies) != 4 || len(d.Colonies[0].Setups) != 0 || len(d.Colonies[2].AddOns) != 0 {
		t.Fatalf("%d colonies, %d set ups, %d add-ons; want 4, 0, 0", len(d.Colonies), len(d.Colonies[0].Setups), len(d.Colonies[2].AddOns))
	}
	if c := d.Colonies[3]; c.ID != 6 || c.InUse("SDRV", 1) != 21 || c.Population.UEM != 30 {
		t.Errorf("new ship %+v", c)
	}
	if home, ship := d.Colonies[0], d.Colonies[2]; home.Population.UEM != 0 || home.Stored("SDRV", 1) != 0 ||
		ship.Population.UEM != 30 || ship.Stored("SDRV", 1) != 40 || ship.Transport.Used != 70 {
		t.Errorf("home %+v\nship %+v", home, ship)
	}
}

func TestAddOnCrew(t *testing.T) {
	d := helpers()
	d.Colonies[0].Depot[1].Qty = 9_999 // STRN
	d.Colonies[0].Cadres.CWRK = 0
	errs := apply(t, &d, "setup CSHP from 1; 10,000 STRN; 21 SDRV-1; 5 UEM; end\nadd-on to 6 from 5 using 1 CWRK; 1 STRN; end")
	if errs[0] != nil || errs[1] != nil {
		t.Fatalf("errors %v", errs)
	}
	engine.Setups(&d)
	if s := d.Colonies[0].Setups[0]; s.Built || s.Waiting != "structure" {
		t.Fatalf("set up %+v", s)
	}
	// The last STRN arrives, and the ship's CWRK does the 500 EMU of work.
	apply(t, &d, "")
	engine.Setups(&d)
	if len(d.Colonies) != 4 || len(d.Colonies[2].AddOns) != 0 {
		t.Fatalf("%d colonies and %d add-ons, want 4 and 0", len(d.Colonies), len(d.Colonies[2].AddOns))
	}
	if ship := d.Colonies[2]; ship.Transport.Used != 5 || ship.Construction.Setup != 500 || ship.Stored("STRN", 0) != 9 {
		t.Errorf("the ship carried %d MU, did %d EMU, and has %d STRN", ship.Transport.Used, ship.Construction.Setup, ship.Stored("STRN", 0))
	}
}

func TestAddOnEnds(t *testing.T) {
	const text = "setup CSHP from 1; 10,000 STRN; 21 SDRV-1; 5 UEM; end\nadd-on to 6 from 5; 16 SDRV-1; end"
	start := func(t *testing.T) game.Data {
		d := helpers()
		d.Colonies[0].Depot[3].Qty = 5 // SDRV-1
		if errs := apply(t, &d, text); errs[0] != nil || errs[1] != nil {
			t.Fatalf("errors %v", errs)
		}
		engine.Setups(&d)
		apply(t, &d, "")
		engine.Setups(&d) // 8 SDRV-1 are delivered
		return d
	}
	t.Run("cancelled", func(t *testing.T) {
		d := start(t)
		errs := apply(t, &d, "cancel add-on to 9 from 5\ncancel add-on to 6 from 5")
		if !errors.Is(errs[0], engine.ErrNoSuchAddOn) || errs[1] != nil {
			t.Fatalf("errors %v", errs)
		}
		ship, job := d.Colonies[2], d.Colonies[0].Setups[0]
		if len(ship.AddOns) != 0 || ship.Claimed("SDRV", 1) != 0 || len(job.Pledged) != 0 || len(job.Delivered) != 1 {
			t.Errorf("ship %+v\nset up %+v", ship, job)
		}
	})
	t.Run("the set up is cancelled", func(t *testing.T) {
		d := start(t)
		if errs := apply(t, &d, "cancel setup 6"); errs[0] != nil {
			t.Fatal(errs[0])
		}
		// What was delivered is the builder's.
		if home, ship := d.Colonies[0], d.Colonies[2]; len(ship.AddOns) != 0 || ship.Stored("SDRV", 1) != 42 || home.Stored("SDRV", 1) != 13 {
			t.Errorf("home has %d SDRV-1, the ship %d and %d add-ons", home.Stored("SDRV", 1), ship.Stored("SDRV", 1), len(ship.AddOns))
		}
	})
	t.Run("an entity that leaves keeps its reservation", func(t *testing.T) {
		d := start(t)
		d.Colonies[2].PlanetID = 2
		apply(t, &d, "")
		engine.Setups(&d)
		if ship := d.Colonies[2]; len(ship.AddOns) != 1 || ship.AddOns[0].Waiting != "orbit" || ship.Claimed("SDRV", 1) != 8 || ship.Transport.Used != 0 {
			t.Errorf("ship %+v", ship)
		}
	})
}

func TestAddOnRejected(t *testing.T) {
	const setup = "setup CSHP from 1; 10,000 STRN; 21 SDRV-1; 5 UEM; end\n"
	for _, tc := range []struct {
		name, text string
		edit       func(*game.Data)
		err        error
	}{
		{name: "not the empire's entity", text: "add-on to 6 from 2; 5 SDRV-1; end", err: engine.ErrNotControlled},
		{name: "no such set up", text: "add-on to 9 from 5; 5 SDRV-1; end", err: engine.ErrNoSuchSetup},
		{name: "the builder itself", text: "add-on to 6 from 1; 5 SDRV-1; end", err: engine.ErrOwnSetup},
		{name: "somewhere else", text: "add-on to 6 from 5; 5 SDRV-1; end", err: engine.ErrNotSameOrbit,
			edit: func(d *game.Data) { d.Colonies[2].PlanetID = 2 }},
		{name: "nothing the set up is short of", text: "add-on to 6 from 5; 5 FOOD; 5 STRN; end", err: engine.ErrNothingWanted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := helpers()
			d.Colonies[0].Depot[3].Qty = 5 // SDRV-1
			if tc.edit != nil {
				tc.edit(&d)
			}
			errs := apply(t, &d, setup+tc.text)
			if errs[0] != nil || !errors.Is(errs[1], tc.err) {
				t.Fatalf("errors %v, want nil and %v", errs, tc.err)
			}
			if len(d.Colonies[2].AddOns) != 0 || len(d.Colonies[0].Setups[0].Pledged) != 0 {
				t.Errorf("the rejected order changed something")
			}
		})
	}
}

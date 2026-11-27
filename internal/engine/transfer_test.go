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

// traders is an orbiting colony (1) with a TL-2 laboratory group, 250,000
// RSCH, and 2 TRNS-1, which carry 400 MU a turn; a ship (2) of the same
// empire at the same planet, enclosing 1,000 VU; a colony (3) of the same
// empire at another planet; and a colony (4) of another empire.
func traders() game.Data {
	type unit = game.Unit
	return game.Data{
		Empires: []game.Empire{{ID: 1}, {ID: 2}},
		Colonies: []game.Colony{
			{
				ID: 1, EmpireID: 1, PlanetID: 1, Type: game.CORB, TechLevel: 1, Rations: game.DefaultRations,
				Population:  game.Population{USK: 50, PRO: 5},
				Operational: []unit{{Type: "STRN", Qty: 100_000}, {Type: "LAB", TechLevel: 2, Qty: 5}},
				Cargo: []unit{
					{Type: "RSCH", Qty: 250_000}, {Type: "FUEL", Qty: 10}, {Type: "FOOD", Qty: 1000},
					{Type: "TRNS", TechLevel: 1, Qty: 2}, {Type: "HDRV", TechLevel: 1, Qty: 20},
				},
				LaboratoryGroups: []game.LaboratoryGroup{
					{No: 3, ShutDown: 5, Units: []game.GroupUnits{{Qty: 5, TechLevel: 2}}},
					{No: 4},
				},
			},
			{
				ID: 2, EmpireID: 1, PlanetID: 1, Type: game.CSHP, TechLevel: 1, Rations: game.DefaultRations,
				Population:  game.Population{PRO: 2},
				Operational: []unit{{Type: "STRN", Qty: 10_000}},
			},
			{ID: 3, EmpireID: 1, PlanetID: 2, Type: game.COPN, TechLevel: 1},
			{ID: 4, EmpireID: 2, PlanetID: 1, Type: game.COPN, TechLevel: 1},
		},
	}
}

// apply carries out the orders of empire 1 and returns the error of each.
func apply(t *testing.T, d *game.Data, text string) []error {
	t.Helper()
	result, err := engine.ApplyOrders(d, []orders.File{file(t, 1, text)})
	if err != nil {
		t.Fatal(err)
	}
	errs := make([]error, len(d.OrderResults))
	for _, r := range result.Rejected {
		errs[r.Line-1] = r.Err
	}
	return errs
}

func TestAssemblePrototype(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		errs       []error
		rsch, prto int // PRTO-2
	}{
		{name: "one prototype", text: "assemble PRTO-2 on 1 from lab-group 3", errs: []error{nil}, rsch: 150_000, prto: 1},
		{name: "as many as the research pays for",
			text: "assemble prto-2 on 1 from lab-group 3\nassemble PRTO-2 on 1 from laboratory-group 3\nassemble PRTO-2 on 1 from lab-group 3",
			errs: []error{nil, nil, game.ErrNotEnoughResearch}, rsch: 50_000, prto: 2},
		{name: "another tech level", text: "assemble PRTO-1 on 1 from lab-group 3", errs: []error{game.ErrPrototypeTechLevel}, rsch: 250_000},
		{name: "a group with no units", text: "assemble PRTO-2 on 1 from lab-group 4", errs: []error{game.ErrNoLabGroup}, rsch: 250_000},
		{name: "no such group", text: "assemble PRTO-2 on 1 from lab-group 9", errs: []error{game.ErrNoLabGroup}, rsch: 250_000},
		{name: "a ship has no laboratories", text: "assemble PRTO-2 on 2 from lab-group 3", errs: []error{game.ErrNoLabGroup}, rsch: 250_000},
		{name: "another empire's entity", text: "assemble PRTO-2 on 4 from lab-group 3", errs: []error{engine.ErrNotControlled}, rsch: 250_000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := traders()
			errs := apply(t, &d, tc.text)
			for i, want := range tc.errs {
				if !errors.Is(errs[i], want) || (want == nil && errs[i] != nil) {
					t.Errorf("order %d: %v, want %v", i+1, errs[i], want)
				}
			}
			if c := d.Colonies[0]; c.Stock("RSCH") != tc.rsch || c.Stored("PRTO", 2) != tc.prto {
				t.Errorf("RSCH %d, PRTO-2 %d; want %d and %d", c.Stock("RSCH"), c.Stored("PRTO", 2), tc.rsch, tc.prto)
			}
		})
	}
}

func TestTransfer(t *testing.T) {
	type unit = game.Unit
	hdrv := func(n int) unit { return unit{Type: "HDRV", TechLevel: 1, Qty: n} }
	for _, tc := range []struct {
		name string
		edit func(d *game.Data)
		text string
		err  error // of the last order
		// What is wanted afterwards: the sender's transport, less its fleet
		// and capacity, and the units the two entities have in storage.
		want     game.Transport
		from, to unit
	}{
		// 45 MU each, and 2 TRNS-1 carry 400 MU: 8 of them.
		{name: "what the TRNS can carry", text: "transfer 20 HDRV-1 from 1 to 2",
			want: game.Transport{Used: 360, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: hdrv(8), Ordered: 20}}},
			from: hdrv(12), to: hdrv(8)},
		{name: "transfers share the turn's capacity", text: "transfer 5 HDRV-1 from 1 to 2\ntransfer 5 HDRV-1 from 1 to 2",
			want: game.Transport{Used: 360, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: hdrv(5), Ordered: 5}, {To: 2, Unit: hdrv(3), Ordered: 5}}},
			from: hdrv(12), to: hdrv(8)},
		{name: "and then there is none", text: "transfer 8 HDRV-1 from 1 to 2\ntransfer 1 HDRV-1 from 1 to 2", err: game.ErrNoTransport,
			want: game.Transport{Used: 360, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: hdrv(8), Ordered: 8}}},
			from: hdrv(12), to: hdrv(8)},
		{name: "the units there are", text: "transfer 5,000 FUEL from 1 to 2",
			// 10 FUEL, of which the trip burns 1.
			want: game.Transport{Used: 9, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: unit{Type: "FUEL", Qty: 9}, Ordered: 5000}}},
			from: unit{Type: "FUEL"}, to: unit{Type: "FUEL", Qty: 9}},
		// Stowed, a HDRV-1 takes 22.5 VU.
		{name: "the whole units the receiver has room for", text: "transfer 8 HDRV-1 from 1 to 2",
			edit: func(d *game.Data) { d.Colonies[1].Operational[0].Qty = 500 },
			want: game.Transport{Used: 90, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: hdrv(2), Ordered: 8}}},
			from: hdrv(18), to: hdrv(2)},
		{name: "no room", text: "transfer 8 HDRV-1 from 1 to 2", err: game.ErrNoRoom,
			edit: func(d *game.Data) { d.Colonies[1].Operational[0].Qty = 200 }, from: hdrv(20), to: hdrv(0)},
		{name: "no FUEL", text: "transfer 8 HDRV-1 from 1 to 2", err: game.ErrNoTransport,
			edit: func(d *game.Data) { d.Colonies[0].Cargo[1].Qty = 0 }, from: hdrv(20), to: hdrv(0)},
		{name: "no PRO", text: "transfer 8 HDRV-1 from 1 to 2", err: game.ErrNoTransport,
			edit: func(d *game.Data) { d.Colonies[0].Population.PRO = 0 }, from: hdrv(20), to: hdrv(0)},
		{name: "no TRNS", text: "transfer 8 HDRV-1 from 1 to 2", err: game.ErrNoTransport,
			edit: func(d *game.Data) { d.Colonies[0].Cargo[3].Qty = 0 }, from: hdrv(20), to: hdrv(0)},
		{name: "units being assembled stay", text: "assemble 15 HDRV-1 on 1\ntransfer 8 HDRV-1 from 1 to 2",
			want: game.Transport{Used: 225, Crew: 1, Fuel: 1, Transfers: []game.Transfer{{To: 2, Unit: hdrv(5), Ordered: 8}}},
			from: hdrv(15), to: hdrv(5)},
		{name: "none in storage", text: "transfer 8 HDRV-2 from 1 to 2", err: game.ErrNothingToTransfer, from: hdrv(20), to: hdrv(0)},
		// TRNS move themselves, and still do the sender's work this turn; the
		// receiver can't use them until the next.
		{name: "TRNS move themselves", text: "transfer 2 TRNS-1 from 1 to 2\ntransfer 2 HDRV-1 from 1 to 2\ntransfer 1 HDRV-1 from 2 to 1", err: game.ErrNoTransport,
			want: game.Transport{Used: 90, Crew: 1, Fuel: 1, Transfers: []game.Transfer{
				{To: 2, Unit: unit{Type: "TRNS", TechLevel: 1, Qty: 2}, Ordered: 2}, {To: 2, Unit: hdrv(2), Ordered: 2}}},
			from: hdrv(18), to: hdrv(2)},
		{name: "RSCH", text: "transfer 100 RSCH from 1 to 2", err: engine.ErrResearchNeedsPrototype, from: hdrv(20), to: hdrv(0)},
		{name: "to itself", text: "transfer 1 HDRV-1 from 1 to 1", err: engine.ErrSameEntity, from: hdrv(20), to: hdrv(0)},
		{name: "to another planet", text: "transfer 1 HDRV-1 from 1 to 3", err: engine.ErrNotSameOrbit, from: hdrv(20), to: hdrv(0)},
		{name: "to another empire", text: "transfer 1 HDRV-1 from 1 to 4", err: engine.ErrForeignTransfer, from: hdrv(20), to: hdrv(0)},
		{name: "to nowhere", text: "transfer 1 HDRV-1 from 1 to 9", err: engine.ErrNoSuchEntity, from: hdrv(20), to: hdrv(0)},
		{name: "from another empire", text: "transfer 1 HDRV-1 from 4 to 1", err: engine.ErrNotControlled, from: hdrv(20), to: hdrv(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := traders()
			if tc.edit != nil {
				tc.edit(&d)
			}
			errs := apply(t, &d, tc.text)
			for i, err := range errs {
				if last := i == len(errs)-1; (!last || tc.err == nil) && err != nil {
					t.Errorf("order %d: %v", i+1, err)
				} else if last && !errors.Is(err, tc.err) {
					t.Errorf("order %d: %v, want %v", i+1, err, tc.err)
				}
			}
			from, to := d.Colonies[0], d.Colonies[1]
			got := from.Transport
			got.Fleet, got.Capacity = nil, 0
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("transport = %+v, want %+v", got, tc.want)
			}
			if n := from.Stored(tc.from.Type, tc.from.TechLevel); n != tc.from.Qty {
				t.Errorf("sender has %d %s, want %d", n, tc.from.Type, tc.from.Qty)
			}
			if n := to.Stored(tc.to.Type, tc.to.TechLevel); n != tc.to.Qty {
				t.Errorf("receiver has %d %s, want %d", n, tc.to.Type, tc.to.Qty)
			}
			if want := 10 - tc.want.Fuel - tc.to.Qty*btoi(tc.to.Type == "FUEL"); tc.edit == nil && from.Stock("FUEL") != want {
				t.Errorf("sender has %d FUEL, want %d", from.Stock("FUEL"), want)
			}
		})
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// The crew comes off the higher TRNS first, a PRO for every 10 of them, and
// the FUEL is 1 for every 2,000 MU, rounded up over the turn.
func TestTransportCrewAndFuel(t *testing.T) {
	d := traders()
	c := &d.Colonies[0]
	c.Cargo = append(c.Cargo, game.Unit{Type: "TRNS", TechLevel: 3, Qty: 11}, game.Unit{Type: "MTL", Qty: 50_000})
	d.Colonies[1].Operational[0].Qty = 10_000_000
	// 11 TRNS-3 carry 19,800 MU and 2 TRNS-1 another 400.
	for _, tc := range []struct {
		qty, used, crew, fuel int
	}{
		{1, 1, 1, 1}, {1999, 2000, 1, 1}, {1, 2001, 1, 2}, {15_999, 18_000, 1, 9},
		{1, 18_001, 2, 10}, {1800, 19_801, 2, 10},
		// The last FUEL: 10 FUEL move 20,000 MU, though the TRNS could carry 200 more.
		{5000, 20_000, 2, 10},
	} {
		if _, err := c.Transfer(&d.Colonies[1], "MTL", 0, tc.qty); err == nil && c.Transport.Capacity == 0 {
			t.Fatal("transport not started")
		} else if c.Transport.Capacity == 0 {
			c.StartTransport()
			if _, err := c.Transfer(&d.Colonies[1], "MTL", 0, tc.qty); err != nil {
				t.Fatal(err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		if got := c.Transport; got.Used != tc.used || got.Crew != tc.crew || got.Fuel != tc.fuel || got.Capacity != 20_200 {
			t.Errorf("after %d more: %+v, want used %d, crew %d, fuel %d", tc.qty, got, tc.used, tc.crew, tc.fuel)
		}
	}
	if c.Stock("FUEL") != 0 {
		t.Errorf("FUEL = %d, want 0", c.Stock("FUEL"))
	}
}

// Research gets from one entity to another as a prototype, in one turn, and
// the PRO who operated the TRNS are not in the turn's worker cadre.
func TestTransferResearch(t *testing.T) {
	d := traders()
	d.Colonies[1].Cargo = []game.Unit{{Type: "FOOD", Qty: 100}}
	d.Game.Turn = 1 // no worker cadre is made in the setup turn
	text := "assemble PRTO-2 on 1 from lab-group 3\ntransfer 1 PRTO-2 from 1 to 2\nassemble tech-level on 2 with 1 PRTO-2"
	result, err := engine.RunTurn(&d, []orders.File{file(t, 1, text)})
	if err != nil || len(result.Rejected) != 0 {
		t.Fatalf("rejected %+v, %v", result.Rejected, err)
	}
	from, to := d.Colonies[0], d.Colonies[1]
	if from.Stock("RSCH") != 150_000 || from.Stored("PRTO", 2) != 0 || to.Stored("PRTO", 2) != 0 || to.RaisingTechLevel() != 2 {
		t.Errorf("sender RSCH %d, PRTO-2 %d; receiver PRTO-2 %d, raising to %d", from.Stock("RSCH"), from.Stored("PRTO", 2), to.Stored("PRTO", 2), to.RaisingTechLevel())
	}
	if from.Transport.Used != 2 || from.Transport.Crew != 1 || from.Workers.PRO != 4 {
		t.Errorf("transport %+v, workers %+v; want 2 MU, a crew of 1, and 4 PRO in the cadre", from.Transport, from.Workers)
	}
	// The next turn starts afresh.
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	if got := d.Colonies[0]; got.Transport.Used != 0 || got.Transport.Crew != 0 || len(got.Transport.Transfers) != 0 || got.Workers.PRO != 5 {
		t.Errorf("transport %+v, workers %+v; want none, and 5 PRO", got.Transport, got.Workers)
	}
}

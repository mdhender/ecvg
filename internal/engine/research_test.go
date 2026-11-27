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

func TestLabRules(t *testing.T) {
	for _, tc := range []struct {
		tl, qty            int
		rsch, fuel, perTen int // perTen: units that 10 FUEL can run
	}{
		{1, 1, 0, 1, 20}, // 0.25 RSCH, rounded down; 0.5 FUEL, rounded up
		{1, 3, 0, 2, 20}, // 0.75
		{1, 4, 1, 2, 20}, // the total is rounded, not each unit
		{3, 7, 5, 11, 6}, // 5.25 RSCH and 10.5 FUEL
		{10, 1, 2, 5, 2}, // 2.5
		{10, 100000, 250000, 500000, 2},
	} {
		if got := game.LabResearch(game.LabQuarters(tc.tl, tc.qty)); got != tc.rsch {
			t.Errorf("LabResearch(LabQuarters(%d, %d)) = %d, want %d", tc.tl, tc.qty, got, tc.rsch)
		}
		if got := game.LabFuel(tc.tl, tc.qty); got != tc.fuel {
			t.Errorf("LabFuel(%d, %d) = %d, want %d", tc.tl, tc.qty, got, tc.fuel)
		}
		if got := game.LabsFuelled(tc.tl, 10); got != tc.perTen {
			t.Errorf("LabsFuelled(%d, 10) = %d, want %d", tc.tl, got, tc.perTen)
		}
	}
}

func TestResearch(t *testing.T) {
	labs := func(no, qty, tl int) game.LaboratoryGroup {
		return game.LaboratoryGroup{No: no, Units: []game.GroupUnits{{Qty: qty, TechLevel: tl}}}
	}
	for _, tc := range []struct {
		name   string
		turn   int
		colony game.Colony
		// What is wanted afterwards.
		research game.Research
		idle     []game.Idle // of each laboratory group
		cargo    []game.Unit
		depot    []game.Unit
		workers  game.Workers
	}{
		{
			name: "laboratories burn FUEL and make RSCH, which goes into the cargo",
			turn: 1,
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 100, PRO: 300},
				Depot:            []game.Unit{{Type: "FUEL", Qty: 1000}},
				LaboratoryGroups: []game.LaboratoryGroup{labs(1, 100, 1)},
			},
			// 100 LAB-1 take 300 PRO and 100 USK, burn 50 FUEL, and make 25.
			research: game.Research{Units: 100, Worked: 100, Fuel: 50, RSCH: 25},
			idle:     []game.Idle{{}},
			cargo:    []game.Unit{{Type: "RSCH", Qty: 25}},
			depot:    []game.Unit{{Type: "FUEL", Qty: 950}},
			workers:  game.Workers{PRO: 300, USK: 100},
		},
		{
			name: "the colony's total is rounded down, not each group's, and adds to what it has",
			turn: 1,
			colony: game.Colony{
				Type: game.CORB, Population: game.Population{USK: 100, PRO: 300},
				Cargo:            []game.Unit{{Type: "FUEL", Qty: 1000}, {Type: "RSCH", Qty: 40}},
				LaboratoryGroups: []game.LaboratoryGroup{labs(1, 3, 1), labs(2, 1, 3), labs(3, 1, 5)},
			},
			// 0.75 + 0.75 + 1.25 = 2.75 RSCH; FUEL is 2 + 2 + 3. The colony has
			// no structure, so it is over its capacity, and RSCH takes no room.
			research: game.Research{Units: 5, Worked: 5, Fuel: 7, RSCH: 2},
			idle:     []game.Idle{{}, {}, {}},
			cargo:    []game.Unit{{Type: "FUEL", Qty: 993}, {Type: "RSCH", Qty: 42}},
			workers:  game.Workers{PRO: 300, USK: 100, IdlePRO: 285, IdleUSK: 95},
		},
		{
			name: "farms and mines come first, then laboratories by seniority",
			turn: 1,
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 40, PRO: 30},
				Depot: []game.Unit{{Type: "FUEL", Qty: 20}},
				// Listed out of order.
				LaboratoryGroups: []game.LaboratoryGroup{
					labs(9, 10, 2),
					{No: 8, ShutDown: 2, Units: []game.GroupUnits{{Qty: 8, TechLevel: 4}}},
				},
				FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 2, TechLevel: 6}}}},
				MiningGroups:  []game.MiningGroup{{No: 2, Units: []game.GroupUnits{{Qty: 4, TechLevel: 1}}}},
			},
			// Staff: the 2 farms and 4 mines take 6 PRO and 18 USK, leaving 24
			// PRO and 22 USK: 6 of group 8's laboratories (2 are shut down)
			// and 2 of group 9's. Fuel: 20, less 12 for the farms and 2 for
			// the mines, leaves 6 for 3 LAB-4, and none for group 9.
			research: game.Research{Units: 18, Worked: 3, Ordered: 2, Unstaffed: 8, Unfuelled: 5, Fuel: 6, RSCH: 3},
			idle:     []game.Idle{{Unstaffed: 8, Unfuelled: 2}, {Ordered: 2, Unfuelled: 3}},
			cargo:    []game.Unit{{Type: "FOOD", Qty: 60}, {Type: "RSCH", Qty: 3}},
			depot:    []game.Unit{{Type: "FUEL", Qty: 0}},
			workers:  game.Workers{PRO: 30, USK: 40, IdleUSK: 14},
		},
		{
			name: "no research in the setup turn",
			turn: 0,
			colony: game.Colony{
				Type: game.COPN, Population: game.Population{USK: 100, PRO: 300},
				Research:         game.Research{Units: 9},
				Depot:            []game.Unit{{Type: "FUEL", Qty: 1000}},
				LaboratoryGroups: []game.LaboratoryGroup{{No: 1, Units: []game.GroupUnits{{Qty: 100, TechLevel: 1, Idle: game.Idle{Unstaffed: 3}}}}},
			},
			idle:  []game.Idle{{}},
			depot: []game.Unit{{Type: "FUEL", Qty: 1000}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := game.Data{Colonies: []game.Colony{tc.colony}}
			allocs := engine.AllocateResources(&d, tc.turn)
			engine.Farming(&d, tc.turn, allocs)
			engine.Mining(&d, tc.turn, allocs)
			engine.Research(&d, tc.turn, allocs)
			c := d.Colonies[0]
			if c.Research != tc.research {
				t.Errorf("research = %+v, want %+v", c.Research, tc.research)
			}
			for j, g := range c.LaboratoryGroups {
				if g.Units[0].Idle != tc.idle[j] {
					t.Errorf("group %d: idle = %+v, want %+v", g.No, g.Units[0].Idle, tc.idle[j])
				}
			}
			if !reflect.DeepEqual(c.Cargo, tc.cargo) {
				t.Errorf("cargo = %+v, want %+v", c.Cargo, tc.cargo)
			}
			if !reflect.DeepEqual(c.Depot, tc.depot) {
				t.Errorf("depot = %+v, want %+v", c.Depot, tc.depot)
			}
			if c.Workers != tc.workers {
				t.Errorf("workers = %+v, want %+v", c.Workers, tc.workers)
			}
		})
	}
}

// An entity has one laboratory group of each tech level, which the order's
// units join, and the laboratories work in the turn they are assembled.
func TestAssembleLaboratoryGroup(t *testing.T) {
	type unit = game.Unit
	d := builders()
	d.Game.Turn = 1
	c := &d.Colonies[0]
	c.Cadres.CWRK = 100
	c.Cargo = append(c.Cargo, unit{Type: "LAB", TechLevel: 1, Qty: 30})
	c.Depot = append(c.Depot, unit{Type: "LAB", TechLevel: 2, Qty: 8})
	text := `assemble laboratory-group on 1 with 20 LAB-1
assemble laboratory-group on 1 with 8 LAB-2
assemble laboratory-group on 1 with 50 LAB-1 using 99 CWRK
assemble laboratory-group on 1 with 1 LAB-1
assemble laboratory-group on 1 with 1 LAB-3
assemble 1 LAB-2 on 1
disassemble 1 LAB-1 on 1
assemble laboratory-group on 3 with 1 LAB-1`
	result, err := engine.RunTurn(&d, []orders.File{file(t, 1, text)})
	if err != nil {
		t.Fatal(err)
	}
	var got []error
	for _, r := range result.Rejected {
		got = append(got, r.Err)
	}
	// The third order takes the 10 LAB-1 that are left, so there are none for
	// the fourth.
	want := []error{engine.ErrNotEnoughUnits, engine.ErrNotEnoughUnits, engine.ErrGroupUnit, engine.ErrGroupUnit, engine.ErrNotControlled}
	if len(got) != len(want) {
		t.Fatalf("rejected %+v, want %d orders", result.Rejected, len(want))
	}
	for i := range want {
		if !errors.Is(got[i], want[i]) {
			t.Errorf("rejection %d = %v, want %v", i+1, got[i], want[i])
		}
	}
	c = &d.Colonies[0]
	groups := []game.LaboratoryGroup{
		{No: 5, Units: []game.GroupUnits{{Qty: 30, TechLevel: 1}}},
		{No: 6, Units: []game.GroupUnits{{Qty: 8, TechLevel: 2}}},
	}
	if !reflect.DeepEqual(c.LaboratoryGroups, groups) || c.LastGroupNo != 6 {
		t.Errorf("groups %+v, last %d; want %+v, 6", c.LaboratoryGroups, c.LastGroupNo, groups)
	}
	// An open-air colony's laboratories are in its depot. 30 LAB-1 and 8
	// LAB-2 burn 15 + 8 FUEL and make 7.5 + 4 RSCH.
	if n := c.InUse("LAB", 1) + c.InUse("LAB", 2); n != 38 || len(c.Operational) != 0 {
		t.Errorf("%d LAB in use, operational %+v; want 38 in the depot", n, c.Operational)
	}
	if want := (game.Research{Units: 38, Worked: 38, Fuel: 23, RSCH: 11}); c.Research != want {
		t.Errorf("research = %+v, want %+v", c.Research, want)
	}
	if c.Stock("RSCH") != 11 {
		t.Errorf("RSCH = %d, want 11", c.Stock("RSCH"))
	}
}

// Any colony may assemble laboratories. A ship can only carry them.
func TestLaboratorySites(t *testing.T) {
	for typ, ok := range map[game.EntityType]bool{game.COPN: true, game.CENC: true, game.CORB: true, game.CSHP: false} {
		if err := game.CanAssembleLabs(typ); ok != (err == nil) || (err != nil && !errors.Is(err, game.ErrLabSite)) {
			t.Errorf("CanAssembleLabs(%s) = %v, want ok %v", typ, err, ok)
		}
	}
	d := builders()
	c := &d.Colonies[0]
	c.Type = game.CSHP
	c.Cargo = append(c.Cargo, game.Unit{Type: "LAB", TechLevel: 1, Qty: 30})
	result, err := applyAll(&d, []orders.File{file(t, 1, "assemble laboratory-group on 1 with 20 LAB-1")})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, game.ErrLabSite) {
		t.Errorf("rejected %+v, want ErrLabSite", result.Rejected)
	}
	if c := d.Colonies[0]; len(c.LaboratoryGroups) != 0 || len(c.Assembly) != 0 || c.LastGroupNo != 0 {
		t.Errorf("groups %+v, work %+v, last %d; want nothing changed", c.LaboratoryGroups, c.Assembly, c.LastGroupNo)
	}
}

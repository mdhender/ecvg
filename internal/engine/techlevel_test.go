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

func TestTechLevelCosts(t *testing.T) {
	for _, tc := range []struct{ tl, cost, holds int }{
		{1, 0, 50_000}, {2, 100_000, 100_000}, {3, 200_000, 200_000}, {5, 800_000, 800_000},
		{10, 25_600_000, 25_600_000}, {11, 0, 0}, {0, 0, 0},
	} {
		if got := game.TechLevelCost(tc.tl); got != tc.cost {
			t.Errorf("TechLevelCost(%d) = %d, want %d", tc.tl, got, tc.cost)
		}
		if got := game.PrototypeResearch(tc.tl); got != tc.holds {
			t.Errorf("PrototypeResearch(%d) = %d, want %d", tc.tl, got, tc.holds)
		}
	}
	if got := game.TechLevelsCost(1, 4); got != 700_000 {
		t.Errorf("TechLevelsCost(1, 4) = %d, want 700,000", got)
	}
}

// techColony is a TL-1 ship of empire 1 with 1,000 VU of enclosed space, so
// that a tech level is 1,000 EMU of work, and 2 CWRK, who do 1,000 EMU a turn.
func techColony(cargo ...game.Unit) game.Data {
	return game.Data{
		Empires: []game.Empire{{ID: 1}, {ID: 2}},
		Colonies: []game.Colony{
			{
				ID: 1, EmpireID: 1, Type: game.CSHP, TechLevel: 1, Rations: game.DefaultRations,
				Population:  game.Population{USK: 10, PRO: 10},
				Cadres:      game.Cadres{CWRK: 2},
				Operational: []game.Unit{{Type: "STRN", Qty: 10_000}},
				Cargo:       cargo,
			},
			{ID: 2, EmpireID: 2, Type: game.COPN, TechLevel: 1},
		},
	}
}

func TestAssembleTechLevelOrders(t *testing.T) {
	type unit = game.Unit
	rsch := func(n int) unit { return unit{Type: "RSCH", Qty: n} }
	prto := func(tl, n int) unit { return unit{Type: "PRTO", TechLevel: tl, Qty: n} }
	job := func(to, need, maxCWRK int) []game.Assembly {
		return []game.Assembly{{Action: game.WorkTechLevel, Unit: unit{TechLevel: to, Qty: 1}, Need: need, MaxCWRK: maxCWRK}}
	}
	for _, tc := range []struct {
		name  string
		tl    int // of the entity, if not 1
		cargo []unit
		text  string
		err   error
		// What is wanted afterwards, if the order is accepted.
		work  []game.Assembly
		after []unit
	}{
		{name: "one level by default", cargo: []unit{rsch(150_000)}, text: "assemble tech-level on 1",
			work: job(2, 1000, 0), after: []unit{rsch(50_000)}},
		{name: "several levels", cargo: []unit{rsch(700_000)}, text: "assemble tech-level on 1 to TL-4 using 1 CWRK",
			work: job(4, 3000, 1), after: []unit{rsch(0)}},
		{name: "from a higher level", tl: 4, cargo: []unit{rsch(2_400_000)}, text: "assemble tech-level on 1 to tl-6",
			work: job(6, 2000, 0), after: []unit{rsch(0)}},
		{name: "all or nothing", cargo: []unit{rsch(699_999)}, text: "assemble tech-level on 1 to TL-4", err: game.ErrNotEnoughResearch},
		{name: "no research", text: "assemble tech-level on 1", err: game.ErrNotEnoughResearch},
		{name: "prototypes are not used unless the order names them", cargo: []unit{prto(2, 1)}, text: "assemble tech-level on 1", err: game.ErrNotEnoughResearch},
		{name: "the same level", tl: 3, cargo: []unit{rsch(900_000)}, text: "assemble tech-level on 1 to TL-3", err: game.ErrTechLevelNotHigher},
		{name: "a lower level", tl: 3, cargo: []unit{rsch(900_000)}, text: "assemble tech-level on 1 to TL-2", err: game.ErrTechLevelNotHigher},
		{name: "past the highest level", tl: 10, cargo: []unit{rsch(90_000_000)}, text: "assemble tech-level on 1", err: game.ErrTechLevelTooHigh},
		{name: "a second order", cargo: []unit{rsch(900_000)}, text: "assemble tech-level on 1\nassemble tech-level on 1 to TL-3", err: game.ErrTechLevelInProgress,
			work: job(2, 1000, 0), after: []unit{rsch(800_000)}},
		{name: "another empire's entity", cargo: []unit{rsch(900_000)}, text: "assemble tech-level on 2", err: engine.ErrNotControlled},

		{name: "a prototype of the level", cargo: []unit{rsch(5), prto(2, 3)}, text: "assemble tech-level on 1 with 1 PRTO-2",
			work: job(2, 1000, 0), after: []unit{rsch(5), prto(2, 2)}},
		// 100,000 + 200,000 + 400,000 of the 800,000; the rest is lost,
		// since the entity has no laboratories.
		{name: "what is left in a prototype is lost", cargo: []unit{prto(5, 1)}, text: "assemble tech-level on 1 to TL-4 with 1 PRTO-5",
			work: job(4, 3000, 0), after: []unit{prto(5, 0)}},
		{name: "only as many prototypes as it takes", cargo: []unit{prto(1, 9)}, text: "assemble tech-level on 1 to TL-3 with 8 PRTO-1 using 3 CWRK",
			work: job(3, 2000, 3), after: []unit{prto(1, 3)}},
		{name: "the order takes the prototypes there are", cargo: []unit{prto(2, 1)}, text: "assemble tech-level on 1 with 5 PRTO-2",
			work: job(2, 1000, 0), after: []unit{prto(2, 0)}},
		{name: "too few prototypes named", cargo: []unit{prto(1, 9)}, text: "assemble tech-level on 1 to TL-3 with 5 PRTO-1", err: game.ErrNotEnoughResearch},
		{name: "too few prototypes held", cargo: []unit{prto(1, 5), rsch(900_000)}, text: "assemble tech-level on 1 to TL-3 with 6 PRTO-1", err: game.ErrNotEnoughResearch},
		{name: "none of those prototypes", cargo: []unit{prto(3, 5)}, text: "assemble tech-level on 1 with 1 PRTO-2", err: game.ErrNotEnoughResearch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := techColony(tc.cargo...)
			if tc.tl != 0 {
				d.Colonies[0].TechLevel = tc.tl
			}
			before := d.Colonies[0].TechLevel
			result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, tc.text)})
			if err != nil {
				t.Fatal(err)
			}
			c := d.Colonies[0]
			switch {
			case tc.err == nil && len(result.Rejected) != 0:
				t.Fatalf("rejected %+v, want none", result.Rejected)
			case tc.err != nil && (len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, tc.err)):
				t.Fatalf("rejected %+v, want %v", result.Rejected, tc.err)
			}
			work, after := tc.work, tc.after
			if tc.err != nil && work == nil {
				after = tc.cargo
			}
			if !reflect.DeepEqual(c.Assembly, work) {
				t.Errorf("work = %+v, want %+v", c.Assembly, work)
			}
			if !reflect.DeepEqual(c.Cargo, after) {
				t.Errorf("cargo = %+v, want %+v", c.Cargo, after)
			}
			// The level changes only when the work is done.
			if c.TechLevel != before {
				t.Errorf("tech level = %d, want %d still", c.TechLevel, before)
			}
		})
	}
}

// An entity with assembled laboratories, running or not, keeps what is left
// in the prototypes it uses.
func TestLaboratoriesKeepWhatIsLeftInAPrototype(t *testing.T) {
	for _, tc := range []struct {
		name string
		labs []game.LaboratoryGroup
		want int // RSCH
	}{
		{name: "no laboratories", want: 7},
		{name: "an empty group", labs: []game.LaboratoryGroup{{No: 1}}, want: 7},
		{name: "laboratories", labs: []game.LaboratoryGroup{{No: 1, Units: []game.GroupUnits{{Qty: 1, TechLevel: 1}}}}, want: 100_007},
		{name: "shut down", labs: []game.LaboratoryGroup{{No: 1, ShutDown: 1, Units: []game.GroupUnits{{Qty: 1, TechLevel: 1}}}}, want: 100_007},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := techColony(game.Unit{Type: "PRTO", TechLevel: 5, Qty: 2}, game.Unit{Type: "RSCH", Qty: 7})
			c := &d.Colonies[0]
			c.Type, c.LaboratoryGroups = game.CORB, tc.labs
			result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, "assemble tech-level on 1 to TL-4 with 2 PRTO-5")})
			if err != nil || len(result.Rejected) != 0 {
				t.Fatalf("rejected %+v, %v", result.Rejected, err)
			}
			if c.Stock("RSCH") != tc.want || c.Stored("PRTO", 5) != 1 {
				t.Errorf("RSCH %d, PRTO-5 %d; want %d and 1", c.Stock("RSCH"), c.Stored("PRTO", 5), tc.want)
			}
		})
	}
}

// The entity gets no benefit until the whole job is done, however many levels
// it is, and the research is spent at the start.
func TestTechLevelIsRaisedWhenTheWorkIsDone(t *testing.T) {
	d := techColony(game.Unit{Type: "PRTO", TechLevel: 5, Qty: 1}, game.Unit{Type: "FOOD", Qty: 1000})
	text := "assemble tech-level on 1 to TL-4 with 1 PRTO-5"
	for turn, want := range []struct {
		tl, emu, used int
		left          int // jobs
	}{
		{1, 1000, 1000, 1}, // the setup turn: assembly work is done in it
		{1, 2000, 1000, 1},
		{4, 3000, 1000, 1}, // done, and shown as done in the next report
		{4, 0, 0, 0},
	} {
		var files []orders.File
		if turn == 0 {
			files = append(files, file(t, 1, text))
		}
		result, err := engine.RunTurn(&d, files)
		if err != nil || len(result.Rejected) != 0 {
			t.Fatalf("turn %d: rejected %+v, %v", turn, result.Rejected, err)
		}
		c := d.Colonies[0]
		emu := 0
		if len(c.Assembly) > 0 {
			emu = c.Assembly[0].EMU
		}
		if c.TechLevel != want.tl || len(c.Assembly) != want.left || emu != want.emu || c.Construction.Used != want.used {
			t.Errorf("turn %d: TL %d, work %+v, construction %+v; want TL %d, %d EMU", turn, c.TechLevel, c.Assembly, c.Construction, want.tl, want.emu)
		}
		if c.RaisingTechLevel() != 0 != (want.tl == 1) {
			t.Errorf("turn %d: RaisingTechLevel = %d", turn, c.RaisingTechLevel())
		}
	}
	// Once the job is done the entity can start another.
	d.Colonies[0].Cargo = append(d.Colonies[0].Cargo, game.Unit{Type: "RSCH", Qty: 800_000})
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, "assemble tech-level on 1")})
	if err != nil || len(result.Rejected) != 0 || d.Colonies[0].RaisingTechLevel() != 5 {
		t.Errorf("rejected %+v, %v; raising to %d, want 5", result.Rejected, err, d.Colonies[0].RaisingTechLevel())
	}
}

// Disassembly and scrap have first call on the CWRK, then assembly work,
// oldest first, and the tech level takes its turn among it.
func TestTechLevelSharesTheCWRK(t *testing.T) {
	d := techColony(game.Unit{Type: "RSCH", Qty: 100_000}, game.Unit{Type: "HDRV", TechLevel: 1, Qty: 20})
	// 20 HDRV-1 at 45 MU are 900 EMU, which leaves 100 for the tech level.
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "assemble 20 HDRV-1 on 1\nassemble tech-level on 1")}); err != nil {
		t.Fatal(err)
	}
	c := d.Colonies[0]
	if c.TechLevel != 1 || c.Assembly[1].EMU != 100 || c.InUse("HDRV", 1) != 20 {
		t.Errorf("TL %d, work %+v", c.TechLevel, c.Assembly)
	}
}

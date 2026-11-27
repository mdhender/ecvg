// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestAssembleCap(t *testing.T) {
	type unit = game.Unit
	hdrv := func(qty int) unit { return unit{Type: "HDRV", TechLevel: 1, Qty: qty} }

	t.Run("the cap is respected", func(t *testing.T) {
		// An HDRV-1 is 45 MU. 20 of the 100 CWRK do 10,000 EMU: 222 units.
		c := game.Colony{
			Type:     game.COPN,
			Cadres:   game.Cadres{CWRK: 100},
			Depot:    []unit{hdrv(5000)},
			Assembly: []game.Assembly{{Unit: hdrv(5000), MaxCWRK: 20}},
		}
		work(&c)
		if a := c.Assembly[0]; a.Assembled != 222 || a.Unit.Qty != 4778 || a.EMU != 10 ||
			c.Construction != (game.Construction{Pool: 50000, Used: 10000}) ||
			!reflect.DeepEqual(c.DepotOperational, []unit{hdrv(222)}) {
			t.Errorf("work %+v, construction %+v, depot operational %+v", a, c.Construction, c.DepotOperational)
		}
	})

	t.Run("a cap above what the work needs is ignored", func(t *testing.T) {
		c := game.Colony{
			Type:     game.COPN,
			Cadres:   game.Cadres{CWRK: 100},
			Depot:    []unit{hdrv(1)},
			Assembly: []game.Assembly{{Unit: hdrv(1), MaxCWRK: 20000}},
		}
		work(&c)
		if a := c.Assembly[0]; a.Assembled != 1 || a.Unit.Qty != 0 || c.Construction != (game.Construction{Pool: 50000, Used: 45}) {
			t.Errorf("work %+v, construction %+v", a, c.Construction)
		}
	})

	t.Run("an order's cap is on its own work", func(t *testing.T) {
		// FACT-1 is 14 MU and MINE-10 30 MU. The factory order's 1 CWRK does
		// 500 EMU: 35 of the 40 FACT-1, with 10 EMU over.
		fact := unit{Type: "FACT", TechLevel: 1, Qty: 40}
		mine := unit{Type: "MINE", TechLevel: 10, Qty: 5}
		c := game.Colony{
			Type:          game.COPN,
			Cadres:        game.Cadres{CWRK: 10},
			Depot:         []unit{fact, mine},
			FactoryGroups: []game.Group{{No: 1}},
			MiningGroups:  []game.MiningGroup{{No: 2}},
			Assembly: []game.Assembly{
				{Kind: game.KindFactory, GroupNo: 1, Unit: fact, MaxCWRK: 1},
				{Kind: game.KindMining, GroupNo: 2, Unit: mine},
			},
		}
		work(&c)
		if a := c.Assembly; a[0].Assembled != 35 || a[0].EMU != 10 || a[1].Assembled != 5 ||
			c.Construction != (game.Construction{Pool: 5000, Used: 650}) {
			t.Fatalf("turn 1: work %+v, construction %+v", a, c.Construction)
		}
		// The mining work is done and dropped; the rest of the FACT-1 take
		// 60 EMU more.
		work(&c)
		if a := c.Assembly; len(a) != 1 || a[0].Assembled != 5 || c.Construction.Used != 60 {
			t.Errorf("turn 2: work %+v, construction %+v", a, c.Construction)
		}
	})
}

func TestDisassemble(t *testing.T) {
	type unit = game.Unit
	// An enclosed colony's 1,000 STRC-1 make 200 VU. Its 10 LFSU-1 take 80
	// VU assembled and 40 VU stowed, and are 8 MU each; a STRC-1 is 1 MU and
	// takes 1 VU stowed.
	c := game.Colony{
		Type:        game.CENC,
		Cadres:      game.Cadres{CWRK: 1},
		Operational: []unit{{Type: "STRC", TechLevel: 1, Qty: 1000}, {Type: "LFSU", TechLevel: 1, Qty: 10}},
		Assembly: []game.Assembly{
			{Action: game.WorkDisassemble, Unit: unit{Type: "LFSU", TechLevel: 1, Qty: 10}},
			{Action: game.WorkDisassemble, Unit: unit{Type: "STRC", TechLevel: 1, Qty: 1000}},
		},
	}
	work(&c)
	// The LFSU take 80 EMU, and 9 go back to cargo: 1 in 10 is lost. The
	// other 420 EMU are for 420 STRC-1, but each one stowed takes space and
	// makes none: only 148 fit, 14 of them lost, leaving 852 that make 170
	// VU for 36 + 134 VU of cargo.
	if a := c.Assembly; a[0].Assembled != 10 || a[0].Destroyed != 1 || a[0].Unit.Qty != 0 || a[1].Assembled != 148 || a[1].Destroyed != 14 || a[1].EMU != 272 || a[1].Waiting != "room" {
		t.Errorf("work %+v", a)
	}
	wantOp := []unit{{Type: "STRC", TechLevel: 1, Qty: 852}, {Type: "LFSU", TechLevel: 1}}
	wantCargo := []unit{{Type: "LFSU", TechLevel: 1, Qty: 9}, {Type: "STRC", TechLevel: 1, Qty: 134}}
	if !reflect.DeepEqual(c.Operational, wantOp) || !reflect.DeepEqual(c.Cargo, wantCargo) || c.SpaceFree() != 0 {
		t.Errorf("operational %+v, cargo %+v, free %d", c.Operational, c.Cargo, c.SpaceFree())
	}

	t.Run("in the depot", func(t *testing.T) {
		auto := func(qty int) unit { return unit{Type: "AUTO", TechLevel: 1, Qty: qty} }
		c := game.Colony{
			Type:             game.COPN,
			Cadres:           game.Cadres{CWRK: 1},
			DepotOperational: []unit{auto(10)},
			Assembly:         []game.Assembly{{Action: game.WorkDisassemble, Unit: auto(20)}},
		}
		work(&c)
		if a := c.Assembly[0]; a.Assembled != 10 || a.Destroyed != 1 || a.Waiting != "units" || !reflect.DeepEqual(c.Depot, []unit{auto(9)}) || len(c.Cargo) != 0 {
			t.Errorf("work %+v, depot %+v, cargo %+v", a, c.Depot, c.Cargo)
		}
	})
}

// Disassembly and scrap come before assembly, whatever the age of the work,
// and have first call on the CWRK.
func TestDisassemblyComesFirst(t *testing.T) {
	type unit = game.Unit
	auto := func(qty int) unit { return unit{Type: "AUTO", TechLevel: 10, Qty: qty} }
	// An AUTO-10 is 40 MU: 1 CWRK does 12 of them, with 20 EMU over.
	c := game.Colony{
		Type:             game.COPN,
		Cadres:           game.Cadres{CWRK: 1},
		Depot:            []unit{auto(10)},
		DepotOperational: []unit{auto(10)},
		Assembly: []game.Assembly{
			{Unit: auto(10)},
			{Action: game.WorkDisassemble, Unit: auto(10)},
		},
	}
	c.Disassemble()
	if a := c.Assembly; a[0].EMU != 0 || a[1].Assembled != 10 || c.Stored("AUTO", 10) != 19 || c.Construction != (game.Construction{Pool: 500, Used: 400}) {
		t.Fatalf("after disassembly: work %+v, depot %+v, construction %+v", a, c.Depot, c.Construction)
	}
	c.Assemble()
	if a := c.Assembly; a[0].Assembled != 2 || a[0].EMU != 20 || c.InUse("AUTO", 10) != 2 || c.Construction != (game.Construction{Pool: 500, Used: 500}) {
		t.Errorf("after assembly: work %+v, in use %d, construction %+v", a, c.InUse("AUTO", 10), c.Construction)
	}
}

func TestScrap(t *testing.T) {
	type unit = game.Unit
	for _, tc := range []struct {
		name            string
		colony          game.EntityType
		structure       int // STRC-1 assembled
		scrap           unit
		mtl, nmtl, lost int
	}{
		// An HDRV-1 is built from 25 MTL and 20 NMTL.
		{name: "70% is recovered", colony: game.CENC, structure: 20000, scrap: unit{Type: "HDRV", TechLevel: 1, Qty: 100}, mtl: 1750, nmtl: 1400},
		// 10 STRC-1 are built from 7 MTL and 3 NMTL: 3 of them give 1.47
		// and 0.855.
		{name: "the totals are rounded down", colony: game.CENC, structure: 20000, scrap: unit{Type: "STRC", TechLevel: 1, Qty: 3}, mtl: 1, nmtl: 0},
		{name: "what there is no room for is lost", colony: game.CENC, scrap: unit{Type: "HDRV", TechLevel: 1, Qty: 100}, mtl: 1750, nmtl: 1400, lost: 3150},
		{name: "a depot has room", colony: game.COPN, scrap: unit{Type: "HDRV", TechLevel: 1, Qty: 100}, mtl: 1750, nmtl: 1400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := game.Colony{
				Type: tc.colony,
				// 100 HDRV-1 are 4,500 MU, which 15 CWRK scrap at 300 MU each.
				Cadres:      game.Cadres{CWRK: 15},
				Operational: []unit{{Type: "STRC", TechLevel: 1, Qty: tc.structure}, {Type: tc.scrap.Type, TechLevel: tc.scrap.TechLevel, Qty: 7}},
				Cargo:       []unit{tc.scrap},
				Assembly:    []game.Assembly{{Action: game.WorkScrap, Unit: tc.scrap}},
			}
			work(&c)
			a := c.Assembly[0]
			if a.Assembled != tc.scrap.Qty || a.Unit.Qty != 0 || a.MTL != tc.mtl || a.NMTL != tc.nmtl || a.Lost != tc.lost {
				t.Errorf("work %+v", a)
			}
			stored := c.Stored("MTL", 0) + c.Stored("NMTL", 0)
			if stored != tc.mtl+tc.nmtl-tc.lost || c.Stored(tc.scrap.Type, tc.scrap.TechLevel) != 0 {
				t.Errorf("stored %d MTL and NMTL; cargo %+v, depot %+v", stored, c.Cargo, c.Depot)
			}
			// Scrap never takes the units in use.
			if got := c.InUse(tc.scrap.Type, tc.scrap.TechLevel); got != 7+map[bool]int{true: tc.structure}[tc.scrap.Type == "STRC"] {
				t.Errorf("%d in use", got)
			}
		})
	}
}

// work does a turn of the colony's work: the dis-assembly phase, then the
// assembly phase.
func work(c *game.Colony) {
	c.Disassemble()
	c.Assemble()
}

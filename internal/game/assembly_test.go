// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

func TestAssemble(t *testing.T) {
	type unit = game.Unit
	mine10 := func(qty int) unit { return unit{Type: "MINE", TechLevel: 10, Qty: qty} }

	t.Run("work carries over the turns", func(t *testing.T) {
		// 1 CWRK does 500 EMU a turn, and a MINE-10 is 30 MU.
		c := game.Colony{
			Type:         game.COPN,
			Cadres:       game.Cadres{CWRK: 1},
			Cargo:        []unit{mine10(25)},
			Depot:        []unit{mine10(20)},
			MiningGroups: []game.MiningGroup{{No: 1, Units: []game.GroupUnits{{TechLevel: 10}}}},
			Assembly:     []game.Assembly{{Kind: game.KindMining, GroupNo: 1, Unit: mine10(40)}},
		}
		for turn, want := range []struct {
			left, emu, assembled, group, cargo, depot int
			used                                      int
		}{
			{left: 24, emu: 20, assembled: 16, group: 16, cargo: 9, depot: 20, used: 500},
			{left: 7, emu: 10, assembled: 17, group: 33, cargo: 0, depot: 12, used: 500},
			{left: 0, emu: 0, assembled: 7, group: 40, cargo: 0, depot: 5, used: 200},
		} {
			work(&c)
			a := c.Assembly[0]
			if a.Unit.Qty != want.left || a.EMU != want.emu || a.Assembled != want.assembled || a.Waiting != "" ||
				c.MiningGroups[0].Units[0].Qty != want.group || c.Cargo[0].Qty != want.cargo || c.Depot[0].Qty != want.depot ||
				c.Construction != (game.Construction{Pool: 500, Used: want.used}) {
				t.Errorf("turn %d: work %+v, group %+v, cargo %+v, depot %+v, construction %+v; want %+v",
					turn+1, a, c.MiningGroups[0].Units, c.Cargo, c.Depot, c.Construction, want)
			}
			// An open-air colony assembles its groups in its depot.
			if !reflect.DeepEqual(c.DepotOperational, []unit{mine10(want.group)}) || len(c.Operational) != 0 {
				t.Errorf("turn %d: depot operational %+v, operational %+v", turn+1, c.DepotOperational, c.Operational)
			}
		}
		work(&c)
		if len(c.Assembly) != 0 || c.Construction != (game.Construction{Pool: 500}) {
			t.Errorf("after the work was done: %+v, %+v", c.Assembly, c.Construction)
		}
	})

	t.Run("the oldest work comes first", func(t *testing.T) {
		// 20 FACT-10 are 640 MU and a MINE-1 is 12 MU.
		c := game.Colony{
			Type:          game.COPN,
			Cadres:        game.Cadres{CWRK: 1},
			Depot:         []unit{{Type: "FACT", TechLevel: 10, Qty: 20}, {Type: "MINE", TechLevel: 1, Qty: 100}, {Type: "FACT", TechLevel: 2, Qty: 1}},
			FactoryGroups: []game.Group{{No: 2, Units: []game.GroupUnits{{TechLevel: 10}}}},
			MiningGroups:  []game.MiningGroup{{No: 1, Units: []game.GroupUnits{{TechLevel: 1}}}},
			Assembly: []game.Assembly{
				{Kind: game.KindFactory, GroupNo: 2, Unit: unit{Type: "FACT", TechLevel: 10, Qty: 20}},
				{Kind: game.KindMining, GroupNo: 1, Unit: unit{Type: "MINE", TechLevel: 1, Qty: 100}},
			},
		}
		work(&c)
		if a, b := c.Assembly[0], c.Assembly[1]; a.Assembled != 15 || a.EMU != 20 || b.Assembled != 0 || b.EMU != 0 {
			t.Errorf("turn 1: %+v", c.Assembly)
		}
		// The factories need 140 more; the mines get the other 360.
		c.Assembly = append(c.Assembly, game.Assembly{Kind: game.KindFactory, GroupNo: 2, Unit: unit{Type: "FACT", TechLevel: 2, Qty: 1}})
		work(&c)
		if a, b, x := c.Assembly[0], c.Assembly[1], c.Assembly[2]; a.Assembled != 5 || a.Unit.Qty != 0 || b.Assembled != 30 || b.EMU != 0 || x.EMU != 0 {
			t.Errorf("turn 2: %+v", c.Assembly)
		}
		// The mines are older work than the FACT-2 and take the whole pool.
		work(&c)
		if b, x := c.Assembly[0], c.Assembly[1]; b.Assembled != 41 || b.EMU != 8 || x.Assembled != 0 || x.EMU != 0 {
			t.Errorf("turn 3: %+v", c.Assembly)
		}
		// A factory group can mix tech levels.
		work(&c)
		want := []game.GroupUnits{{Qty: 20, TechLevel: 10}, {Qty: 1, TechLevel: 2}}
		if !reflect.DeepEqual(c.FactoryGroups[0].Units, want) || len(c.Assembly) != 2 {
			t.Errorf("turn 4: group %+v, work %+v", c.FactoryGroups[0].Units, c.Assembly)
		}
	})

	t.Run("an enclosed colony needs room for the assembled units", func(t *testing.T) {
		// 2,500 STRN enclose 500 VU. 73 FOOD take 219 and 20 stowed FARM-6
		// 180, which leaves 101. Assembled, a FARM-6 takes 9 VU more.
		c := game.Colony{
			Type:          game.CENC,
			Cadres:        game.Cadres{CWRK: 10},
			Operational:   []unit{{Type: "STRN", Qty: 2500}},
			Cargo:         []unit{{Type: "FOOD", Qty: 73}, {Type: "FARM", TechLevel: 6, Qty: 20}},
			FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{TechLevel: 6}}}},
			Assembly:      []game.Assembly{{Kind: game.KindFarming, GroupNo: 1, Unit: unit{Type: "FARM", TechLevel: 6, Qty: 20}}},
		}
		// Cadres hold people the colony doesn't have here, but the pool only
		// counts the CWRK.
		work(&c)
		a := c.Assembly[0]
		if a.Assembled != 11 || a.Unit.Qty != 9 || a.EMU != 162 || a.Waiting != "room" || c.SpaceFree() != units.Whole(2) ||
			c.FarmingGroups[0].Units[0].Qty != 11 || c.Operational[1] != (unit{Type: "FARM", TechLevel: 6, Qty: 11}) ||
			c.Construction != (game.Construction{Pool: 5000, Used: 360}) {
			t.Errorf("work %+v, free %v, group %+v, operational %+v, construction %+v", a, c.SpaceFree(), c.FarmingGroups[0].Units, c.Operational, c.Construction)
		}
	})

	t.Run("the units must be in storage", func(t *testing.T) {
		c := game.Colony{
			Type:         game.COPN,
			Cadres:       game.Cadres{CWRK: 10},
			Cargo:        []unit{mine10(5)},
			MiningGroups: []game.MiningGroup{{No: 1, Units: []game.GroupUnits{{TechLevel: 10}}}},
			Assembly:     []game.Assembly{{Kind: game.KindMining, GroupNo: 1, Unit: mine10(10)}},
		}
		work(&c)
		if a := c.Assembly[0]; a.Assembled != 5 || a.Unit.Qty != 5 || a.EMU != 150 || a.Waiting != "units" {
			t.Errorf("work %+v", a)
		}
	})

	t.Run("no CWRK, no work", func(t *testing.T) {
		c := game.Colony{
			Type:     game.COPN,
			Cargo:    []unit{mine10(5)},
			Assembly: []game.Assembly{{Kind: game.KindMining, GroupNo: 1, Unit: mine10(5), Assembled: 3, Waiting: "room"}},
		}
		work(&c)
		if a := c.Assembly[0]; a != (game.Assembly{Kind: game.KindMining, GroupNo: 1, Unit: mine10(5)}) || c.Construction != (game.Construction{}) {
			t.Errorf("work %+v, construction %+v", a, c.Construction)
		}
	})
}

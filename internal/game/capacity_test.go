// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

func TestCapacity(t *testing.T) {
	// The same units in each type of entity.
	colony := func(typ game.EntityType) game.Colony {
		return game.Colony{
			Type:       typ,
			Population: game.Population{UEM: 10, USK: 20, PRO: 5, SLD: 5}, // 1 VU each, in the open too
			Operational: []game.Unit{
				{Type: "STRN", Qty: 100},              // 100 STRN equivalents
				{Type: "STRL", Qty: 20},               // 100
				{Type: "STRS", Qty: 5},                // 100
				{Type: "FARM", TechLevel: 10, Qty: 4}, // 26 VU each assembled
				{Type: "AUTO", TechLevel: 1, Qty: 1},  // 4
			},
			Cargo: []game.Unit{
				{Type: "FOOD", Qty: 10},               // 3 each
				{Type: "FUEL", Qty: 5},                // 0.5 each: 2.5
				{Type: "CNGD", Qty: 11},               // 0.3 each: 3.3
				{Type: "FACT", TechLevel: 10, Qty: 2}, // 16 each stowed
				{Type: "STRN", Qty: 30},               // 0.05 each: stowed structure counts
			},
			// The depot is outside the entity.
			Depot:            []game.Unit{{Type: "MTL", Qty: 1_000_000}, {Type: "STRS", Qty: 1000}},
			DepotOperational: []game.Unit{{Type: "MINE", TechLevel: 10, Qty: 1000}, {Type: "STRS", Qty: 1000}},
		}
	}
	// 40 + 104 + 4 + 30 + 2.5 + 3.3 + 32 + 1.5, and never rounded.
	used := units.NewFrac(2173, 10)
	for _, tc := range []struct {
		t        game.EntityType
		ratio    int
		capacity units.Frac
	}{
		{game.COPN, 1, units.Whole(300)},
		{game.CENC, 5, units.Whole(60)},
		{game.CORB, 8, units.NewFrac(75, 2)},
		{game.CSHP, 10, units.Whole(30)},
	} {
		c := colony(tc.t)
		if got := tc.t.StructureRatio(); got != tc.ratio {
			t.Errorf("%s: structure ratio %d, want %d", tc.t, got, tc.ratio)
		}
		if got := c.StructureEquivalents(); got != 300 {
			t.Errorf("%s: %d STRN equivalents, want 300", tc.t, got)
		}
		if got := c.Capacity(); got != tc.capacity {
			t.Errorf("%s: capacity %v, want %v", tc.t, got, tc.capacity)
		}
		if got := c.SpaceUsed(); got != used {
			t.Errorf("%s: used %v, want %v", tc.t, got, used)
		}
		if got := c.SpaceFree(); got != tc.capacity.Sub(used) {
			t.Errorf("%s: free %v, want %v", tc.t, got, tc.capacity.Sub(used))
		}
	}
}

func TestRoomAndStoreCargo(t *testing.T) {
	// An enclosed colony with 10 VU of space, and 1 FUEL (0.5 VU) in cargo.
	colony := func() game.Colony {
		return game.Colony{
			Type:        game.CENC,
			Operational: []game.Unit{{Type: "STRN", Qty: 50}},
			Cargo:       []game.Unit{{Type: "FUEL", Qty: 1}},
		}
	}
	half := units.NewFrac(1, 2)
	for _, tc := range []struct {
		code    string
		tl, qty int
		room    int
		use     units.Frac // the space used afterwards
	}{
		{"FOOD", 0, 2, 2, units.NewFrac(13, 2)},
		{"FOOD", 0, 3, 3, units.NewFrac(19, 2)},
		{"FOOD", 0, 4, 3, units.NewFrac(19, 2)},
		{"FOOD", 0, 1000, 3, units.NewFrac(19, 2)},
		{"FUEL", 0, 1, 1, units.Whole(1)},
		{"FUEL", 0, 100, 19, units.Whole(10)}, // an exact fit is accepted
		{"MTL", 0, 100, 19, units.Whole(10)},
		{"CNGD", 0, 100, 31, units.NewFrac(98, 10)}, // 0.3 VU each; a unit is never split
		{"STRN", 0, 1000, 190, units.Whole(10)},     // 0.05 VU each stowed
		{"FACT", 1, 5, 1, units.Frac{}},             // 7 VU each stowed
		{"FOOD", 0, 0, 0, half},
		{"FOOD", 0, -5, 0, half},
		{"ORE", 0, 5, 0, half},
	} {
		c := colony()
		if got := c.Room(tc.code, tc.tl, tc.qty); got != tc.room {
			t.Errorf("Room(%s-%d, %d) = %d, want %d", tc.code, tc.tl, tc.qty, got, tc.room)
			continue
		}
		if tc.tl != 0 {
			continue
		}
		if got := c.StoreCargo(tc.code, tc.qty); got != tc.room || c.SpaceUsed() != tc.use || c.SpaceFree().Sign() < 0 {
			t.Errorf("StoreCargo(%s, %d) = %d, leaving %v used and %v free; want %d and %v used", tc.code, tc.qty, got, c.SpaceUsed(), c.SpaceFree(), tc.room, tc.use)
		}
	}
	t.Run("the manual's examples", func(t *testing.T) {
		// F3.2.1: with 300 VU remaining, 6,000 of 10,000 STRN are accepted;
		// with 0.025 VU, a STRN is rejected; and with 1 VU, an HDRV-10 is
		// rejected and 50 CSUP at 0.02 VU each are accepted.
		c := game.Colony{Type: game.CSHP, Operational: []game.Unit{{Type: "STRN", Qty: 3000}}}
		if got := c.Room("STRN", 0, 10_000); got != 6000 {
			t.Errorf("300 VU: room for %d STRN, want 6000", got)
		}
		// 27 STRN make 3.375 VU in an orbital colony; a FOOD and a CNGD take
		// 3.3, which leaves room for one stowed STRN (0.05 VU) and no more.
		c = game.Colony{Type: game.CORB, Operational: []game.Unit{{Type: "STRN", Qty: 27}}, Cargo: []game.Unit{{Type: "FOOD", Qty: 1}, {Type: "CNGD", Qty: 1}}}
		if got := c.StoreCargo("STRN", 5); got != 1 || c.SpaceFree() != units.NewFrac(1, 40) {
			t.Errorf("0.075 VU: stored %d STRN, leaving %v; want 1 and 1/40", got, c.SpaceFree())
		}
		if got := c.Room("STRN", 0, 1); got != 0 {
			t.Errorf("0.025 VU: room for %d STRN, want 0", got)
		}
		c = game.Colony{Type: game.CSHP, Operational: []game.Unit{{Type: "STRN", Qty: 10}}}
		if hdrv, csup := c.Room("HDRV", 10, 1), c.Room("CSUP", 0, 50); hdrv != 0 || csup != 50 {
			t.Errorf("1 VU: room for %d HDRV-10 and %d CSUP; want 0 and 50", hdrv, csup)
		}
	})
	t.Run("over capacity", func(t *testing.T) {
		c := colony()
		c.Cargo = append(c.Cargo, game.Unit{Type: "FOOD", Qty: 100})
		if got := c.StoreCargo("FUEL", 1); got != 0 || c.SpaceFree() != units.NewFrac(-581, 2) {
			t.Errorf("stored %d with %v free; want nothing stored and -581/2 free", got, c.SpaceFree())
		}
	})
	t.Run("an open-air colony may hold more than it encloses", func(t *testing.T) {
		c := colony()
		c.Type = game.COPN // 50 STRN enclose 50 VU
		if got := c.StoreCargo("FOOD", 1000); got != 1000 || c.SpaceFree() != units.NewFrac(-5901, 2) {
			t.Errorf("stored %d with %v free; want 1000 stored and -5901/2 free", got, c.SpaceFree())
		}
	})
	t.Run("resources", func(t *testing.T) {
		open, enclosed := colony(), colony()
		open.Type = game.COPN
		enclosed.Operational[0].Qty = 10 // 10 STRN enclose 2 VU, 1.5 of them free
		if got := open.StoreResource("GOLD", 1000); got != 1000 || len(open.Depot) != 1 || open.Depot[0].Qty != 1000 || open.SpaceUsed() != half {
			t.Errorf("open-air: stored %d, depot %+v", got, open.Depot)
		}
		if got := enclosed.StoreResource("GOLD", 1000); got != 3 || len(enclosed.Depot) != 0 || enclosed.SpaceFree().Sign() != 0 {
			t.Errorf("enclosed: stored %d, depot %+v, %v free; want 3 stored in cargo", got, enclosed.Depot, enclosed.SpaceFree())
		}
	})
}

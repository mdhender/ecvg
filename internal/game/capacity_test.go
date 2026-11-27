// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestCapacity(t *testing.T) {
	// The same units in each type of entity.
	colony := func(typ game.EntityType) game.Colony {
		return game.Colony{
			Type:       typ,
			Population: game.Population{UEM: 10, USK: 20, PRO: 5, SLD: 5},
			Operational: []game.Unit{
				{Type: "STRC", TechLevel: 3, Qty: 105}, // 945
				{Type: "STRL", TechLevel: 10, Qty: 20}, // 2,000
				{Type: "FARM", TechLevel: 10, Qty: 4},  // 26 VU each assembled
				{Type: "AUTO", TechLevel: 1, Qty: 1},   // 4
			},
			Cargo: []game.Unit{
				{Type: "FOOD", Qty: 10},               // 3 each
				{Type: "FUEL", Qty: 5},                // 0.5 each: 2.5, rounded up
				{Type: "CNGD", Qty: 11},               // 0.3 each: 3.3, rounded up
				{Type: "FACT", TechLevel: 10, Qty: 2}, // 16 each stowed
				{Type: "STRC", TechLevel: 1, Qty: 20}, // 1 each: stowed structure counts
			},
			// The depot is outside the entity.
			Depot:            []game.Unit{{Type: "MTL", Qty: 1_000_000}, {Type: "STRC", TechLevel: 10, Qty: 1000}},
			DepotOperational: []game.Unit{{Type: "MINE", TechLevel: 10, Qty: 1000}, {Type: "STRC", TechLevel: 10, Qty: 1000}},
		}
	}
	const stored = 104 + 4 + 30 + 3 + 4 + 32 + 20
	for _, tc := range []struct {
		t              game.EntityType
		capacity, used int
	}{
		{game.COPN, 2945, stored},
		{game.CENC, 589, stored + 40},
		{game.CORB, 294, stored + 40},
		{game.CSHP, 294, stored + 40},
	} {
		c := colony(tc.t)
		if got := c.Capacity(); got != tc.capacity {
			t.Errorf("%s: capacity %d, want %d", tc.t, got, tc.capacity)
		}
		if got := c.SpaceUsed(); got != tc.used {
			t.Errorf("%s: used %d, want %d", tc.t, got, tc.used)
		}
		if got := c.SpaceFree(); got != tc.capacity-tc.used {
			t.Errorf("%s: free %d, want %d", tc.t, got, tc.capacity-tc.used)
		}
	}
}

func TestRoomAndStoreCargo(t *testing.T) {
	// An enclosed colony with 10 VU of space, and 1 FUEL (0.5 VU, which takes
	// a whole VU) in cargo.
	colony := func() game.Colony {
		return game.Colony{
			Type:        game.CENC,
			Operational: []game.Unit{{Type: "STRC", TechLevel: 1, Qty: 50}},
			Cargo:       []game.Unit{{Type: "FUEL", Qty: 1}},
		}
	}
	for _, tc := range []struct {
		code      string
		tl, qty   int
		room, use int // use: the space used afterwards
	}{
		{"FOOD", 0, 2, 2, 7},
		{"FOOD", 0, 3, 3, 10},
		{"FOOD", 0, 4, 3, 10},
		{"FOOD", 0, 1000, 3, 10},
		{"FUEL", 0, 1, 1, 1}, // the second FUEL shares the first one's VU
		{"FUEL", 0, 100, 19, 10},
		{"MTL", 0, 100, 18, 10},
		{"CNGD", 0, 100, 30, 10},
		{"FACT", 1, 5, 1, 8}, // 7 VU each stowed
		{"FOOD", 0, 0, 0, 1},
		{"FOOD", 0, -5, 0, 1},
		{"ORE", 0, 5, 0, 1},
	} {
		c := colony()
		if got := c.Room(tc.code, tc.tl, tc.qty); got != tc.room {
			t.Errorf("Room(%s-%d, %d) = %d, want %d", tc.code, tc.tl, tc.qty, got, tc.room)
			continue
		}
		if tc.tl != 0 {
			continue
		}
		if got := c.StoreCargo(tc.code, tc.qty); got != tc.room || c.SpaceUsed() != tc.use || c.SpaceFree() < 0 {
			t.Errorf("StoreCargo(%s, %d) = %d, leaving %d used and %d free; want %d and %d used", tc.code, tc.qty, got, c.SpaceUsed(), c.SpaceFree(), tc.room, tc.use)
		}
	}
	t.Run("over capacity", func(t *testing.T) {
		c := colony()
		c.Cargo = append(c.Cargo, game.Unit{Type: "FOOD", Qty: 100})
		if got := c.StoreCargo("FUEL", 1); got != 0 || c.SpaceFree() != -291 {
			t.Errorf("stored %d with %d free; want nothing stored and -291 free", got, c.SpaceFree())
		}
	})
	t.Run("an open-air colony may hold more than it encloses", func(t *testing.T) {
		c := colony()
		c.Type = game.COPN // 50 STRC-1 enclose 50 VU
		if got := c.StoreCargo("FOOD", 1000); got != 1000 || c.SpaceFree() != -2951 {
			t.Errorf("stored %d with %d free; want 1000 stored and -2951 free", got, c.SpaceFree())
		}
	})
	t.Run("resources", func(t *testing.T) {
		open, enclosed := colony(), colony()
		open.Type = game.COPN
		enclosed.Operational[0].Qty = 10 // 10 STRC-1 enclose 2 VU, 1 of them free
		if got := open.StoreResource("GOLD", 1000); got != 1000 || len(open.Depot) != 1 || open.Depot[0].Qty != 1000 || open.SpaceUsed() != 1 {
			t.Errorf("open-air: stored %d, depot %+v", got, open.Depot)
		}
		if got := enclosed.StoreResource("GOLD", 1000); got != 2 || len(enclosed.Depot) != 0 || enclosed.SpaceFree() != 0 {
			t.Errorf("enclosed: stored %d, depot %+v, %d free; want 2 stored in cargo", got, enclosed.Depot, enclosed.SpaceFree())
		}
	})
}

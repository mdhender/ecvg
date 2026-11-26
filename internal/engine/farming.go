// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Farming is the farming production phase of the turn numbered turn. In every
// colony in d, the FARM units in its farming groups that were allocated staff
// and fuel make FOOD, which goes into the colony's cargo; the fuel is burned
// from the cargo; and the colony's Harvest records what happened. Allocs is
// what AllocateResources returned for the turn. Nothing is farmed in turn
// SetupTurn.
//
// A working FARM-2 to FARM-10 unit makes game.FarmFood that turn. A FARM-1
// unit delivers only after game.FarmCropTurns consecutive turns of work, and
// loses the crop it had begun in a turn it doesn't work (game.AdvanceCrop).
func Farming(d *game.Data, turn int, allocs []Allocation) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Harvest = game.Harvest{}
		if turn <= SetupTurn || i >= len(allocs) {
			continue
		}
		c.Harvest = harvest(c, allocs[i])
	}
}

// harvest works the farming groups of c with what a allocated to them,
// burning its FUEL and adding to its FOOD.
func harvest(c *game.Colony, a Allocation) game.Harvest {
	var h game.Harvest
	for j, g := range c.FarmingGroups {
		for k := range g.Units {
			u := &g.Units[k]
			var w Work
			if j < len(a.Farms) && k < len(a.Farms[j]) {
				w = a.Farms[j][k]
			}
			h.Units += u.Qty
			h.Ordered += w.Ordered
			h.Unstaffed += u.Qty - w.Ordered - w.Staffed
			h.Unfuelled += w.Staffed - w.Worked
			h.Worked += w.Worked
			h.Fuel += w.Fuel
			if u.TechLevel == 1 {
				var delivered, ruined int
				u.Pipeline, delivered, ruined = game.AdvanceCrop(u.Qty, u.Pipeline, w.Worked)
				h.Ruined += ruined
				h.Food += game.FarmFood(u.TechLevel, delivered)
			} else {
				h.Food += game.FarmFood(u.TechLevel, w.Worked)
			}
		}
	}
	left := h.Fuel
	for j := range c.Cargo {
		if u := &c.Cargo[j]; u.Type == units.FUEL {
			n := min(left, u.Qty)
			u.Qty -= n
			left -= n
		}
	}
	if h.Food > 0 {
		stored := false
		for j := range c.Cargo {
			if u := &c.Cargo[j]; u.Type == units.FOOD {
				u.Qty += h.Food
				stored = true
				break
			}
		}
		if !stored {
			c.Cargo = append(c.Cargo, game.Unit{Type: units.FOOD, Qty: h.Food})
		}
	}
	return h
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Farming is the farming production phase of the turn numbered turn. In every
// colony in d, the FARM units in its farming groups that were allocated staff
// and fuel make FOOD, which goes into the colony's cargo, as much as there is room
// for; the fuel is burned
// from its cargo and then its depot; and the colony's Harvest records what happened. Allocs is
// what AllocateResources returned for the turn. Nothing is farmed in turn
// SetupTurn.
//
// A working FARM unit makes game.FarmFood that turn.
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
			u := g.Units[k]
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
			h.Food += game.FarmFood(u.TechLevel, w.Worked)
		}
	}
	c.Spend(units.FUEL, h.Fuel)
	// The colony can't accept more FOOD than it has room for.
	stored := c.StoreCargo(units.FOOD, h.Food)
	h.Lost = h.Food - stored
	h.Food = stored
	return h
}

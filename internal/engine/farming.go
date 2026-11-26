// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Farming is the farming production phase of the turn numbered turn. In every
// colony in d, the FARM units in its farming groups make FOOD, which goes
// into the colony's cargo, and the colony's Harvest records what they did.
// Nothing is farmed in turn SetupTurn.
//
// The units are taken in group order, and within a group in the order they
// are listed. A unit works only if it has staff and fuel:
//
//   - Staff is game.FarmPRO PRO and game.FarmUSK USK, taken from the colony's
//     worker cadre (see AllocateResources, which must run first). A unit left
//     without fuel gives its staff back.
//   - Fuel is game.FarmFuel, burned from the FUEL in the colony's cargo,
//     unless the unit is solar powered (game.FarmSolar).
//
// A working FARM-2 to FARM-10 unit makes game.FarmFood that turn. A FARM-1
// unit delivers only after game.FarmCropTurns consecutive turns of work, and
// loses the crop it had begun in a turn it doesn't work (game.AdvanceCrop).
func Farming(d *game.Data, turn int) {
	orbitNo := map[int]int{} // by planet ID
	orbits := map[int]int{}
	for _, o := range d.Orbits {
		orbits[o.ID] = o.OrbitNo
	}
	for _, p := range d.Planets {
		orbitNo[p.ID] = orbits[p.OrbitID]
	}
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Harvest = game.Harvest{}
		if turn <= SetupTurn {
			continue
		}
		c.Harvest = harvest(c, orbitNo[c.PlanetID])
	}
}

// harvest works the farming groups of c, which is in orbit orbitNo, burning
// its FUEL and adding to its FOOD.
func harvest(c *game.Colony, orbitNo int) game.Harvest {
	var h game.Harvest
	fuel := 0
	for _, u := range c.Cargo {
		if u.Type == units.FUEL {
			fuel += u.Qty
		}
	}
	for _, g := range c.FarmingGroups {
		for k := range g.Units {
			u := &g.Units[k]
			h.Units += u.Qty
			staffed := c.Workers.Take(u.Qty, game.FarmPRO, game.FarmUSK)
			h.Unstaffed += u.Qty - staffed
			worked, burned := staffed, 0
			if !game.FarmSolar(u.TechLevel, c.Type, orbitNo) {
				worked = min(staffed, game.FarmsFuelled(u.TechLevel, fuel))
				burned = game.FarmFuel(u.TechLevel, worked)
			}
			h.Unfuelled += staffed - worked
			c.Workers.Return(staffed-worked, game.FarmPRO, game.FarmUSK)
			fuel -= burned
			h.Worked += worked
			h.Fuel += burned
			if u.TechLevel == 1 {
				var delivered, ruined int
				u.Pipeline, delivered, ruined = game.AdvanceCrop(u.Qty, u.Pipeline, worked)
				h.Ruined += ruined
				h.Food += game.FarmFood(u.TechLevel, delivered)
			} else {
				h.Food += game.FarmFood(u.TechLevel, worked)
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

// AllocateResources is the resource allocation phase of the turn numbered
// turn. It makes each colony's worker cadre for the turn (game.NewWorkers),
// which the production phases then drain; food has first call on it, so
// Farming runs before any other production. No cadre is made in turn
// SetupTurn.
func AllocateResources(d *game.Data, turn int) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Workers = game.Workers{}
		if turn > SetupTurn {
			c.Workers = game.NewWorkers(*c)
		}
	}
}

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
//   - Staff is game.FarmPRO PRO and game.FarmUSK USK, from the colony's
//     population. Nobody staffs two units, and a unit left without fuel gives
//     its staff back.
//   - Fuel is game.FarmFuel, burned from the FUEL in the colony's cargo,
//     unless the unit is solar powered (game.FarmSolar).
//
// A working unit makes game.FarmFood.
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
	pro, usk := c.Population.PRO, c.Population.USK
	fuel := 0
	for _, u := range c.Cargo {
		if u.Type == units.FUEL {
			fuel += u.Qty
		}
	}
	for _, g := range c.FarmingGroups {
		for _, u := range g.Units {
			h.Units += u.Qty
			staffed := max(0, min(u.Qty, pro/game.FarmPRO, usk/game.FarmUSK))
			h.Unstaffed += u.Qty - staffed
			worked, burned := staffed, 0
			if !game.FarmSolar(u.TechLevel, c.Type, orbitNo) {
				worked = min(staffed, game.FarmsFuelled(u.TechLevel, fuel))
				burned = game.FarmFuel(u.TechLevel, worked)
			}
			h.Unfuelled += staffed - worked
			pro -= worked * game.FarmPRO
			usk -= worked * game.FarmUSK
			fuel -= burned
			h.Worked += worked
			h.Fuel += burned
			h.Food += game.FarmFood(u.TechLevel, worked)
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

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Work is what one entry of a group's units was allocated for a turn.
type Work struct {
	// Staffed is the number of the units that got staff, and Worked is the
	// number of those that also got the fuel they need, so that they work.
	Staffed int
	Worked  int
	// Fuel is the FUEL allocated to the units that work.
	Fuel int
}

// Allocation is what a colony's groups were allocated for a turn.
type Allocation struct {
	// Farms is indexed like the colony's FarmingGroups and then like the
	// group's Units.
	Farms [][]Work
}

// AllocateResources is the resource allocation phase of the turn numbered
// turn. For each colony in d it makes the worker cadre for the turn
// (game.NewWorkers) and shares the cadre and the FUEL in the colony's cargo
// among its groups. It returns the allocations, one per colony in d.Colonies
// order. Nothing is allocated in turn SetupTurn, and it returns nil.
//
// The scheme is deliberately blunt, not efficient. It makes two passes over
// the groups, each time the farming groups first (mining and factory groups
// will follow them), and each kind in order of seniority: the group number,
// lowest first. Within a group the units are taken in the order listed.
//
//  1. Staff. Each group takes as much of the cadre as it can use
//     (game.FarmPRO and game.FarmUSK for each FARM unit), until the cadre
//     can't staff another unit. Staff is never given back, even by a unit
//     that then gets no fuel.
//  2. Fuel. Each group takes as much FUEL as its staffed units can use
//     (game.FarmFuel; none for solar powered farms, game.FarmSolar). The pass
//     stops when the fuel runs out, or at the first group that has no staff.
//
// The fuel is only allocated here; the production phases burn it.
func AllocateResources(d *game.Data, turn int) []Allocation {
	if turn <= SetupTurn {
		for i := range d.Colonies {
			d.Colonies[i].Workers = game.Workers{}
		}
		return nil
	}
	orbits := map[int]int{} // orbit number by orbit ID
	for _, o := range d.Orbits {
		orbits[o.ID] = o.OrbitNo
	}
	orbitNo := map[int]int{} // by planet ID
	for _, p := range d.Planets {
		orbitNo[p.ID] = orbits[p.OrbitID]
	}
	allocs := make([]Allocation, len(d.Colonies))
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Workers = game.NewWorkers(*c)
		allocs[i] = allocate(c, orbitNo[c.PlanetID])
	}
	return allocs
}

// allocate shares the worker cadre and FUEL of c, which is in orbit orbitNo,
// among its groups.
func allocate(c *game.Colony, orbitNo int) Allocation {
	a := Allocation{Farms: make([][]Work, len(c.FarmingGroups))}

	// Pass 1: staff.
	for j, g := range c.FarmingGroups {
		a.Farms[j] = make([]Work, len(g.Units))
		for k, u := range g.Units {
			a.Farms[j][k].Staffed = c.Workers.Take(u.Qty, game.FarmPRO, game.FarmUSK)
		}
	}

	// Pass 2: fuel.
	fuel := 0
	for _, u := range c.Cargo {
		if u.Type == units.FUEL {
			fuel += u.Qty
		}
	}
	for j, g := range c.FarmingGroups {
		staffed := 0
		for k, u := range g.Units {
			w := &a.Farms[j][k]
			staffed += w.Staffed
			if game.FarmSolar(u.TechLevel, c.Type, orbitNo) {
				w.Worked = w.Staffed
				continue
			}
			w.Worked = min(w.Staffed, game.FarmsFuelled(u.TechLevel, fuel))
			w.Fuel = game.FarmFuel(u.TechLevel, w.Worked)
			fuel -= w.Fuel
		}
		if staffed == 0 {
			break
		}
	}
	return a
}

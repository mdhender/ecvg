// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"cmp"
	"slices"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Work is what one entry of a group's units was allocated for a turn.
type Work struct {
	// Ordered is the number of the units the empire ordered to idle. They
	// ask for nothing.
	Ordered int
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
	// Mines is indexed like the colony's MiningGroups and then like the
	// group's Units.
	Mines [][]Work
}

// AllocateResources is the resource allocation phase of the turn numbered
// turn. For each colony in d it makes the worker cadre for the turn
// (game.NewWorkers) and shares the cadre and the colony's FUEL (in its cargo
// and depot) among its groups. It returns the allocations, one per colony in d.Colonies
// order. Nothing is allocated in turn SetupTurn, and it returns nil.
//
// There are two levels of idle. The first is the empire's: the units a
// group's IdlePercent idles (game.IdleUnits) ask for no staff or fuel. The
// second is set here: units that asked and got no staff, or staff but no
// fuel. Both are recorded in the Idle of the group's units, for the
// production phases to respect and the report to show.
//
// The scheme is deliberately blunt, not efficient. It makes two passes over
// the groups, each time the farming groups first, then the mining groups
// (factory groups will follow them), and each kind in order of seniority: the group number,
// lowest first. Within a group the units are taken in the order listed.
//
//  1. Staff. Each group takes as much of the cadre as it can use
//     (game.FarmPRO and game.FarmUSK for each FARM unit, game.MinePRO and
//     game.MineUSK for each MINE unit), until the cadre
//     can't staff another unit. Staff is never given back, even by a unit
//     that then gets no fuel.
//  2. Fuel. Each group takes as much FUEL as its staffed units can use
//     (game.FarmFuel, but none for solar powered farms, game.FarmSolar;
//     game.MineFuel). The pass
//     stops when the fuel runs out, or at the first group that has no staff.
//
// The fuel is only allocated here; the production phases burn it.
func AllocateResources(d *game.Data, turn int) []Allocation {
	if turn <= SetupTurn {
		for i := range d.Colonies {
			c := &d.Colonies[i]
			c.Workers = game.Workers{}
			for _, g := range c.FarmingGroups {
				for k := range g.Units {
					g.Units[k].Idle = game.Idle{}
				}
			}
			for _, g := range c.MiningGroups {
				for k := range g.Units {
					g.Units[k].Idle = game.Idle{}
				}
			}
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

// slot is one group in the order the passes take them.
type slot struct {
	mine        bool
	idlePercent int
	units       []game.GroupUnits
	work        []Work
}

// allocate shares the worker cadre and FUEL of c, which is in orbit orbitNo,
// among its groups.
func allocate(c *game.Colony, orbitNo int) Allocation {
	a := Allocation{
		Farms: make([][]Work, len(c.FarmingGroups)),
		Mines: make([][]Work, len(c.MiningGroups)),
	}
	// The groups in the order of both passes: farms, then mines, each kind
	// by seniority.
	var slots []slot
	for _, j := range bySeniority(len(c.FarmingGroups), func(j int) int { return c.FarmingGroups[j].No }) {
		g := c.FarmingGroups[j]
		a.Farms[j] = make([]Work, len(g.Units))
		slots = append(slots, slot{idlePercent: g.IdlePercent, units: g.Units, work: a.Farms[j]})
	}
	for _, j := range bySeniority(len(c.MiningGroups), func(j int) int { return c.MiningGroups[j].No }) {
		g := c.MiningGroups[j]
		a.Mines[j] = make([]Work, len(g.Units))
		slots = append(slots, slot{mine: true, idlePercent: g.IdlePercent, units: g.Units, work: a.Mines[j]})
	}

	// Pass 1: staff.
	for _, s := range slots {
		pro, usk := game.FarmPRO, game.FarmUSK
		if s.mine {
			pro, usk = game.MinePRO, game.MineUSK
		}
		for k, u := range s.units {
			w := &s.work[k]
			w.Ordered = game.IdleUnits(u.Qty, s.idlePercent)
			w.Staffed = c.Workers.Take(u.Qty-w.Ordered, pro, usk)
		}
	}

	// Pass 2: fuel.
	fuel := c.Stock(units.FUEL)
	for _, s := range slots {
		staffed, wanted := 0, false
		for k, u := range s.units {
			w := &s.work[k]
			staffed += w.Staffed
			wanted = wanted || u.Qty > w.Ordered
			switch {
			case s.mine:
				w.Worked = min(w.Staffed, game.MinesFuelled(u.TechLevel, fuel))
				w.Fuel = game.MineFuel(u.TechLevel, w.Worked)
			case game.FarmSolar(u.TechLevel, c.Type, orbitNo):
				w.Worked = w.Staffed
			default:
				w.Worked = min(w.Staffed, game.FarmsFuelled(u.TechLevel, fuel))
				w.Fuel = game.FarmFuel(u.TechLevel, w.Worked)
			}
			fuel -= w.Fuel
		}
		// A group that asked for staff and got none ends the pass. One that
		// is idle by order asked for nothing, and is passed over.
		if staffed == 0 && wanted {
			break
		}
	}

	// Record who is idle, and why.
	for _, s := range slots {
		for k := range s.units {
			w := s.work[k]
			s.units[k].Idle = game.Idle{
				Ordered:   w.Ordered,
				Unstaffed: s.units[k].Qty - w.Ordered - w.Staffed,
				Unfuelled: w.Staffed - w.Worked,
			}
		}
	}
	return a
}

// bySeniority returns the indexes 0..n-1 ordered by the group number that no
// returns for each, lowest first.
func bySeniority(n int, no func(int) int) []int {
	idx := make([]int, n)
	for i := range idx {
		idx[i] = i
	}
	slices.SortStableFunc(idx, func(a, b int) int { return cmp.Compare(no(a), no(b)) })
	return idx
}

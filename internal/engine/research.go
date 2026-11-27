// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Research is the research production phase of the turn numbered turn. In
// every colony in d, the LAB units in its laboratory groups that were
// allocated staff and fuel burn the fuel, from its cargo and then its depot,
// and make research points (RSCH), which go into the colony's cargo; the
// colony's Research records what happened. Allocs is what AllocateResources
// returned for the turn. No research is done in turn SetupTurn.
//
// A working LAB unit does 0.25 x TL of a research point that turn. The
// fractions of all the colony's laboratories are added up and the total is
// rounded down (game.LabQuarters, game.LabResearch); what is rounded off is
// lost, not carried to the next turn.
func Research(d *game.Data, turn int, allocs []Allocation) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Research = game.Research{}
		if turn <= SetupTurn || i >= len(allocs) {
			continue
		}
		c.Research = research(c, allocs[i])
	}
}

// research works the laboratory groups of c with what a allocated to them,
// burning its FUEL and adding to its RSCH.
func research(c *game.Colony, a Allocation) game.Research {
	var r game.Research
	quarters := 0
	for j, g := range c.LaboratoryGroups {
		for k, u := range g.Units {
			var w Work
			if j < len(a.Labs) && k < len(a.Labs[j]) {
				w = a.Labs[j][k]
			}
			r.Units += u.Qty
			r.Ordered += w.Ordered
			r.Unstaffed += u.Qty - w.Ordered - w.Staffed
			r.Unfuelled += w.Staffed - w.Worked
			r.Worked += w.Worked
			r.Fuel += w.Fuel
			quarters += game.LabQuarters(u.TechLevel, w.Worked)
		}
	}
	c.Spend(units.FUEL, r.Fuel)
	r.RSCH = game.LabResearch(quarters)
	c.AddResearch(r.RSCH)
	return r
}

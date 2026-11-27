// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/units"
)

// ErrLabSite is matched by the errors of CanAssembleLabs.
const ErrLabSite = cerrs.Error("laboratories can't be assembled here")

// HasLabs reports whether the colony has any assembled laboratories, running
// or not. It takes them to put research into prototypes or to take out of a
// prototype more research than it uses.
func (c *Colony) HasLabs() bool {
	for _, g := range c.LaboratoryGroups {
		if GroupQty(g.Units) > 0 {
			return true
		}
	}
	return false
}

// CanAssembleLabs reports whether an entity of type t may assemble a
// laboratory group: any colony can, and a ship can't, though it may carry LAB
// units in storage. The error, if any, matches ErrLabSite.
func CanAssembleLabs(t EntityType) error {
	if t == CSHP {
		return fmt.Errorf("%w: a laboratory group needs a colony, not a %s", ErrLabSite, t)
	}
	return nil
}

// What a LAB unit needs to work for a turn, whatever its tech level.
const (
	// LabPRO and LabUSK are the population units that staff one LAB unit.
	LabPRO = 3
	LabUSK = 1
	// LabsPerRSCH is the number of working LAB-1 units it takes to make one
	// research point in a turn: a LAB makes 0.25 x TL.
	LabsPerRSCH = 4
)

// LabFuel returns the FUEL that qty LAB units of tech level tl burn in a
// turn: 0.5 x TL each, the total rounded up.
func LabFuel(tl, qty int) int {
	return (tl*qty + 1) / 2
}

// LabsFuelled returns the most LAB units of tech level tl that fuel FUEL can
// run for a turn.
func LabsFuelled(tl, fuel int) int {
	if tl < 1 {
		return 0
	}
	return 2 * fuel / tl
}

// LabQuarters returns the research that qty working LAB units of tech level
// tl do in a turn, in quarters of a research point: TL each. Add up the
// quarters of all of an entity's laboratories before turning them into
// points (LabResearch), so that only the total is rounded.
func LabQuarters(tl, qty int) int {
	return tl * qty
}

// LabResearch returns the research points that quarters quarters of a point
// make: one for every LabsPerRSCH, rounded down.
func LabResearch(quarters int) int {
	return max(0, quarters) / LabsPerRSCH
}

// LaboratoryGroup is a group of laboratories. Every unit of it is of the same
// tech level, so its Units has one entry, and an entity has at most one
// laboratory group of each tech level.
type LaboratoryGroup struct {
	No int `json:"no"`
	// ShutDown is as for Group.
	ShutDown int          `json:"shut_down,omitzero"`
	Units    []GroupUnits `json:"units"`
}

// Research is what a colony's laboratory groups did in a turn.
type Research struct {
	// Units is the number of LAB units in the laboratory groups, and Worked
	// is how many of them did research.
	Units  int `json:"units"`
	Worked int `json:"worked"`
	// Ordered units were shut down by the empire's order. Unstaffed units had
	// no PRO and USK to work them, and Unfuelled units had the staff but no
	// FUEL.
	Ordered   int `json:"ordered,omitzero"`
	Unstaffed int `json:"unstaffed,omitzero"`
	Unfuelled int `json:"unfuelled,omitzero"`
	// Fuel is the FUEL the units burned, and RSCH is the research points
	// they made, which went into the colony's cargo.
	Fuel int `json:"fuel"`
	RSCH int `json:"rsch"`
}

// AddResearch puts qty research points into the colony's cargo. A research
// point has no mass or volume, so there is always room for it.
func (c *Colony) AddResearch(qty int) {
	c.Cargo = store(c.Cargo, units.RSCH, qty)
}

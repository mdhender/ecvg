// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// EMUPerCWRK is the work one CWRK does in a turn, in EMU (equivalent mass
// units): it assembles up to that many MU.
const EMUPerCWRK = 500

// Assembly is the work of assembling units into one of a colony's groups. An
// assemble order starts it, and it lasts until every unit is assembled.
type Assembly struct {
	Kind    GroupKind `json:"kind"`
	GroupNo int       `json:"group_no"`
	// Unit is the units still to assemble. They stay in storage until then.
	Unit Unit `json:"unit"`
	// EMU is the work done toward them. A unit is assembled when the work
	// reaches its mass, so a partly assembled unit keeps its EMU between
	// turns.
	EMU int `json:"emu"`
	// Assembled is the number of units assembled in the prior turn, and
	// Waiting says why units that had the work done weren't: "room" or
	// "units".
	Assembled int    `json:"assembled,omitzero"`
	Waiting   string `json:"waiting,omitzero"`
}

// Construction is what a colony's CWRK did in a turn.
type Construction struct {
	// Pool is the EMU the CWRK could do, EMUPerCWRK each, and Used the EMU
	// that went into assembly. The rest is lost.
	Pool int `json:"pool"`
	Used int `json:"used"`
}

// Stored returns the quantity of the unit with the given code and tech level
// in the colony's storage: its cargo and its depot together.
func (c *Colony) Stored(code string, tl int) int {
	n := 0
	for _, list := range [][]Unit{c.Cargo, c.Depot} {
		for _, u := range list {
			if u.Type == code && u.TechLevel == tl {
				n += u.Qty
			}
		}
	}
	return n
}

// Claimed returns the quantity of the unit with the given code and tech level
// that the colony's assembly work is still to assemble.
func (c *Colony) Claimed(code string, tl int) int {
	n := 0
	for _, a := range c.Assembly {
		if a.Unit.Type == code && a.Unit.TechLevel == tl {
			n += a.Unit.Qty
		}
	}
	return n
}

// Assemble does a turn of the colony's assembly work. The colony's CWRK make
// a pool of EMUPerCWRK EMU each, and each piece of work, oldest first, takes
// as much of it as it can use. The units whose mass the work has reached
// move from storage into use, in the depot if the colony has one and
// otherwise in the colony itself, if it has room for them assembled, and
// join their group. EMU nothing took is lost. Work that finished in the prior
// turn is dropped first.
func (c *Colony) Assemble() {
	jobs := c.Assembly[:0]
	for _, a := range c.Assembly {
		if a.Unit.Qty > 0 {
			a.Assembled, a.Waiting = 0, ""
			jobs = append(jobs, a)
		}
	}
	c.Assembly = jobs
	pool := max(0, c.Cadres.CWRK) * EMUPerCWRK
	left := pool
	for i := range c.Assembly {
		a := &c.Assembly[i]
		d, ok := units.Lookup(a.Unit.Type)
		if !ok {
			continue
		}
		tl := a.Unit.TechLevel
		take := max(0, min(left, d.Mass(tl, a.Unit.Qty)-a.EMU))
		a.EMU += take
		left -= take
		// The most units whose mass the work has reached.
		n := sort.Search(a.Unit.Qty, func(k int) bool { return d.Mass(tl, k+1) > a.EMU })
		if have := c.Stored(a.Unit.Type, tl); have < n {
			n, a.Waiting = have, "units"
		}
		if !c.HasDepot() {
			// Assembled in the colony, they take up their assembled volume
			// in place of their stowed volume.
			free := max(0, c.SpaceFree())
			fit := sort.Search(n, func(k int) bool { return d.Volume(tl, k+1)-d.StowedVolume(tl, k+1) > free })
			if fit < n {
				n, a.Waiting = fit, "room"
			}
		}
		if n <= 0 {
			continue
		}
		c.takeStored(a.Unit.Type, tl, n)
		if c.HasDepot() {
			c.DepotOperational = addUnits(c.DepotOperational, a.Unit.Type, tl, n)
		} else {
			c.Operational = addUnits(c.Operational, a.Unit.Type, tl, n)
		}
		c.addToGroup(a.Kind, a.GroupNo, tl, n)
		a.EMU -= d.Mass(tl, n)
		a.Unit.Qty -= n
		a.Assembled = n
	}
	c.Construction = Construction{Pool: pool, Used: pool - left}
}

// takeStored removes qty of a unit from the colony's storage, cargo first.
func (c *Colony) takeStored(code string, tl, qty int) {
	for _, list := range [][]Unit{c.Cargo, c.Depot} {
		for i := range list {
			if u := &list[i]; u.Type == code && u.TechLevel == tl {
				n := min(qty, u.Qty)
				u.Qty -= n
				qty -= n
			}
		}
	}
}

// addUnits adds qty of a unit to list.
func addUnits(list []Unit, code string, tl, qty int) []Unit {
	for i := range list {
		if u := &list[i]; u.Type == code && u.TechLevel == tl {
			u.Qty += qty
			return list
		}
	}
	return append(list, Unit{Type: code, TechLevel: tl, Qty: qty})
}

// GroupUnitsAt returns the entry of us for the tech level tl, adding an empty
// one if there is none, and the list.
func GroupUnitsAt(us []GroupUnits, tl int) (*GroupUnits, []GroupUnits) {
	for i := range us {
		if us[i].TechLevel == tl {
			return &us[i], us
		}
	}
	us = append(us, GroupUnits{TechLevel: tl})
	return &us[len(us)-1], us
}

// addToGroup adds qty units of tech level tl to the colony's group of the
// given kind numbered no.
func (c *Colony) addToGroup(kind GroupKind, no, tl, qty int) {
	add := func(us *[]GroupUnits) {
		var e *GroupUnits
		e, *us = GroupUnitsAt(*us, tl)
		e.Qty += qty
	}
	switch kind {
	case KindFarming:
		for i := range c.FarmingGroups {
			if c.FarmingGroups[i].No == no {
				add(&c.FarmingGroups[i].Units)
			}
		}
	case KindFactory:
		for i := range c.FactoryGroups {
			if c.FactoryGroups[i].No == no {
				add(&c.FactoryGroups[i].Units)
			}
		}
	case KindMining:
		for i := range c.MiningGroups {
			if c.MiningGroups[i].No == no {
				add(&c.MiningGroups[i].Units)
			}
		}
	}
}

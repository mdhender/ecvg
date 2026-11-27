// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// StructureRatio returns the number of STRN equivalents it takes to enclose
// one VU in an entity of type t (manual F3.1.1).
func (t EntityType) StructureRatio() int {
	switch t {
	case COPN:
		return 1
	case CENC:
		return 5
	case CORB:
		return 8
	}
	return 10 // ships
}

// StructureEquivalents returns the colony's assembled, operational structure
// in STRN equivalents: 1 for each STRN, 5 for each STRL, and 20 for each
// STRS. Structure in cargo or in the depot counts for nothing.
func (c *Colony) StructureEquivalents() int {
	n := 0
	for _, u := range c.Operational {
		if d, ok := units.Lookup(u.Type); ok {
			n += u.Qty * d.Encloses()
		}
	}
	return n
}

// Capacity returns the enclosure the colony has, in VU: its structure's STRN
// equivalents divided by its structure ratio. It is exact.
func (c *Colony) Capacity() units.Frac {
	return units.NewFrac(c.StructureEquivalents(), c.Type.StructureRatio())
}

// SpaceUsed returns the enclosure the colony requires, in VU, exactly.
// Everything in the entity counts: its population, which wants to be
// enclosed even in an open-air colony; its operational units, at their
// assembled volume, which is nothing for structure; and its cargo, at its
// stowed volume. What is in the depot is outside the entity and takes none.
func (c *Colony) SpaceUsed() units.Frac {
	used := units.Whole(c.Population.Volume())
	for _, u := range c.Operational {
		if d, ok := units.Lookup(u.Type); ok {
			used = used.Add(d.Volume(u.TechLevel, u.Qty))
		}
	}
	for _, u := range c.Cargo {
		if d, ok := units.Lookup(u.Type); ok {
			used = used.Add(d.StowedVolume(u.TechLevel, u.Qty))
		}
	}
	return used
}

// SpaceFree returns the enclosure left in the colony, in VU. It is negative
// if the colony requires more than it has.
func (c *Colony) SpaceFree() units.Frac {
	return c.Capacity().Sub(c.SpaceUsed())
}

// Room returns the most of qty units with the given code and tech level that
// the colony's cargo has space for. Capacity is checked every time something
// is stored: an entity accepts only the whole units that fit, and an exact
// fit is accepted (manual F3.2.1), unless it is an open-air surface colony,
// which may hold more than it encloses (manual F3.2).
func (c *Colony) Room(code string, tl, qty int) int {
	d, ok := units.Lookup(code)
	if !ok || qty <= 0 {
		return 0
	}
	if !c.Type.Enclosed() {
		return qty
	}
	free := c.SpaceFree()
	if free.Sign() < 0 {
		return 0
	}
	// The most n for which the n units fit: the first that doesn't, less 1.
	return sort.Search(qty, func(n int) bool {
		return d.StowedVolume(tl, n+1).Cmp(free) > 0
	})
}

// StoreCargo puts as many of qty units with the given code and no tech level
// into the colony's cargo as it has room for, and returns the number stored.
func (c *Colony) StoreCargo(code string, qty int) int {
	n := c.Room(code, 0, qty)
	c.Cargo = store(c.Cargo, code, n)
	return n
}

// StoreResource puts qty of a mined resource into the colony's inventory and
// returns the quantity stored: all of it into the depot, which is unlimited,
// if the colony has one, and otherwise as much as the cargo has room for. An
// enclosed surface colony keeps its MTL and NMTL in an external depot, where
// they never run short of room (manual H2.2): they go into its depot list,
// which holds nothing else.
func (c *Colony) StoreResource(code string, qty int) int {
	if c.HasDepot() || (c.Type == CENC && (code == units.MTL || code == units.NMTL)) {
		c.Depot = store(c.Depot, code, qty)
		return max(0, qty)
	}
	return c.StoreCargo(code, qty)
}

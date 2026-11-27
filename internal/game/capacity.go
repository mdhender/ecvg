// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// StructuralRequirement returns the number that divides the space a
// structural unit creates in an entity of type t: a unit of tech level TL
// creates TL^2 / StructuralRequirement VU.
func (t EntityType) StructuralRequirement() int {
	switch t {
	case COPN:
		return 1
	case CENC:
		return 5
	}
	return 10 // ships and orbital colonies
}

// Capacity returns the space the colony has, in VU: the sum of the space
// created by its assembled, operational structural units (STRC and STRL),
// rounded down. Structure in cargo or in the depot creates none.
func (c *Colony) Capacity() int {
	space := 0
	for _, u := range c.Operational {
		if d, ok := units.Lookup(u.Type); ok && d.Class == units.Structural {
			space += u.Qty * u.TechLevel * u.TechLevel
		}
	}
	return space / c.Type.StructuralRequirement()
}

// SpaceUsed returns the space taken up in the colony, in VU. Everything in
// the entity counts: its population, if it is enclosed; its operational
// units, at their assembled volume, except the structural units; and its
// cargo, at its stowed volume. Each quantity is rounded up. What is in the
// depot is outside the entity and takes none.
func (c *Colony) SpaceUsed() int {
	used := c.Population.Volume(c.Type)
	for _, u := range c.Operational {
		if d, ok := units.Lookup(u.Type); ok && d.Class != units.Structural {
			used += d.Volume(u.TechLevel, u.Qty)
		}
	}
	for _, u := range c.Cargo {
		if d, ok := units.Lookup(u.Type); ok {
			used += d.StowedVolume(u.TechLevel, u.Qty)
		}
	}
	return used
}

// SpaceFree returns the space left in the colony, in VU. It is negative if
// the colony holds more than it has capacity for.
func (c *Colony) SpaceFree() int {
	return c.Capacity() - c.SpaceUsed()
}

// Room returns the most of qty units with the given code and tech level that
// the colony's cargo has space for. Capacity is checked every time something
// is stored: an entity can't accept the excess.
func (c *Colony) Room(code string, tl, qty int) int {
	d, ok := units.Lookup(code)
	if !ok || qty <= 0 {
		return 0
	}
	// The units join the first cargo entry of their kind, and the volume of
	// an entry is rounded up as a whole.
	have := 0
	for _, u := range c.Cargo {
		if u.Type == code && u.TechLevel == tl {
			have = u.Qty
			break
		}
	}
	free, base := c.SpaceFree(), d.StowedVolume(tl, have)
	if free < 0 {
		return 0
	}
	// The most n for which the n units fit: the first that doesn't, less 1.
	return sort.Search(qty, func(n int) bool {
		return d.StowedVolume(tl, have+n+1)-base > free
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
// enclosed surface colony keeps its MTL and NMTL outside its walls, where
// they never run short of room (manual H2.2): they go into its depot list,
// which holds nothing else.
func (c *Colony) StoreResource(code string, qty int) int {
	if c.HasDepot() || (c.Type == CENC && (code == units.MTL || code == units.NMTL)) {
		c.Depot = store(c.Depot, code, qty)
		return max(0, qty)
	}
	return c.StoreCargo(code, qty)
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// ErrMineSite is matched by the errors of CanAssembleMines.
const ErrMineSite = cerrs.Error("mines can't be assembled here")

// CanAssembleMines reports whether an entity of type t may assemble a mining
// group: only surface colonies, open-air or enclosed, can. Orbital colonies
// and ships can't. The error, if any, matches ErrMineSite.
func CanAssembleMines(t EntityType) error {
	if t == COPN || t == CENC {
		return nil
	}
	return fmt.Errorf("%w: a mining group needs a surface colony, not a %s", ErrMineSite, t)
}

// What a MINE unit needs to work for a turn, whatever its tech level.
const (
	// MinePRO and MineUSK are the population units that staff one MINE unit.
	MinePRO = 1
	MineUSK = 3
	// MineOrePerTL is the ore a working MINE unit extracts in a turn for each
	// of its tech levels, before the deposit's yield is applied.
	MineOrePerTL = 25
)

// MineFuel returns the FUEL that qty MINE units of tech level tl burn in a
// turn: 0.5 x TL each, the total rounded up.
func MineFuel(tl, qty int) int {
	return (tl*qty + 1) / 2
}

// MinesFuelled returns the most MINE units of tech level tl that fuel FUEL
// can run for a turn.
func MinesFuelled(tl, fuel int) int {
	if tl < 1 {
		return 0
	}
	return 2 * fuel / tl
}

// MineOutput returns the refined product that qty working MINE units of tech
// level tl extract in a turn from a deposit with the given yield percentage:
// MineOrePerTL x TL x yield% each, the total rounded down.
func MineOutput(tl, qty, yield int) int {
	return MineOrePerTL * tl * qty * yield / 100
}

// Mined is what a mining group did in a turn.
type Mined struct {
	// Worked is the number of the group's units that worked, and Fuel is the
	// FUEL they burned.
	Worked int `json:"worked"`
	Fuel   int `json:"fuel"`
	// Product is the refined product that went into the colony's inventory.
	Product int `json:"product"`
	// Returned is the product the colony had no room for. It went back into
	// the deposit: the fuel was spent for nothing.
	Returned int `json:"returned,omitzero"`
}

// Stock returns the quantity the colony holds of the unit with the given
// code that has no tech level (a resource, FOOD, ...), in its cargo and its
// depot together.
func (c *Colony) Stock(code string) int {
	n := 0
	for _, list := range [][]Unit{c.Cargo, c.Depot} {
		for _, u := range list {
			if u.Type == code && u.TechLevel == 0 {
				n += u.Qty
			}
		}
	}
	return n
}

// Spend takes up to qty of the unit with the given code and no tech level
// out of the colony's inventory, the cargo first and then the depot, and
// returns the quantity it took.
func (c *Colony) Spend(code string, qty int) int {
	left := max(0, qty)
	for _, list := range [][]Unit{c.Cargo, c.Depot} {
		for i := range list {
			if u := &list[i]; u.Type == code && u.TechLevel == 0 {
				n := min(left, u.Qty)
				u.Qty -= n
				left -= n
			}
		}
	}
	return max(0, qty) - left
}

// MaxFactoryGroups is the most factory groups an entity may have (manual
// H3.0).
const MaxFactoryGroups = 25

// HasDepot reports whether the colony has a depot: storage outside the
// entity, which only an open-air surface colony has. (An enclosed surface
// colony keeps MTL and NMTL in an external depot, but nothing else: see
// StoreResource.)
func (c *Colony) HasDepot() bool {
	return c.Type == COPN
}

func store(list []Unit, code string, qty int) []Unit {
	if qty <= 0 {
		return list
	}
	for i := range list {
		if u := &list[i]; u.Type == code && u.TechLevel == 0 {
			u.Qty += qty
			return list
		}
	}
	return append(list, Unit{Type: code, Qty: qty})
}

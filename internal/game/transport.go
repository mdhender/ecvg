// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"
	"sort"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/units"
)

const (
	// TransportMUPerFuel is the mass that a unit of FUEL moves. A TRNS
	// carries 200 x TL^2 MU and burns 0.1 x TL^2 FUEL a turn, so it is the
	// same at every tech level.
	TransportMUPerFuel = 2000
	// TRNSPerPRO is the number of TRNS that one PRO operates.
	TRNSPerPRO = 10
)

const (
	// ErrNothingToTransfer is the error for a transfer of units the entity
	// has none of in storage that its work hasn't claimed.
	ErrNothingToTransfer = cerrs.Error("the entity has none of the units in storage to transfer")
	// ErrNoTransport is the error for a transfer that the entity's TRNS
	// can't carry a single unit of: they have no capacity left this turn, or
	// no FUEL, or no PRO to operate them.
	ErrNoTransport = cerrs.Error("the entity's TRNS can't carry it: no capacity left, no FUEL, or no PRO to operate them")
	// ErrNoRoom is the error for a transfer to an entity that has no room
	// for a single unit of it.
	ErrNoRoom = cerrs.Error("the receiving entity has no room for it")
)

// TransportCapacity returns the mass that qty TRNS of tech level tl carry in
// a turn, in MU: 200 x TL^2 each (manual K2.1).
func TransportCapacity(tl, qty int) int {
	return 200 * tl * tl * max(0, qty)
}

// Transport is what a colony's TRNS did, or have done so far, in a turn.
type Transport struct {
	// Fleet is the TRNS the colony had when the turn started, which do the
	// turn's work even if they are transferred away in it, and Capacity the
	// MU they can carry.
	Fleet    []Unit `json:"fleet,omitempty"`
	Capacity int    `json:"capacity"`
	// Used is the MU they carried. Each transfer's mass is rounded up.
	Used int `json:"used"`
	// Crew is the PRO that operated the TRNS it took, the highest tech
	// levels first, and Fuel the FUEL they burned. The crew is not in the
	// turn's worker cadre.
	Crew int `json:"crew"`
	Fuel int `json:"fuel"`
	// Transfers lists what the colony sent, in the order of its orders.
	Transfers []Transfer `json:"transfers,omitempty"`
}

// Transfer is the outcome of a transfer order.
type Transfer struct {
	// To is the ID of the receiving colony.
	To int `json:"to"`
	// Unit is what was moved, and Ordered the quantity the order asked for.
	Unit    Unit `json:"unit"`
	Ordered int  `json:"ordered"`
}

// StartTransport starts the colony's transport for a turn: every TRNS it
// has, whether in storage or in use, is ready to carry, and nothing has been
// carried.
func (c *Colony) StartTransport() {
	t := Transport{}
	for _, list := range [][]Unit{c.Operational, c.DepotOperational, c.Cargo, c.Depot} {
		for _, u := range list {
			if u.Type == units.TRNS && u.Qty > 0 {
				t.Fleet = addUnits(t.Fleet, units.TRNS, u.TechLevel, u.Qty)
				t.Capacity += TransportCapacity(u.TechLevel, u.Qty)
			}
		}
	}
	c.Transport = t
}

// needs returns the crew and FUEL it takes to carry used MU in the turn. The
// TRNS of the highest tech levels are used first, so the crew is the least
// that can do it. The FUEL is used / TransportMUPerFuel, rounded up.
func (t Transport) needs(used int) (crew, fuel int) {
	trns, left := 0, used
	for tl := units.MaxTechLevel; tl >= units.MinTechLevel && left > 0; tl-- {
		each := TransportCapacity(tl, 1)
		n := min(count(units.TRNS, tl, t.Fleet), (left+each-1)/each)
		trns += n
		left -= n * each
	}
	return (trns + TRNSPerPRO - 1) / TRNSPerPRO, (used + TransportMUPerFuel - 1) / TransportMUPerFuel
}

// Transfer moves up to qty of the unit with the given code and tech level
// from the colony's storage into the storage of the colony to, and returns
// the number it moved. The colony's TRNS carry them (StartTransport must have
// been called this turn). It takes what there is: the order is cut down to
// the units the colony has in storage that its work hasn't claimed, to what
// its TRNS can still carry this turn with the FUEL and the free PRO it has,
// and to the whole units the receiver has room for. TRNS move themselves and
// need none of that. Nothing changes if it can't move a single unit:
// ErrNothingToTransfer, ErrNoTransport, or ErrNoRoom.
func (c *Colony) Transfer(to *Colony, code string, tl, qty int) (int, error) {
	d, _ := units.Lookup(code)
	name := code
	if tl != 0 {
		name = fmt.Sprintf("%s-%d", code, tl)
	}
	ordered := qty
	if qty = min(qty, c.Stored(code, tl)-c.Claimed(code, tl)); qty < 1 {
		return 0, fmt.Errorf("%s: %w", name, ErrNothingToTransfer)
	}
	t := &c.Transport
	// What it takes to carry k of them on top of what has been carried.
	load := func(k int) (used, crew, fuel int) {
		if code == units.TRNS {
			return t.Used, t.Crew, t.Fuel
		}
		used = t.Used + d.Mass(tl, k).Ceil()
		crew, fuel = t.needs(used)
		return used, crew, fuel
	}
	n := sort.Search(qty, func(k int) bool {
		used, crew, fuel := load(k + 1)
		stock := c.Stock(units.FUEL)
		if code == units.FUEL {
			stock -= k + 1
		}
		return used > t.Capacity || crew > c.FreePRO() || fuel-t.Fuel > stock
	})
	if n < 1 {
		return 0, fmt.Errorf("%s: %w", name, ErrNoTransport)
	}
	depot := (to.HasDepot() && d.DepotAllowed()) || (to.Type == CENC && (code == units.MTL || code == units.NMTL))
	if !depot {
		if n = to.Room(code, tl, n); n < 1 {
			return 0, fmt.Errorf("%s: %w", name, ErrNoRoom)
		}
	}
	used, crew, fuel := load(n)
	c.takeStored(code, tl, n)
	c.Spend(units.FUEL, fuel-t.Fuel)
	t.Used, t.Crew, t.Fuel = used, crew, fuel
	t.Transfers = append(t.Transfers, Transfer{To: to.ID, Unit: Unit{Type: code, TechLevel: tl, Qty: n}, Ordered: ordered})
	if depot {
		to.Depot = addUnits(to.Depot, code, tl, n)
	} else {
		to.Cargo = addUnits(to.Cargo, code, tl, n)
	}
	return n, nil
}

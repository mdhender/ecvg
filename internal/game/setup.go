// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// MaxShipsPerLocation is the most ships an empire may have set up at one
// location (manual F4.3.1).
const MaxShipsPerLocation = 99

// SetupItem is one kind of unit that a set up installs in the new entity.
type SetupItem struct {
	// Unit is the units to move. Its Type may be a population unit (UEM,
	// USK, PRO, SLD) or a cadre (CWRK, SPY), which takes its people along.
	Unit Unit
	// Stowed is set if units that can be assembled are to go into the new
	// entity's storage instead.
	Stowed bool
	// Kind and GroupNo are the group of the new entity that the assembled
	// units join, if any.
	Kind    GroupKind
	GroupNo int
}

// SetupWorkAvailable returns the EMU of assembly work that the colony's CWRK
// can still do for setup orders this turn: EMUPerCWRK each, less what they
// have done for them already.
func (c *Colony) SetupWorkAvailable() int {
	return max(0, max(0, c.Cadres.CWRK)*EMUPerCWRK-c.Construction.Setup)
}

// SetUp moves the units of one line of a setup order from the colony into
// the new entity to, and returns how many it moved. It takes what there is,
// and is limited by each of these in turn (manual F4.0):
//
//   - What the colony has: units in its storage that its work hasn't
//     claimed, never assembled ones; people that no cadre holds and that
//     aren't crewing its TRNS; or, for CWRK and SPY, the cadre it has, whose
//     members go along (a CWRK is a PRO and a USK, a SPY a PRO and an SLD).
//   - What the colony's TRNS can still carry this turn (see Transfer). A
//     population unit is 1 MU.
//   - The room in the new entity (manual F3.2.1), if it is enclosed: units
//     take their assembled volume if they are assembled there and their
//     stowed volume if not, and people 1 VU each. Assembled structure takes
//     none, and makes the room for what is listed after it.
//   - The assembly work: a unit that requires assembly, and isn't stowed, is
//     assembled in the new entity, which takes its mass in EMU out of *emu,
//     the work that the CWRK can do for the order. Units that can't be
//     assembled stay where they are.
//
// Assembled units join the item's group. Stored units go to the new entity's
// depot if it has one and they can be there. The move is added to the
// colony's Transport.Transfers, even if nothing could be moved.
func (c *Colony) SetUp(to *Colony, it SetupItem, emu *int) int {
	code, tl, ordered := it.Unit.Type, it.Unit.TechLevel, it.Unit.Qty
	d, _ := units.Lookup(code)
	none := func(int) int { return 0 }
	n := 0
	if d.Population() {
		// What a unit of it is made of, and how many the colony can spare.
		var pro, usk, sld, uem, have int
		proOut := none
		switch code {
		case units.UEM:
			uem, have = 1, c.Population.UEM
		case units.USK:
			usk, have = 1, c.FreeUSK()
		case units.PRO:
			pro, have = 1, c.FreePRO()-c.Transport.Crew
			proOut = func(k int) int { return k }
		case units.SLD:
			sld, have = 1, c.FreeSLD()
		case units.CWRK:
			pro, usk, have = 1, 1, min(c.Cadres.CWRK, c.Population.PRO, c.Population.USK)
		case units.SPY:
			pro, sld, have = 1, 1, min(c.Cadres.SPY, c.Population.PRO, c.Population.SLD)
		}
		each := pro + usk + sld + uem
		if each > 0 {
			n = c.loadLimit(min(ordered, have), func(k int) int { return k * each * PopulationUnitMass }, none, proOut)
			if to.Type.Enclosed() {
				n = min(n, max(0, to.SpaceFree().Floor())/(each*PopulationUnitVolume))
			}
		}
		if n = max(0, n); n > 0 {
			moved := Population{UEM: uem * n, USK: usk * n, PRO: pro * n, SLD: sld * n}
			c.Population = c.Population.Less(moved)
			to.Population = to.Population.Less(Population{UEM: -moved.UEM, USK: -moved.USK, PRO: -moved.PRO, SLD: -moved.SLD})
			switch code {
			case units.CWRK:
				c.Cadres.CWRK -= n
				to.Cadres.CWRK += n
			case units.SPY:
				c.Cadres.SPY -= n
				to.Cadres.SPY += n
			}
			c.load(n * each * PopulationUnitMass)
		}
	} else {
		assemble := d.RequiresAssembly && !it.Stowed
		mass := func(k int) int {
			if code == units.TRNS {
				return 0 // TRNS move themselves
			}
			return d.Mass(tl, k).Ceil()
		}
		fuelOut := none
		if code == units.FUEL {
			fuelOut = func(k int) int { return k }
		}
		n = c.loadLimit(min(ordered, c.Stored(code, tl)-c.Claimed(code, tl)), mass, fuelOut, none)
		depot := (to.HasDepot() && d.DepotAllowed()) || (to.Type == CENC && (code == units.MTL || code == units.NMTL))
		switch {
		case assemble && (d.Encloses() > 0 || depot || !to.Type.Enclosed()):
			// Structure, and what goes up in the depot, needs no room.
		case assemble:
			free := to.SpaceFree()
			n = sort.Search(n, func(k int) bool { return d.Volume(tl, k+1).Cmp(free) > 0 })
		case !depot:
			n = to.Room(code, tl, n)
		}
		if assemble {
			n = sort.Search(n, func(k int) bool { return workEMU(WorkAssemble, d, tl, k+1) > *emu })
		}
		if n = max(0, n); n > 0 {
			c.takeStored(code, tl, n)
			switch {
			case assemble:
				*emu -= workEMU(WorkAssemble, d, tl, n)
				to.Cargo = addUnits(to.Cargo, code, tl, n)
				to.install(d, tl, n)
				to.addToGroup(it.Kind, it.GroupNo, tl, n)
			case depot:
				to.Depot = addUnits(to.Depot, code, tl, n)
			default:
				to.Cargo = addUnits(to.Cargo, code, tl, n)
			}
			c.load(mass(n))
		}
	}
	c.Transport.Transfers = append(c.Transport.Transfers, Transfer{To: to.ID, Unit: Unit{Type: code, TechLevel: tl, Qty: n}, Ordered: ordered})
	return n
}

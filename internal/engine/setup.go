// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"
	"math"
	"slices"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
	"github.com/mdhender/ecvg/internal/units"
)

const (
	// ErrOneColonyOfAType is the error for a set up of a colony of a type
	// that the empire already has at the location (manual F4.3.1).
	ErrOneColonyOfAType = cerrs.Error("the empire already has a colony of that type at this location")
	// ErrTooManyShips is the error for a set up of a ship where the empire
	// has all the ships it may have at one location.
	ErrTooManyShips = cerrs.Error("the empire has all the ships it may have at this location")
	// ErrNotHabitable is the error for a set up of an open-air colony on a
	// planet that isn't habitable (manual F2.1).
	ErrNotHabitable = cerrs.Error("an open-air colony needs a habitable planet")
	// ErrNeedsPermission is the error for a set up of a colony on a planet
	// where other empires have colonies and the empire has none.
	ErrNeedsPermission = cerrs.Error("another empire has a colony on the planet, and permission to colonize is not accepted yet")
	// ErrNoPopulation and ErrNoSpaceDrive are the errors for a set up that
	// gets no population, or a ship that gets no SDRV, into the new entity.
	ErrNoPopulation = cerrs.Error("no population could be set up in the new entity")
	ErrNoSpaceDrive = cerrs.Error("no SDRV could be set up in the new ship")
	// ErrShipFactory is the error for a set up of a factory group in a ship.
	ErrShipFactory = cerrs.Error("ships do not have functioning factories: list the FACT units as stowed")
)

// Setup carries out a setup order, given by the empire with ID empireID: the
// entity From sets up new entities of the order's type at its own location
// (manual F4.0, S5.3). It is done at once, like a transfer. Each new entity
// gets the next entity ID, the tech level of the entity that set it up, and
// the units of the order's lines, taken in the order listed (see
// game.Colony.SetUp): the TRNS of From carry them, and its CWRK assemble
// them, in this turn only. The CWRK the order lists do that work first and
// stay in the new entity; the other CWRK of From help, as many as the order's
// using clause allows, and what they do comes out of From's assembly work
// for the turn (game.Construction.Setup).
//
// An order for several ships sets them up one after another. One that gets
// no population, or no SDRV, is not set up, and neither are the ones after
// it; the order fails only if that is the first.
//
// Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no entity From.
//   - orders.ErrInvalidQuantity: more than one colony.
//   - ErrOneColonyOfAType, ErrTooManyShips, ErrNotHabitable,
//     ErrNeedsPermission: the entity may not be set up there.
//   - ErrResearchNeedsPrototype: a line lists RSCH.
//   - ErrShipFactory, ErrTooManyGroups, game.ErrFarmSite, ErrFarmCap,
//     game.ErrMineSite, ErrNoSuchDeposit, ErrDepositClaimed,
//     game.ErrLabSite: the new entity can't have the group a line asks for.
//   - ErrNoPopulation, ErrNoSpaceDrive: nothing came of the first entity.
func Setup(d *game.Data, empireID int, o orders.Setup) error {
	fromAt := slices.IndexFunc(d.Colonies, func(c game.Colony) bool { return c.ID == o.From && c.EmpireID == empireID })
	if fromAt < 0 {
		return fmt.Errorf("entity %d: %w", o.From, ErrNotControlled)
	}
	planetID := d.Colonies[fromAt].PlanetID
	var planet game.Planet
	orbitNo := 0
	for _, p := range d.Planets {
		if p.ID == planetID {
			planet = p
			for _, orbit := range d.Orbits {
				if orbit.ID == p.OrbitID {
					orbitNo = orbit.OrbitNo
				}
			}
		}
	}

	// Where the entity may be set up, and how many.
	qty := max(1, o.Qty)
	ships, mine, others := 0, false, false
	for _, c := range d.Colonies {
		if c.PlanetID != planetID {
			continue
		}
		switch {
		case c.EmpireID != empireID:
			others = others || c.Type != game.CSHP
		case c.Type == game.CSHP:
			ships++
		default:
			mine = true
			if c.Type == o.Type {
				return fmt.Errorf("%s: %w", o.Type, ErrOneColonyOfAType)
			}
		}
	}
	switch {
	case o.Type == game.CSHP:
		if qty = min(qty, game.MaxShipsPerLocation-ships); qty < 1 {
			return fmt.Errorf("%d ships: %w", ships, ErrTooManyShips)
		}
	case qty != 1:
		return fmt.Errorf("%w: only one colony is set up at a time", orders.ErrInvalidQuantity)
	case o.Type == game.COPN && !planet.Habitable:
		return ErrNotHabitable
	case others && !mine:
		return ErrNeedsPermission
	}

	// What the lines ask for must be possible in an entity of this type.
	farms := 0 // FARM-1 units the order would assemble
	deposits := map[int]int{}
	for _, it := range o.Items {
		u := it.Unit
		def, _ := units.Lookup(u.Type)
		if u.Type == units.RSCH {
			return ErrResearchNeedsPrototype
		}
		if it.Stowed || !def.RequiresAssembly {
			continue
		}
		switch u.Type {
		case units.FACT:
			if o.Type == game.CSHP {
				return ErrShipFactory
			}
		case units.FARM:
			if err := game.CanInstallFarm(u.TechLevel, o.Type, orbitNo, planet); err != nil {
				return err
			}
			if u.TechLevel == 1 {
				farms += u.Qty
			}
		case units.MINE:
			if err := game.CanAssembleMines(o.Type); err != nil {
				return err
			}
			id := 0
			for _, dep := range d.Deposits {
				if dep.PlanetID == planetID && dep.DepositNo == it.DepositNo {
					id = dep.ID
				}
			}
			if id == 0 {
				return fmt.Errorf("deposit %d: %w", it.DepositNo, ErrNoSuchDeposit)
			}
			if _, twice := deposits[it.DepositNo]; twice {
				return fmt.Errorf("deposit %d: %w", it.DepositNo, ErrDepositClaimed)
			}
			for _, other := range d.Colonies {
				for _, g := range other.MiningGroups {
					if g.DepositID == id {
						return fmt.Errorf("deposit %d: %w", it.DepositNo, ErrDepositClaimed)
					}
				}
			}
			deposits[it.DepositNo] = id
		case units.LAB:
			if err := game.CanAssembleLabs(o.Type); err != nil {
				return err
			}
		}
	}
	if n := countItems(o.Items, units.FACT); n > game.MaxFactoryGroups {
		return fmt.Errorf("%d factory groups: %w", n, ErrTooManyGroups)
	}
	if farms > 0 {
		have := 0
		for _, other := range d.Colonies {
			if other.PlanetID != planetID {
				continue
			}
			have += other.BeingAssembled(units.FARM, 1)
			for _, g := range other.FarmingGroups {
				for _, gu := range g.Units {
					if gu.TechLevel == 1 {
						have += gu.Qty
					}
				}
			}
		}
		if have+farms > planet.FarmCap() {
			return fmt.Errorf("%d of %d are taken: %w", have, planet.FarmCap(), ErrFarmCap)
		}
	}

	// The helpers' work for the whole order, if the order caps it and the
	// cap is not for each entity.
	budget := math.MaxInt
	if o.MaxCWRK > 0 {
		budget = o.MaxCWRK * game.EMUPerCWRK
	}
	nextID, nextNo := 0, 0
	for _, c := range d.Colonies {
		nextID = max(nextID, c.ID)
		if c.EmpireID == empireID {
			nextNo = max(nextNo, c.No)
		}
	}
	for k := 0; k < qty; k++ {
		// Work on copies, so that an entity that fails changes nothing.
		from := d.Colonies[fromAt]
		from.Cargo, from.Depot = slices.Clone(from.Cargo), slices.Clone(from.Depot)
		from.Transport.Transfers = slices.Clone(from.Transport.Transfers)
		nextID++
		nextNo++
		to := game.Colony{
			ID: nextID, EmpireID: empireID, PlanetID: planetID, No: nextNo,
			Type: o.Type, TechLevel: from.TechLevel, Rations: game.DefaultRations,
			SOL: game.DefaultSOL, DeathRate: game.DefaultDeathRate, BirthRate: game.DefaultBirthRate,
		}

		// The CWRK the order lists work first, and then the helpers.
		listed := 0
		for _, it := range o.Items {
			if it.Unit.Type == units.CWRK {
				listed += it.Unit.Qty
			}
		}
		avail := from.SetupWorkAvailable()
		own := min(listed*game.EMUPerCWRK, avail)
		help := min(avail-own, budget)
		if o.Each && o.MaxCWRK > 0 {
			help = min(avail-own, o.MaxCWRK*game.EMUPerCWRK)
		}
		emu := own + help

		for _, it := range o.Items {
			item := game.SetupItem{Unit: it.Unit, Stowed: it.Stowed}
			if def, _ := units.Lookup(it.Unit.Type); !it.Stowed && def.RequiresAssembly {
				item.Kind, item.GroupNo = setupGroup(&to, it, deposits)
			}
			from.SetUp(&to, item, &emu)
		}
		dropEmptyGroups(&to)

		var err error
		switch {
		case to.Population.Units() < 1:
			err = ErrNoPopulation
		case o.Type == game.CSHP && !hasUnit(to, units.SDRV):
			err = ErrNoSpaceDrive
		}
		if err != nil {
			if k == 0 {
				return err
			}
			break
		}
		// The work is charged to the CWRK that went along first, and the
		// rest to the helpers, who stay.
		used := own + help - emu
		to.Construction.Setup = min(used, to.Cadres.CWRK*game.EMUPerCWRK)
		helped := used - to.Construction.Setup
		from.Construction.Setup += helped
		if !o.Each && o.MaxCWRK > 0 {
			budget = max(0, budget-helped)
		}
		d.Colonies[fromAt] = from
		d.Colonies = append(d.Colonies, to)
	}
	return nil
}

// countItems returns the number of lines that assemble the unit with the
// given code.
func countItems(items []orders.SetupItem, code string) int {
	n := 0
	for _, it := range items {
		if it.Unit.Type == code && !it.Stowed {
			n++
		}
	}
	return n
}

// setupGroup returns the group of the new entity c that the assembled units
// of the line it join, making the group if need be: a new factory or mining
// group for each line, and the farming or laboratory group of the units' tech
// level. Units that don't work in a group join none.
func setupGroup(c *game.Colony, it orders.SetupItem, deposits map[int]int) (game.GroupKind, int) {
	tl := it.Unit.TechLevel
	entry := func(us *[]game.GroupUnits) { _, *us = game.GroupUnitsAt(*us, tl) }
	switch it.Unit.Type {
	case units.FACT:
		no := c.NextGroupNo()
		c.FactoryGroups = append(c.FactoryGroups, game.Group{No: no, Producing: it.Producing})
		entry(&c.FactoryGroups[len(c.FactoryGroups)-1].Units)
		return game.KindFactory, no
	case units.MINE:
		no := c.NextGroupNo()
		c.MiningGroups = append(c.MiningGroups, game.MiningGroup{No: no, DepositID: deposits[it.DepositNo]})
		entry(&c.MiningGroups[len(c.MiningGroups)-1].Units)
		return game.KindMining, no
	case units.FARM:
		for _, g := range c.FarmingGroups {
			for _, gu := range g.Units {
				if gu.TechLevel == tl {
					return game.KindFarming, g.No
				}
			}
		}
		no := c.NextGroupNo()
		c.FarmingGroups = append(c.FarmingGroups, game.Group{No: no})
		entry(&c.FarmingGroups[len(c.FarmingGroups)-1].Units)
		return game.KindFarming, no
	case units.LAB:
		for _, g := range c.LaboratoryGroups {
			for _, gu := range g.Units {
				if gu.TechLevel == tl {
					return game.KindLaboratory, g.No
				}
			}
		}
		no := c.NextGroupNo()
		c.LaboratoryGroups = append(c.LaboratoryGroups, game.LaboratoryGroup{No: no})
		entry(&c.LaboratoryGroups[len(c.LaboratoryGroups)-1].Units)
		return game.KindLaboratory, no
	}
	return "", 0
}

// dropEmptyGroups removes the groups of c that got no units, so that a
// mining group with no mines doesn't hold a deposit.
func dropEmptyGroups(c *game.Colony) {
	c.FarmingGroups = slices.DeleteFunc(c.FarmingGroups, func(g game.Group) bool { return game.GroupQty(g.Units) == 0 })
	c.FactoryGroups = slices.DeleteFunc(c.FactoryGroups, func(g game.Group) bool { return game.GroupQty(g.Units) == 0 })
	c.MiningGroups = slices.DeleteFunc(c.MiningGroups, func(g game.MiningGroup) bool { return game.GroupQty(g.Units) == 0 })
	c.LaboratoryGroups = slices.DeleteFunc(c.LaboratoryGroups, func(g game.LaboratoryGroup) bool { return game.GroupQty(g.Units) == 0 })
}

// hasUnit reports whether c holds any of the unit with the given code,
// assembled or in storage.
func hasUnit(c game.Colony, code string) bool {
	for _, list := range [][]game.Unit{c.Operational, c.DepotOperational, c.Cargo, c.Depot} {
		for _, u := range list {
			if u.Type == code && u.Qty > 0 {
				return true
			}
		}
	}
	return false
}

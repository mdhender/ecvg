// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"
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
	// lists no population, or for a ship no SDRV, that the new entity has
	// room for.
	ErrNoPopulation = cerrs.Error("the order lists no population that the new entity has room for")
	ErrNoSpaceDrive = cerrs.Error("the order lists no SDRV that the new ship has room for")
	// ErrNoSuchSetup is the error for cancelling a set up that the empire
	// isn't doing.
	ErrNoSuchSetup = cerrs.Error("the empire is not setting up that entity")
	// ErrNoSuchAddOn is the error for cancelling an add-on that the entity
	// isn't doing.
	ErrNoSuchAddOn = cerrs.Error("the entity has no add-on to that entity")
	// ErrOwnSetup is the error for an add-on by the entity that is doing the
	// set up.
	ErrOwnSetup = cerrs.Error("the entity is doing that set up itself")
	// ErrNothingWanted is the error for an add-on that lists nothing that
	// the set up is short of.
	ErrNothingWanted = cerrs.Error("the set up is not short of anything the order lists")
	// ErrShipFactory is the error for a set up of a factory group in a ship.
	ErrShipFactory = cerrs.Error("ships do not have functioning factories: list the FACT units as stowed")
)

// Setup carries out a setup order, given by the empire with ID empireID: the
// entity From starts to set up new entities of the order's type at its own
// location (manual F4.0, S5.3). Each is a game.Setup of From, which gets the
// next entity ID, the tech level of From, and the units of the order's lines,
// in the order listed but people last, each line cut down to what the new
// entity has room for (game.Colony.PlanSetup). From reserves what it has of
// them, and the rest when it can (game.Colony.Supply). The work is done in
// the Setups phase, starting this turn.
//
// The ships of an order for several share the order's cap on CWRK, and so
// are built in turn, unless the cap is for each of them.
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
//   - ErrNoPopulation, ErrNoSpaceDrive: the entity would have no population,
//     or the ship no SDRV.
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

	// Where the entity may be set up, and how many. The entities that are
	// being set up count as if they were there.
	entities := allEntities(d)
	qty := max(1, o.Qty)
	ships, mine, others := 0, false, false
	nextID, nextNo := 0, 0
	for _, c := range entities {
		nextID = max(nextID, c.ID)
		if c.EmpireID == empireID {
			nextNo = max(nextNo, c.No)
		}
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
			if _, twice := deposits[it.DepositNo]; twice || depositClaimed(d, id) {
				return fmt.Errorf("deposit %d: %w", it.DepositNo, ErrDepositClaimed)
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
		for _, other := range entities {
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

	// People are handed over last, so they get the room that is left.
	items := slices.Clone(o.Items)
	slices.SortStableFunc(items, func(a, b orders.SetupItem) int {
		da, _ := units.Lookup(a.Unit.Type)
		db, _ := units.Lookup(b.Unit.Type)
		switch {
		case da.Population() == db.Population():
			return 0
		case db.Population():
			return -1
		}
		return 1
	})
	// Entities that share the order's crew are one batch.
	batch := 0
	if o.MaxCWRK > 0 && !o.Each {
		for _, s := range d.Colonies[fromAt].Setups {
			batch = max(batch, s.Batch)
		}
		batch++
	}
	for k := 0; k < qty; k++ {
		// Work on a copy, so that an entity that fails changes nothing.
		from := d.Colonies[fromAt]
		nextID++
		nextNo++
		from.Setups = append(slices.Clone(from.Setups), game.Setup{
			Entity: game.Colony{
				ID: nextID, EmpireID: empireID, PlanetID: planetID, No: nextNo,
				Type: o.Type, TechLevel: from.TechLevel, Rations: game.DefaultRations,
				SOL: game.DefaultSOL, DeathRate: game.DefaultDeathRate, BirthRate: game.DefaultBirthRate,
			},
			MaxCWRK: o.MaxCWRK, Batch: batch,
		})
		s := &from.Setups[len(from.Setups)-1]
		for _, it := range items {
			item := game.SetupItem{Unit: it.Unit, Stowed: it.Stowed}
			if def, _ := units.Lookup(it.Unit.Type); !it.Stowed && def.RequiresAssembly {
				item.Kind, item.GroupNo = setupGroup(&s.Entity, it, deposits)
			}
			from.PlanSetup(s, item)
		}
		dropEmptyGroups(&s.Entity)
		from.Supply(s)

		var err error
		switch {
		case s.Entity.Population.Units() < 1:
			err = ErrNoPopulation
		case o.Type == game.CSHP && !hasUnit(s.Entity, units.SDRV):
			err = ErrNoSpaceDrive
		}
		if err != nil {
			return err
		}
		d.Colonies[fromAt] = from
	}
	return nil
}

// CancelSetup carries out a cancel setup order, given by the empire with ID
// empireID: the set up of the entity with ID o.EntityID is dropped, and what
// was reserved for it is released (game.Colony.CancelSetup), and so is what
// add-ons to it had reserved. It fails, with ErrNoSuchSetup, if no entity of
// the empire is setting up that entity.
func CancelSetup(d *game.Data, empireID int, o orders.CancelSetup) error {
	for i := range d.Colonies {
		if c := &d.Colonies[i]; c.EmpireID == empireID && c.CancelSetup(o.EntityID) {
			dropAddOns(d, o.EntityID)
			return nil
		}
	}
	return fmt.Errorf("entity %d: %w", o.EntityID, ErrNoSuchSetup)
}

// dropAddOns removes every add-on to the entity with the given ID.
func dropAddOns(d *game.Data, id int) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.AddOns = slices.DeleteFunc(slices.Clone(c.AddOns), func(a game.AddOn) bool { return a.To == id })
		if len(c.AddOns) == 0 {
			c.AddOns = nil
		}
	}
}

// findSetup returns the colony of d that is setting up the entity with the
// given ID, and the set up, or nil if there is none.
func findSetup(d *game.Data, id int) (*game.Colony, *game.Setup) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		for k := range c.Setups {
			if c.Setups[k].Entity.ID == id {
				return c, &c.Setups[k]
			}
		}
	}
	return nil, nil
}

// AddOn carries out an add-on order, given by the empire with ID empireID:
// the entity From helps with the set up of the entity To, which another
// entity of the empire is doing at the same planet (manual F5.0, S5.4). Each
// line is cut down to what the set up is short of and no add-on has pledged
// (game.Colony.Pledge); From reserves what it has of that, and the rest when
// it can, and delivers it in the Setups phase, from the turn after the set
// up started (game.Colony.RunAddOn). The add-on lasts until it has delivered
// everything, or is cancelled, or the set up is.
//
// Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no entity From.
//   - ErrNoSuchSetup: no entity of the empire is setting up the entity To.
//   - ErrOwnSetup: From is doing the set up itself.
//   - ErrNotSameOrbit: From is not at the planet of the set up.
//   - ErrNothingWanted: the set up is short of nothing that the order lists.
func AddOn(d *game.Data, empireID int, o orders.AddOn) error {
	fromAt := slices.IndexFunc(d.Colonies, func(c game.Colony) bool { return c.ID == o.From && c.EmpireID == empireID })
	if fromAt < 0 {
		return fmt.Errorf("entity %d: %w", o.From, ErrNotControlled)
	}
	from := &d.Colonies[fromAt]
	builder, s := findSetup(d, o.To)
	switch {
	case s == nil || builder.EmpireID != empireID:
		return fmt.Errorf("entity %d: %w", o.To, ErrNoSuchSetup)
	case builder == from:
		return fmt.Errorf("entity %d: %w", o.From, ErrOwnSetup)
	case builder.PlanetID != from.PlanetID:
		return fmt.Errorf("entities %d and %d: %w", o.From, builder.ID, ErrNotSameOrbit)
	}
	wanted := false
	for _, u := range o.Items {
		for _, short := range s.Short() {
			if short.Type == u.Type && short.TechLevel == u.TechLevel && short.Qty > countUnits(s.Pledged, u.Type, u.TechLevel) {
				wanted = true
			}
		}
	}
	if !wanted {
		return ErrNothingWanted
	}
	from.AddOns = append(slices.Clone(from.AddOns), game.AddOn{To: o.To, MaxCWRK: o.MaxCWRK})
	a := &from.AddOns[len(from.AddOns)-1]
	for _, u := range o.Items {
		from.Pledge(a, s, u.Type, u.TechLevel, u.Qty)
	}
	return nil
}

// countUnits returns the quantity of the unit with the given code and tech
// level in list.
func countUnits(list []game.Unit, code string, tl int) int {
	n := 0
	for _, u := range list {
		if u.Type == code && u.TechLevel == tl {
			n += u.Qty
		}
	}
	return n
}

// CancelAddOn carries out a cancel add-on order, given by the empire with ID
// empireID: the entity From stops helping with the set up of the entity To.
// What it had reserved is released, and what it had still to deliver is the
// set up's own entity's to supply again; what it delivered stays. It fails
// with ErrNotControlled, or with ErrNoSuchAddOn if From has no such add-on.
func CancelAddOn(d *game.Data, empireID int, o orders.CancelAddOn) error {
	fromAt := slices.IndexFunc(d.Colonies, func(c game.Colony) bool { return c.ID == o.From && c.EmpireID == empireID })
	if fromAt < 0 {
		return fmt.Errorf("entity %d: %w", o.From, ErrNotControlled)
	}
	from := &d.Colonies[fromAt]
	_, s := findSetup(d, o.To)
	found := false
	var keep []game.AddOn
	for _, a := range from.AddOns {
		if a.To != o.To {
			keep = append(keep, a)
			continue
		}
		found = true
		if s != nil {
			for _, u := range a.Want {
				s.Pledged = lessPledged(s.Pledged, u)
			}
		}
	}
	if !found {
		return fmt.Errorf("entity %d: %w", o.To, ErrNoSuchAddOn)
	}
	from.AddOns = keep
	return nil
}

// lessPledged takes the units u off list, and drops the entry if none are
// left.
func lessPledged(list []game.Unit, u game.Unit) []game.Unit {
	var out []game.Unit
	for _, p := range list {
		if p.Type == u.Type && p.TechLevel == u.TechLevel {
			p.Qty -= u.Qty
		}
		if p.Qty > 0 {
			out = append(out, p)
		}
	}
	return out
}

// Setups is the set up phase of a turn. First every entity does a turn of
// its add-ons, oldest first (game.Colony.RunAddOn); an add-on that has
// delivered everything, or whose set up is gone, is dropped. Then every
// entity does a turn of work on the entities it is setting up
// (game.Colony.RunSetups), and the ones that are done join the game.
func Setups(d *game.Data) {
	for i := range d.Colonies {
		d.Colonies[i].Construction.Setup = 0
	}
	for i := range d.Colonies {
		c := &d.Colonies[i]
		var keep []game.AddOn
		for k := range c.AddOns {
			a := &c.AddOns[k]
			builder, s := findSetup(d, a.To)
			if s == nil || c.RunAddOn(a, builder, s) {
				// Nothing is reserved for it any more.
				a.BOM, a.Want = nil, nil
				continue
			}
			keep = append(keep, *a)
		}
		c.AddOns = keep
	}
	var done []game.Colony
	for i := range d.Colonies {
		done = append(done, d.Colonies[i].RunSetups()...)
	}
	d.Colonies = append(d.Colonies, done...)
}

// allEntities returns the entities of d and the ones that are being set up.
func allEntities(d *game.Data) []game.Colony {
	all := slices.Clone(d.Colonies)
	for _, c := range d.Colonies {
		for _, s := range c.Setups {
			all = append(all, s.Entity)
		}
	}
	return all
}

// depositClaimed reports whether a mining group of any entity, or of one
// that is being set up, works the deposit with the given ID.
func depositClaimed(d *game.Data, id int) bool {
	for _, c := range allEntities(d) {
		for _, g := range c.MiningGroups {
			if g.DepositID == id {
				return true
			}
		}
	}
	return false
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

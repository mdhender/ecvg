// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
	"github.com/mdhender/ecvg/internal/units"
)

const (
	ErrNotEnoughUnits     = cerrs.Error("the entity doesn't have that many of the unit in storage that aren't already being assembled or scrapped")
	ErrNotEnoughAssembled = cerrs.Error("the entity doesn't have that many of the unit assembled that aren't already being disassembled")
	ErrGroupUnit          = cerrs.Error("the unit works in a group: use the group assemble orders (groups can't be disassembled yet)")
	ErrNotScrappable      = cerrs.Error("the unit wasn't built from MTL and NMTL, so there is nothing to recover")
	ErrTechLevel          = cerrs.Error("every unit of the group must be of the same tech level")
	ErrNoSuchDeposit      = cerrs.Error("the entity's planet has no deposit with that number")
	ErrDepositClaimed     = cerrs.Error("another mining group is working that deposit")
	ErrFarmCap            = cerrs.Error("the planet has no room for that many more FARM-1 units")
)

// Disassembly is the dis-assembly phase: every colony's CWRK do a turn of
// its disassembly and scrap work (see game.Colony.Disassemble). It comes
// after production and before the Assembly phase, so what it returns to
// storage is there for the phases between, and it has first call on the
// CWRK. Like the draft, it runs in the setup turn too.
func Disassembly(d *game.Data) {
	for i := range d.Colonies {
		d.Colonies[i].Disassemble()
	}
}

// Assembly is the assembly phase: every colony's CWRK do a turn of its
// assembly work with what the Disassembly phase, which must come first, left
// them (see game.Colony.Assemble). It comes after production, so a unit
// assembled in a turn first works in the next. Like the draft, it runs in
// the setup turn too.
func Assembly(d *game.Data) {
	for i := range d.Colonies {
		d.Colonies[i].Assemble()
	}
}

// AssembleGroup carries out an assemble order for a group, given by the
// empire with ID empireID. It makes the new group, or finds the one the order
// adds to, and starts the assembly work for the order's units; the
// Assembly phase does the work. Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - ErrNoSuchGroup: the entity has no group to add to with that number.
//   - ErrTechLevel: the units aren't of the tech level of the group's units.
//   - ErrNotEnoughUnits: the entity doesn't have the units in storage, or
//     other assembly work has claimed them.
//   - game.ErrFarmSite, game.ErrMineSite: the entity can't have such a group.
//   - ErrFarmCap: the planet's FARM-1 units, everyone's together, would pass
//     its cap.
//   - ErrNoSuchDeposit, ErrDepositClaimed: a new mining group's deposit isn't
//     on the entity's planet, or a mining group of any empire works it.
func AssembleGroup(d *game.Data, empireID int, o orders.AssembleGroup) error {
	var c *game.Colony
	for i := range d.Colonies {
		if d.Colonies[i].ID == o.EntityID && d.Colonies[i].EmpireID == empireID {
			c = &d.Colonies[i]
		}
	}
	if c == nil {
		return fmt.Errorf("entity %d: %w", o.EntityID, ErrNotControlled)
	}

	// The group to add to, if the order names one.
	var us *[]game.GroupUnits
	if o.GroupNo != 0 {
		switch o.Kind {
		case game.KindFarming:
			for i := range c.FarmingGroups {
				if c.FarmingGroups[i].No == o.GroupNo {
					us = &c.FarmingGroups[i].Units
				}
			}
		case game.KindFactory:
			for i := range c.FactoryGroups {
				if c.FactoryGroups[i].No == o.GroupNo {
					us = &c.FactoryGroups[i].Units
				}
			}
		case game.KindMining:
			for i := range c.MiningGroups {
				if c.MiningGroups[i].No == o.GroupNo {
					us = &c.MiningGroups[i].Units
				}
			}
		}
		if us == nil {
			return fmt.Errorf("entity %d: %s group %d: %w", o.EntityID, o.Kind, o.GroupNo, ErrNoSuchGroup)
		}
		for _, gu := range *us {
			if gu.TechLevel != o.Unit.TechLevel {
				return fmt.Errorf("%s group %d is of tech level %d: %w", o.Kind, o.GroupNo, gu.TechLevel, ErrTechLevel)
			}
		}
	}

	// The entity must have the units, and not have promised them elsewhere.
	u := o.Unit
	if have := c.Stored(u.Type, u.TechLevel) - c.Claimed(u.Type, u.TechLevel); u.Qty > have {
		return fmt.Errorf("%s-%d: %d free in storage: %w", u.Type, u.TechLevel, max(0, have), ErrNotEnoughUnits)
	}

	switch o.Kind {
	case game.KindFarming:
		var planet game.Planet
		orbitNo := 0
		for _, p := range d.Planets {
			if p.ID == c.PlanetID {
				planet = p
				for _, orbit := range d.Orbits {
					if orbit.ID == p.OrbitID {
						orbitNo = orbit.OrbitNo
					}
				}
			}
		}
		if err := game.CanInstallFarm(u.TechLevel, c.Type, orbitNo, planet); err != nil {
			return err
		}
		if u.TechLevel != 1 {
			break
		}
		// The cap is on every empire's FARM-1 units together, counting
		// the ones still being assembled: first come, first served.
		have := 0
		for _, other := range d.Colonies {
			if other.PlanetID != c.PlanetID {
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
		if have+u.Qty > planet.FarmCap() {
			return fmt.Errorf("%d of %d are taken: %w", have, planet.FarmCap(), ErrFarmCap)
		}
	case game.KindMining:
		if err := game.CanAssembleMines(c.Type); err != nil {
			return err
		}
	}

	depositID := 0
	if o.Kind == game.KindMining && o.GroupNo == 0 {
		for _, dep := range d.Deposits {
			if dep.PlanetID == c.PlanetID && dep.DepositNo == o.DepositNo {
				depositID = dep.ID
			}
		}
		if depositID == 0 {
			return fmt.Errorf("deposit %d: %w", o.DepositNo, ErrNoSuchDeposit)
		}
		for _, other := range d.Colonies {
			for _, g := range other.MiningGroups {
				if g.DepositID == depositID {
					return fmt.Errorf("deposit %d: %w", o.DepositNo, ErrDepositClaimed)
				}
			}
		}
	}

	// Make the new group. Its number is the entity's next, whatever the kind
	// of group, so it is junior to all the entity's others.
	no := o.GroupNo
	if no == 0 {
		no = c.NextGroupNo()
		switch o.Kind {
		case game.KindFarming:
			c.FarmingGroups = append(c.FarmingGroups, game.Group{No: no})
			us = &c.FarmingGroups[len(c.FarmingGroups)-1].Units
		case game.KindFactory:
			c.FactoryGroups = append(c.FactoryGroups, game.Group{No: no, Producing: o.Producing})
			us = &c.FactoryGroups[len(c.FactoryGroups)-1].Units
		case game.KindMining:
			c.MiningGroups = append(c.MiningGroups, game.MiningGroup{No: no, DepositID: depositID})
			us = &c.MiningGroups[len(c.MiningGroups)-1].Units
		}
	}
	// The group has its entry of units from the start, so that the tech level
	// of a group with nothing assembled yet is known.
	_, *us = game.GroupUnitsAt(*us, u.TechLevel)
	c.Assembly = append(c.Assembly, game.Assembly{Kind: o.Kind, GroupNo: no, Unit: u, MaxCWRK: o.MaxCWRK})
	return nil
}

// UnitWork carries out an assemble, disassemble, or scrap order for units
// that aren't in a group, given by the empire with ID empireID. It starts the
// work; the Assembly phase does it. An order to assemble or disassemble a
// unit that is never assembled is accepted and does nothing. Nothing changes
// if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - ErrGroupUnit: the unit is one that works in a group (FACT, FARM, MINE),
//     which the group orders assemble.
//   - ErrNotScrappable: the unit wasn't built from MTL and NMTL.
//   - ErrNotEnoughUnits: to assemble or scrap, the entity doesn't have the
//     units in storage, or other work has claimed them. Scrap never takes
//     assembled units.
//   - ErrNotEnoughAssembled: to disassemble, the entity doesn't have the
//     units assembled, or other work is disassembling them.
func UnitWork(d *game.Data, empireID int, o orders.UnitWork) error {
	var c *game.Colony
	for i := range d.Colonies {
		if d.Colonies[i].ID == o.EntityID && d.Colonies[i].EmpireID == empireID {
			c = &d.Colonies[i]
		}
	}
	if c == nil {
		return fmt.Errorf("entity %d: %w", o.EntityID, ErrNotControlled)
	}
	def, _ := units.Lookup(o.Unit.Type)
	code, tl := o.Unit.Type, o.Unit.TechLevel
	name := code
	if tl != 0 {
		name = fmt.Sprintf("%s-%d", code, tl)
	}
	if o.Action == game.WorkScrap {
		if _, _, ok := def.Inputs(tl, 1); !ok {
			return fmt.Errorf("%s: %w", name, ErrNotScrappable)
		}
	} else {
		if !def.RequiresAssembly {
			return nil // nothing to do
		}
		if code == units.FACT || code == units.FARM || code == units.MINE {
			return fmt.Errorf("%s: %w", name, ErrGroupUnit)
		}
	}
	if o.Action == game.WorkDisassemble {
		if have := c.InUse(code, tl) - c.BeingDisassembled(code, tl); o.Unit.Qty > have {
			return fmt.Errorf("%s: %d assembled: %w", name, max(0, have), ErrNotEnoughAssembled)
		}
	} else if have := c.Stored(code, tl) - c.Claimed(code, tl); o.Unit.Qty > have {
		return fmt.Errorf("%s: %d free in storage: %w", name, max(0, have), ErrNotEnoughUnits)
	}
	c.Assembly = append(c.Assembly, game.Assembly{Action: o.Action, Unit: o.Unit, MaxCWRK: o.MaxCWRK})
	return nil
}

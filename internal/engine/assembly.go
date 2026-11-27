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
	ErrNotEnoughUnits = cerrs.Error("the entity doesn't have that many of the unit in storage that aren't already being assembled")
	ErrTechLevel      = cerrs.Error("every unit of the group must be of the same tech level")
	ErrNoSuchDeposit  = cerrs.Error("the entity's planet has no deposit with that number")
	ErrDepositClaimed = cerrs.Error("another mining group is working that deposit")
	ErrFarmCap        = cerrs.Error("the planet has no room for that many more FARM-1 units")
)

// Assembly is the assembly phase: every colony's CWRK do a turn of its
// assembly work (see game.Colony.Assemble). It comes after production, so a
// unit assembled in a turn first works in the next. Like the draft, it runs
// in the setup turn too.
func Assembly(d *game.Data) {
	for i := range d.Colonies {
		d.Colonies[i].Assemble()
	}
}

// AssembleGroup carries out an assemble order for a group, given by the
// empire with ID empireID. It makes the new group, or finds the one the order
// adds to, and starts the assembly work for each of the order's units; the
// Assembly phase does the work. Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - ErrNoSuchGroup: the entity has no group to add to with that number.
//   - ErrTechLevel: the units aren't of the tech level of a farming or mining
//     group's units.
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
		if o.Kind != game.KindFactory {
			for _, u := range *us {
				if len(o.Units) > 0 && u.TechLevel != o.Units[0].TechLevel {
					return fmt.Errorf("%s group %d is of tech level %d: %w", o.Kind, o.GroupNo, u.TechLevel, ErrTechLevel)
				}
			}
		}
	}

	// The entity must have the units, and not have promised them elsewhere.
	want := map[game.Unit]int{}
	for _, u := range o.Units {
		k := game.Unit{Type: u.Type, TechLevel: u.TechLevel}
		want[k] += u.Qty
		if have := c.Stored(u.Type, u.TechLevel) - c.Claimed(u.Type, u.TechLevel); want[k] > have {
			return fmt.Errorf("%s-%d: %d free in storage: %w", u.Type, u.TechLevel, max(0, have), ErrNotEnoughUnits)
		}
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
		for _, u := range o.Units {
			if err := game.CanInstallFarm(u.TechLevel, c.Type, orbitNo, planet); err != nil {
				return err
			}
			if u.TechLevel != 1 {
				continue
			}
			// The cap is on every empire's FARM-1 units together, counting
			// the ones still being assembled: first come, first served.
			have := 0
			for _, other := range d.Colonies {
				if other.PlanetID != c.PlanetID {
					continue
				}
				have += other.Claimed(units.FARM, 1)
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
	// The group has an entry for each tech level from the start, so that the
	// tech level of a group with nothing assembled yet is known.
	for _, u := range o.Units {
		_, *us = game.GroupUnitsAt(*us, u.TechLevel)
		c.Assembly = append(c.Assembly, game.Assembly{Kind: o.Kind, GroupNo: no, Unit: u})
	}
	return nil
}

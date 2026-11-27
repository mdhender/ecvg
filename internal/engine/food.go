// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import "github.com/mdhender/ecvg/internal/game"

// FoodUnitType is the type of the cargo unit that population eats.
const FoodUnitType = "FOOD"

// StarvationRation is the ration, as a percentage of a full ration, below
// which part of a population starves.
const StarvationRation = 25

// Meal is what one entity's population ate in a turn.
type Meal struct {
	// Units is the number of population units that ate.
	Units int
	// Wanted is the FOOD the population would eat at the entity's rations,
	// and Eaten the FOOD it ate, which is less if the entity ran short.
	Wanted, Eaten int
}

// Feed returns the meal of the population p in an entity that is on rations
// percent rations and holds food FOOD. The population eats
// p.FoodWanted(rations) if the entity has it, and if not, all there is.
func Feed(p game.Population, rations, food int) Meal {
	wanted := p.FoodWanted(max(rations, 0))
	return Meal{Units: p.Units(), Wanted: wanted, Eaten: min(wanted, max(food, 0))}
}

// Ration returns the ration the population got, as a percentage of a full
// ration of 0.25 FOOD per unit. It is worked out from the FOOD eaten, so it
// can differ from the entity's rations: it is less if the entity ran short,
// and a little more when the FOOD eaten was rounded up to a whole unit.
func (m Meal) Ration() float64 {
	if m.Units == 0 {
		return 0
	}
	return float64(m.Eaten*game.PopulationUnitsPerFood*100) / float64(m.Units)
}

// Starved returns the part of the population p, the population that ate the
// meal, that starves to death after it. Nobody starves on a ration R (see
// Ration) of StarvationRation percent or more. Below that, max(1, 100-4R)
// percent of the units starve, rounded up to a whole unit.
//
// The deaths are shared among the types of unit in proportion to their
// numbers, each share rounded down. The deaths left over are taken first from
// UEM, then SLD, then USK, then PRO.
func (m Meal) Starved(p game.Population) game.Population {
	units := p.Units()
	// R is 400*Eaten/units, so R < 25 is 16*Eaten < units.
	if units == 0 || 16*m.Eaten >= units {
		return game.Population{}
	}
	// The percentage is (100*units - 1600*Eaten)/units, and at least 1.
	pct := max(100*units-16*100*m.Eaten, units)
	deaths := (pct + 99) / 100 // pct/units percent of units, rounded up
	return p.Deaths(deaths)
}

// Rations is the rations phase of the turn numbered turn: the population of
// every entity in d eats its meal (see Feed) from the FOOD in the entity's
// cargo. It returns the meals, one per colony in d.Colonies order. Nothing is
// eaten in turn SetupTurn, and it returns nil.
func Rations(d *game.Data, turn int) []Meal {
	if turn <= SetupTurn {
		return nil
	}
	meals := make([]Meal, len(d.Colonies))
	for i := range d.Colonies {
		c := &d.Colonies[i]
		food := 0
		for _, u := range c.Cargo {
			if u.Type == FoodUnitType {
				food += u.Qty
			}
		}
		meals[i] = Feed(c.Population, c.Rations, food)
		left := meals[i].Eaten
		for j := range c.Cargo {
			if u := &c.Cargo[j]; u.Type == FoodUnitType {
				n := min(left, u.Qty)
				u.Qty -= n
				left -= n
			}
		}
	}
	return meals
}

// PopulationChanges is the population changes phase. So far it only
// starves: each colony in d loses the part of its population that starves
// after its meal (see Meal.Starved), and the colony's Starved records the
// loss. Meals is what Rations returned; with no meals nobody starves.
func PopulationChanges(d *game.Data, meals []Meal) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Starved = game.Population{}
		if i >= len(meals) {
			continue
		}
		s := meals[i].Starved(c.Population)
		c.Starved = s
		c.Population.UEM -= s.UEM
		c.Population.USK -= s.USK
		c.Population.PRO -= s.PRO
		c.Population.SLD -= s.SLD
	}
}

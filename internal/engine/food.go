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
	// Wanted is the FOOD the population would eat at the entity's rations,
	// and Eaten the FOOD it ate, which is less if the entity ran short.
	Wanted, Eaten int
	// The ration eaten is num/den percent of a full ration.
	num, den int
}

// Feed returns the meal of the population p in an entity that is on rations
// percent rations and holds food FOOD. The population eats
// p.FoodWanted(rations) if the entity has it. If not, it eats all there is,
// and the ration it gets is that much FOOD shared among its units.
func Feed(p game.Population, rations, food int) Meal {
	rations, food = max(rations, 0), max(food, 0)
	wanted := p.FoodWanted(rations)
	if food >= wanted {
		return Meal{Wanted: wanted, Eaten: wanted, num: rations, den: 1}
	}
	return Meal{
		Wanted: wanted, Eaten: food,
		num: food * game.PopulationUnitsPerFood * 100, den: p.Units(), // wanted > 0, so there are units
	}
}

// Ration returns the ration the population got, as a percentage of a full
// ration.
func (m Meal) Ration() float64 {
	if m.den == 0 {
		return 0
	}
	return float64(m.num) / float64(m.den)
}

// Starved returns the part of the population p that starves to death after
// the meal. Nobody starves on a ration R of StarvationRation percent or more.
// Below that, max(1, 100-4R) percent of the units of each type starve,
// rounded up to a whole unit.
func (m Meal) Starved(p game.Population) game.Population {
	if m.den == 0 || m.num >= StarvationRation*m.den {
		return game.Population{}
	}
	// The percentage that starves is pctNum/m.den.
	pctNum := max(100*m.den-4*m.num, m.den)
	starve := func(units int) int {
		return (units*pctNum + 100*m.den - 1) / (100 * m.den)
	}
	return game.Population{UEM: starve(p.UEM), USK: starve(p.USK), PRO: starve(p.PRO), SLD: starve(p.SLD)}
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

// PopulationIncrease is the population increases phase. So far it only
// starves: each colony in d loses the part of its population that starves
// after its meal (see Meal.Starved), and the colony's Starved records the
// loss. Meals is what Rations returned; with no meals nobody starves.
func PopulationIncrease(d *game.Data, meals []Meal) {
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

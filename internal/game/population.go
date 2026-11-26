// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import "fmt"

// EntityType is a kind of entity: something that holds population and units.
type EntityType string

const (
	CSHP EntityType = "CSHP" // ship
	COPN EntityType = "COPN" // open-air surface colony
	CENC EntityType = "CENC" // enclosed surface colony
	CORB EntityType = "CORB" // enclosed orbital colony
)

// UnmarshalText implements encoding.TextUnmarshaler, accepting only the four
// entity types.
func (t *EntityType) UnmarshalText(text []byte) error {
	switch v := EntityType(text); v {
	case CSHP, COPN, CENC, CORB:
		*t = v
		return nil
	}
	return fmt.Errorf("invalid entity type %q", text)
}

// Enclosed reports whether the entity is a ship or an enclosed colony, where
// population takes up space and needs life support. Only an open-air colony
// is not enclosed.
func (t EntityType) Enclosed() bool {
	return t != COPN
}

// What one population unit of any type amounts to.
const (
	// PeoplePerPopulationUnit is the number of people in a population unit.
	PeoplePerPopulationUnit = 100
	// PopulationUnitMass is a population unit's mass, in MU.
	PopulationUnitMass = 1
	// PopulationUnitVolume is the space a population unit takes up on a ship
	// or in an enclosed colony, in VU.
	PopulationUnitVolume = 1
	// PopulationUnitLifeSupport is the life support a population unit needs on
	// a ship or in an enclosed colony, in LU.
	PopulationUnitLifeSupport = 1
	// PopulationUnitsPerFood is the number of population units that one FOOD
	// feeds for a turn at the initial ration: each unit eats 0.25 FOOD.
	PopulationUnitsPerFood = 4
)

// Population counts an entity's population units by type. Each unit is a
// block of PeoplePerPopulationUnit people.
type Population struct {
	UEM int `json:"uem"` // unemployables
	USK int `json:"usk"` // unskilled workers
	PRO int `json:"pro"` // professionals
	SLD int `json:"sld"` // soldiers
}

// Units returns the number of population units of all types.
func (p Population) Units() int {
	return p.UEM + p.USK + p.PRO + p.SLD
}

// People returns the number of people in the population.
func (p Population) People() int {
	return p.Units() * PeoplePerPopulationUnit
}

// Mass returns the population's mass, in MU.
func (p Population) Mass() int {
	return p.Units() * PopulationUnitMass
}

// Volume returns the space the population takes up in an entity of type t, in
// VU: nothing unless the entity is enclosed.
func (p Population) Volume(t EntityType) int {
	if !t.Enclosed() {
		return 0
	}
	return p.Units() * PopulationUnitVolume
}

// LifeSupport returns the life support the population needs in an entity of
// type t, in LU: nothing unless the entity is enclosed.
func (p Population) LifeSupport(t EntityType) int {
	if !t.Enclosed() {
		return 0
	}
	return p.Units() * PopulationUnitLifeSupport
}

// Food returns the FOOD the population eats in a turn at the initial ration of
// 0.25 FOOD per unit, rounded up to a whole FOOD.
func (p Population) Food() int {
	return (p.Units() + PopulationUnitsPerFood - 1) / PopulationUnitsPerFood
}

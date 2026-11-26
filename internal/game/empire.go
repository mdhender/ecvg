// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// RacesFileName is the name of the file in a game's folder that holds the
// game's races. EmpiresFileName and ColoniesFileName are the names of the
// files in a turn's folder that hold the empires and their colonies.
const (
	RacesFileName    = "races.json"
	EmpiresFileName  = "empires.json"
	ColoniesFileName = "colonies.json"
)

// MinHandleLength and MaxHandleLength bound the length of a player handle.
const (
	MinHandleLength = 8
	MaxHandleLength = 12
)

// DefaultRations is the rations of a new entity, a percentage of the food its
// population needs. MinRations and MaxRations bound what it can be set to.
const (
	DefaultRations = 100
	MinRations     = 1
	MaxRations     = 500
)

const ErrInvalidHandle = cerrs.Error("invalid handle")

// Race is a species. Each race starts on one home world, and every empire on
// that home world belongs to it.
type Race struct {
	// ID is the ID of the race's home world.
	ID int `json:"id"`
	// HomeWorldID is the ID of the planet the race started on.
	HomeWorldID int `json:"homeworld_id"`
}

// Empire is an empire. The generator creates every empire in the game; an
// empire has no player until it is given a handle.
type Empire struct {
	// ID numbers the game's empires from 1, in home world order.
	ID int `json:"id"`
	// Handle identifies the empire's player: MinHandleLength..MaxHandleLength
	// lowercase ASCII letters, or empty if the empire has no player yet.
	Handle string `json:"handle"`
	// Name is the empire's name, by default "Empire #ID".
	Name string `json:"name"`
	// RaceID is the ID of the empire's race.
	RaceID int `json:"race_id"`
	// HomeWorldID is the ID of the planet the empire started on.
	HomeWorldID int `json:"homeworld_id"`
}

// Colony is one of an empire's colonies on a planet.
type Colony struct {
	// ID numbers the game's colonies from 1, in empire order.
	ID int `json:"id"`
	// EmpireID is the ID of the empire that owns the colony.
	EmpireID int `json:"empire_id"`
	// PlanetID is the ID of the planet the colony is on.
	PlanetID int `json:"planet_id"`
	// No numbers the empire's colonies from 1.
	No int `json:"no"`
	// Type is the kind of entity the colony is.
	Type      EntityType `json:"type"`
	TechLevel int        `json:"tech_level"`
	// Population is the colony's population units.
	Population Population `json:"population"`
	// Rations is the percentage of the food its population needs that the
	// colony hands out, MinRations..MaxRations, by default DefaultRations.
	Rations int `json:"rations"`
	// Starved is the population that starved to death in the prior turn.
	Starved Population `json:"starved,omitzero"`
	// Harvest is what the colony's farming groups did in the prior turn.
	Harvest Harvest `json:"harvest,omitzero"`
	// SOL is the standard of living, by default DefaultSOL.
	SOL float64 `json:"sol"`
	// DeathRate and BirthRate are fractions of the population per turn (0.01
	// is 1%), by default DefaultDeathRate and DefaultBirthRate.
	DeathRate float64 `json:"drt"`
	BirthRate float64 `json:"brt"`
	// Operational lists the units in use, and Cargo the units in storage.
	Operational []Unit `json:"operational"`
	Cargo       []Unit `json:"cargo"`
	// The groups put the operational farm, mine, and factory units to work.
	FarmingGroups []Group       `json:"farming_groups"`
	MiningGroups  []MiningGroup `json:"mining_groups"`
	FactoryGroups []Group       `json:"factory_groups"`
}

// Unit is a quantity of one kind of unit, such as FOOD or FACT.
type Unit struct {
	Type string `json:"type"`
	// TechLevel is 0 for units that don't have one.
	TechLevel int `json:"tech_level,omitzero"`
	Qty       int `json:"qty"`
}

// Group is a farming or factory group.
type Group struct {
	// No numbers the colony's groups of one kind from 1.
	No    int          `json:"no"`
	Units []GroupUnits `json:"units"`
}

// MiningGroup is a group of mines working one deposit.
type MiningGroup struct {
	No int `json:"no"`
	// DepositID is the ID of the deposit the group works.
	DepositID int          `json:"deposit_id"`
	Units     []GroupUnits `json:"units"`
}

// GroupUnits is a quantity of a group's units at one tech level.
type GroupUnits struct {
	Qty       int `json:"qty"`
	TechLevel int `json:"tech_level"`
}

// ValidateHandle checks that handle is MinHandleLength..MaxHandleLength
// lowercase ASCII letters.
func ValidateHandle(handle string) error {
	if n := len(handle); n < MinHandleLength || n > MaxHandleLength {
		return fmt.Errorf("%w: %q: want %d..%d letters, got %d", ErrInvalidHandle, handle, MinHandleLength, MaxHandleLength, n)
	}
	for _, r := range handle {
		if r < 'a' || r > 'z' {
			return fmt.Errorf("%w: %q: want only lowercase ASCII letters", ErrInvalidHandle, handle)
		}
	}
	return nil
}

// LoadRaces reads races.json from dir.
func LoadRaces(dir string) ([]Race, error) {
	var races []Race
	if err := loadJSON(dir, RacesFileName, &races); err != nil {
		return nil, err
	}
	return races, nil
}

// LoadEmpires reads empires.json from dir, a turn's folder.
func LoadEmpires(dir string) ([]Empire, error) {
	var empires []Empire
	if err := loadJSON(dir, EmpiresFileName, &empires); err != nil {
		return nil, err
	}
	return empires, nil
}

// LoadColonies reads colonies.json from dir, a turn's folder.
func LoadColonies(dir string) ([]Colony, error) {
	var colonies []Colony
	if err := loadJSON(dir, ColoniesFileName, &colonies); err != nil {
		return nil, err
	}
	return colonies, nil
}

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
	// Cadres is the cadres drafted from the population, and Drafted what the
	// draft orders of the prior turn asked for and got.
	Cadres  Cadres    `json:"cadres,omitzero"`
	Drafted []Drafted `json:"drafted,omitempty"`
	// Disbanded is the same for the disband orders.
	Disbanded []Drafted `json:"disbanded,omitempty"`
	// Trained is what became of the TRNE cadre in the prior turn.
	Trained Trained `json:"trained,omitzero"`
	// Assembly is the colony's assembly work in progress, oldest first, and
	// Construction what its CWRK did in the prior turn.
	Assembly     []Assembly   `json:"assembly,omitempty"`
	Construction Construction `json:"construction,omitzero"`
	// Rations is the percentage of the food its population needs that the
	// colony hands out, MinRations..MaxRations, by default DefaultRations.
	Rations int `json:"rations"`
	// Starved is the population that starved to death in the prior turn.
	Starved Population `json:"starved,omitzero"`
	// Workers is the colony's worker cadre of the prior turn, and what was
	// left of it.
	Workers Workers `json:"workers,omitzero"`
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
	// Depot lists the units stored outside the entity, and DepotOperational
	// the units assembled and in use there. The depot is unlimited and what
	// is in it takes none of the colony's space. Only an open-air surface
	// colony has one (HasDepot), and only units whose class allows it can be
	// there (units.Def.DepotAllowed).
	Depot            []Unit `json:"depot,omitempty"`
	DepotOperational []Unit `json:"depot_operational,omitempty"`
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

// GroupKind is a kind of group.
type GroupKind string

const (
	KindFarming GroupKind = "farming"
	KindMining  GroupKind = "mining"
	KindFactory GroupKind = "factory"
)

// The percentage of a group's units that an empire may order to idle.
const (
	MinIdlePercent = 0
	MaxIdlePercent = 100
)

// IdleUnits returns the number of qty units that sit idle when their group is
// ordered percent percent idle, rounded up to a whole unit.
func IdleUnits(qty, percent int) int {
	percent = max(MinIdlePercent, min(MaxIdlePercent, percent))
	return (max(0, qty)*percent + 99) / 100
}

// Group is a farming or factory group. Every unit of a farming group is of
// the same tech level, so its Units has one entry; a factory group can mix
// tech levels.
type Group struct {
	// No numbers the colony's groups of one kind from 1. A lower number is
	// more senior, and gets resources first.
	No int `json:"no"`
	// IdlePercent is the percentage of the group's units that the empire has
	// ordered to idle, MinIdlePercent..MaxIdlePercent. Idled units ask for no
	// staff or fuel. It stands until the empire changes it.
	IdlePercent int          `json:"idle_percent,omitzero"`
	Units       []GroupUnits `json:"units"`
	// Producing is what a factory group is to make, if its empire has said.
	Producing Unit `json:"producing,omitzero"`
}

// MiningGroup is a group of mines working one deposit.
type MiningGroup struct {
	No int `json:"no"`
	// IdlePercent is as for Group.
	IdlePercent int `json:"idle_percent,omitzero"`
	// DepositID is the ID of the deposit the group works. A deposit is
	// worked by at most one mining group.
	DepositID int `json:"deposit_id"`
	// Units has one entry: every unit of a mining group is of the same tech
	// level.
	Units []GroupUnits `json:"units"`
	// Mined is what the group did in the prior turn.
	Mined Mined `json:"mined,omitzero"`
}

// GroupUnits is a quantity of a group's units at one tech level.
type GroupUnits struct {
	Qty       int `json:"qty"`
	TechLevel int `json:"tech_level"`
	// Pipeline is for the FARM-1 units of a farming group: the number of
	// them with 1, 2, and 3 consecutive turns of work behind them. See
	// AdvanceCrop.
	Pipeline []int `json:"pipeline,omitempty"`
	// Idle counts the units that didn't work in the prior turn, by why.
	Idle Idle `json:"idle,omitzero"`
}

// Idle counts the units of a group that didn't work in a turn. There are two
// levels of idle: the empire ordered it, or resource allocation couldn't
// supply the unit.
type Idle struct {
	// Ordered units were idled by the empire's order (the group's
	// IdlePercent), and asked for nothing.
	Ordered int `json:"ordered,omitzero"`
	// Unstaffed units got no staff, and Unfuelled units got staff but no
	// FUEL.
	Unstaffed int `json:"unstaffed,omitzero"`
	Unfuelled int `json:"unfuelled,omitzero"`
}

// Units returns the number of idle units.
func (i Idle) Units() int {
	return i.Ordered + i.Unstaffed + i.Unfuelled
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

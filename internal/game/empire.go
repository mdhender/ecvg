// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// RacesFileName and EmpiresFileName are the names of the files in a game's
// folder that hold the game's races and empires. Neither exists until the
// first empire is added.
const (
	RacesFileName   = "races.json"
	EmpiresFileName = "empires.json"
)

// MinHandleLength and MaxHandleLength bound the length of a player handle.
const (
	MinHandleLength = 8
	MaxHandleLength = 12
)

const (
	ErrInvalidHandle   = cerrs.Error("invalid handle")
	ErrDuplicateHandle = cerrs.Error("handle already has an empire")
	ErrNoHomeWorld     = cerrs.Error("no home world available")
)

// Race is a species. Each race starts on one home world, and every empire on
// that home world belongs to it.
type Race struct {
	// ID is the ID of the race's home world.
	ID int `json:"id"`
	// HomeWorldID is the ID of the planet the race started on.
	HomeWorldID int `json:"homeworld_id"`
}

// Empire is a player's empire.
type Empire struct {
	// ID numbers the game's empires from 1, in the order they were added.
	ID int `json:"id"`
	// Handle identifies the player: MinHandleLength..MaxHandleLength
	// lowercase ASCII letters.
	Handle string `json:"handle"`
	// Name is the empire's name, by default "Empire #ID".
	Name string `json:"name"`
	// RaceID is the ID of the empire's race.
	RaceID int `json:"race_id"`
	// HomeWorldID is the ID of the planet the empire started on.
	HomeWorldID int `json:"homeworld_id"`
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

// HomeWorlds returns the IDs of the game's home worlds, lowest first: the
// planets in orbit HomeWorldOrbitNo of the home stars.
func (d Data) HomeWorlds() []int {
	homeOrbits := map[int]bool{}
	for _, o := range d.Orbits {
		if o.OrbitNo == HomeWorldOrbitNo &&
			slices.ContainsFunc(d.Stars, func(s Star) bool { return s.ID == o.StarID && s.HomeStar }) {
			homeOrbits[o.ID] = true
		}
	}
	var ids []int
	for _, p := range d.Planets {
		if homeOrbits[p.OrbitID] {
			ids = append(ids, p.ID)
		}
	}
	slices.Sort(ids)
	return ids
}

// EmpireCapacity returns the most empires that can share the home world with
// ID planetID: the number of its FUEL, MTL, and NMTL deposits. GOLD deposits
// don't count.
func (d Data) EmpireCapacity(planetID int) int {
	n := 0
	for _, x := range d.Deposits {
		if x.PlanetID == planetID && x.Type != GOLD {
			n++
		}
	}
	return n
}

// AddEmpire adds an empire for the player with handle to the first available
// home world: the one with the lowest ID that has fewer empires than its
// EmpireCapacity. If it is the first empire on that home
// world, AddEmpire also adds a race whose ID is the home world's ID, and
// reports newRace. The empire's ID is one more than the largest empire ID so
// far, and its name is name, or "Empire #ID" if name is empty.
//
// It fails with ErrInvalidHandle or ErrDuplicateHandle if handle is not valid
// or already has an empire, and with ErrNoHomeWorld if every home world is
// full or there are none. On error, d is unchanged.
func (d *Data) AddEmpire(handle, name string) (e Empire, newRace bool, err error) {
	if err := ValidateHandle(handle); err != nil {
		return Empire{}, false, err
	}
	if slices.ContainsFunc(d.Empires, func(e Empire) bool { return e.Handle == handle }) {
		return Empire{}, false, fmt.Errorf("%q: %w", handle, ErrDuplicateHandle)
	}

	onWorld := map[int]int{}
	for _, e := range d.Empires {
		onWorld[e.HomeWorldID]++
	}
	homeWorld := 0
	for _, id := range d.HomeWorlds() {
		if onWorld[id] < d.EmpireCapacity(id) {
			homeWorld = id
			break
		}
	}
	if homeWorld == 0 {
		return Empire{}, false, ErrNoHomeWorld
	}

	e = Empire{
		ID:          maxID(d.Empires, func(e Empire) int { return e.ID }) + 1,
		Handle:      handle,
		Name:        name,
		RaceID:      homeWorld,
		HomeWorldID: homeWorld,
	}
	if e.Name == "" {
		e.Name = fmt.Sprintf("Empire #%d", e.ID)
	}
	if !slices.ContainsFunc(d.Races, func(r Race) bool { return r.ID == homeWorld }) {
		d.Races = append(slices.Clone(d.Races), Race{ID: homeWorld, HomeWorldID: homeWorld})
		newRace = true
	}
	d.Empires = append(slices.Clone(d.Empires), e)
	return e, newRace, nil
}

// AddEmpire loads the game in dir, applies Data.AddEmpire, and saves the races
// and empires.
func AddEmpire(dir, handle, name string) (e Empire, newRace bool, err error) {
	d, err := LoadData(dir)
	if err != nil {
		return Empire{}, false, err
	}
	e, newRace, err = d.AddEmpire(handle, name)
	if err != nil {
		return Empire{}, false, err
	}
	err = replaceFiles(dir, []struct {
		name string
		v    any
	}{
		{RacesFileName, d.Races},
		{EmpiresFileName, d.Empires},
	})
	if err != nil {
		return Empire{}, false, err
	}
	return e, newRace, nil
}

// LoadRaces reads races.json from dir. A missing file means no races.
func LoadRaces(dir string) ([]Race, error) {
	var races []Race
	if err := loadJSON(dir, RacesFileName, &races); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return races, nil
}

// LoadEmpires reads empires.json from dir. A missing file means no empires.
func LoadEmpires(dir string) ([]Empire, error) {
	var empires []Empire
	if err := loadJSON(dir, EmpiresFileName, &empires); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return empires, nil
}

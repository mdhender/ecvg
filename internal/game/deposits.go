// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/prng"
)

// DepositsFileName is the name of the file in a game's folder that holds the
// planets' deposits.
const DepositsFileName = "deposits.json"

// DepositType is the natural resource in a deposit.
type DepositType string

const (
	GOLD DepositType = "GOLD"
	FUEL DepositType = "FUEL"
	MTL  DepositType = "MTL"  // metallics
	NMTL DepositType = "NMTL" // non-metallics
)

// UnmarshalText implements encoding.TextUnmarshaler, accepting only the four
// deposit types.
func (t *DepositType) UnmarshalText(text []byte) error {
	switch v := DepositType(text); v {
	case GOLD, FUEL, MTL, NMTL:
		*t = v
		return nil
	}
	return fmt.Errorf("invalid deposit type %q", text)
}

// NumDeposits is the number of deposits in a game: an average of 16 for each
// of the NumPlanets planets.
const NumDeposits = 9936

// depositTypes gives the number of deposits of each type, in the order they
// are dealt before shuffling: 34% FUEL, 32% MTL, 32% NMTL (rounded to the
// nearest deposit), and the remainder GOLD.
var depositTypes = []struct {
	typ   DepositType
	count int
}{
	{GOLD, 198},
	{FUEL, 3378},
	{MTL, 3180},
	{NMTL, 3180},
}

// planetsWithDeposits[n] is the number of planets with n deposits. It is the
// distribution of 1 + the sum of three 11-sided dice numbered 0 to 10 (3d11),
// scaled to NumPlanets planets and rounded so that it stays symmetric about
// 16. Being symmetric, the planets average exactly 16 deposits each, for
// NumDeposits in all. No planet has 1 or 31, the rarest rolls.
var planetsWithDeposits = [32]int{
	0, // unused: every planet has a deposit
	// 1..10
	0, 1, 3, 5, 7, 10, 13, 17, 21, 26,
	// 11..20
	31, 35, 38, 40, 42, 43, 42, 40, 38, 35,
	// 21..31
	31, 26, 21, 17, 13, 10, 7, 5, 3, 1, 0,
}

// Deposit is a deposit of a natural resource on a planet.
type Deposit struct {
	// ID numbers all of the game's deposits from 1, in planet order.
	ID int `json:"id"`
	// PlanetID is the ID of the planet the deposit is on.
	PlanetID int `json:"planet_id"`
	// DepositNo numbers the deposits on a planet from 1.
	DepositNo int `json:"deposit_no"`
	// Type is the resource in the deposit.
	Type DepositType `json:"type"`
}

// GenerateDeposits gives the planets NumDeposits deposits using the stream
// root.Derive(TagDeposits). The number of deposits on each planet comes from a
// fixed list on a bell curve (see planetsWithDeposits), shuffled and dealt to
// the planets in order. The deposit types come from a fixed list with exact
// counts of each type (see depositTypes), shuffled and dealt to the deposits
// in order. Deposit IDs run from 1 in planet order.
//
// GenerateDeposits panics unless there are exactly NumPlanets planets.
func GenerateDeposits(root *prng.Stream, planets []Planet) []Deposit {
	if len(planets) != NumPlanets {
		panic(fmt.Sprintf("game: GenerateDeposits needs %d planets, got %d", NumPlanets, len(planets)))
	}
	s := root.Derive(prng.TagDeposits)

	counts := make([]int, 0, NumPlanets)
	for n, planets := range planetsWithDeposits {
		for range planets {
			counts = append(counts, n)
		}
	}
	s.Shuffle(len(counts), func(i, j int) { counts[i], counts[j] = counts[j], counts[i] })

	types := make([]DepositType, 0, NumDeposits)
	for _, dt := range depositTypes {
		for range dt.count {
			types = append(types, dt.typ)
		}
	}
	s.Shuffle(len(types), func(i, j int) { types[i], types[j] = types[j], types[i] })

	deposits := make([]Deposit, 0, NumDeposits)
	for i, planet := range planets {
		for depositNo := 1; depositNo <= counts[i]; depositNo++ {
			deposits = append(deposits, Deposit{
				ID:        len(deposits) + 1,
				PlanetID:  planet.ID,
				DepositNo: depositNo,
				Type:      types[len(deposits)],
			})
		}
	}
	return deposits
}

// LoadDeposits reads deposits.json from dir.
func LoadDeposits(dir string) ([]Deposit, error) {
	var deposits []Deposit
	if err := loadJSON(dir, DepositsFileName, &deposits); err != nil {
		return nil, err
	}
	return deposits, nil
}

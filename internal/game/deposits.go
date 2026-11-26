// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import "fmt"

// DepositsFileName is the name of the file in a turn's folder that holds the
// planets' deposits.
const DepositsFileName = "deposits.json"

// MaxDepositsPerPlanet is the most deposits a planet can have.
const MaxDepositsPerPlanet = 31

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
	// Size is the amount of refined product left in the deposit. It loses
	// only what a mine refines and puts into storage.
	Size int `json:"size"`
	// Yield is the percentage that scales what a mine extracts in a turn: a
	// working MINE unit refines MineOrePerTL x TL x Yield% of product.
	Yield int `json:"yield"`
}

// LoadDeposits reads deposits.json from dir, a turn's folder.
func LoadDeposits(dir string) ([]Deposit, error) {
	var deposits []Deposit
	if err := loadJSON(dir, DepositsFileName, &deposits); err != nil {
		return nil, err
	}
	return deposits, nil
}

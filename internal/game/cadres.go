// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import "github.com/mdhender/ecvg/internal/units"

// TraineesPerPRO is the most TRNE that one PRO trains.
const TraineesPerPRO = 100

// Cadres counts the cadres an empire has drafted in an entity. A cadre is
// made up of population units that stay in the entity's population: they
// still eat and take up space. They are allocated by the empire, so they
// don't join the worker (WRKR) cadre.
type Cadres struct {
	// CWRK is construction workers: each is 1 PRO and 1 USK.
	CWRK int `json:"cwrk,omitzero"`
	// TRNE is trainees: each is 1 USK, and every TraineesPerPRO of them, or
	// part of that, needs 1 PRO.
	TRNE int `json:"trne,omitzero"`
	// SPY is spies: each is 1 PRO and 1 SLD.
	SPY int `json:"spy,omitzero"`
}

// PRO returns the PRO the cadres hold.
func (k Cadres) PRO() int {
	return k.CWRK + trainers(k.TRNE) + k.SPY
}

// USK returns the USK the cadres hold.
func (k Cadres) USK() int {
	return k.CWRK + k.TRNE
}

// SLD returns the SLD the cadres hold.
func (k Cadres) SLD() int {
	return k.SPY
}

// trainers returns the PRO that trne TRNE need, rounded up.
func trainers(trne int) int {
	return (trne + TraineesPerPRO - 1) / TraineesPerPRO
}

// Fit returns the cadres cut down to what the population p can make up, for
// when the population has shrunk. SPY keep their people first, then CWRK,
// then TRNE.
func (k Cadres) Fit(p Population) Cadres {
	pro, usk, sld := max(0, p.PRO), max(0, p.USK), max(0, p.SLD)
	k.SPY = max(0, min(k.SPY, pro, sld))
	pro -= k.SPY
	k.CWRK = max(0, min(k.CWRK, pro, usk))
	pro -= k.CWRK
	usk -= k.CWRK
	k.TRNE = max(0, min(k.TRNE, usk, pro*TraineesPerPRO))
	return k
}

// FreePRO, FreeUSK, and FreeSLD return the colony's population units of each
// type that no cadre holds.
func (c *Colony) FreePRO() int { return max(0, c.Population.PRO-c.Cadres.PRO()) }
func (c *Colony) FreeUSK() int { return max(0, c.Population.USK-c.Cadres.USK()) }
func (c *Colony) FreeSLD() int { return max(0, c.Population.SLD-c.Cadres.SLD()) }

// Draftable reports whether the unit with the given code can be drafted:
// SLD, CWRK, TRNE, and SPY so far.
func Draftable(code string) bool {
	switch code {
	case units.SLD, units.CWRK, units.TRNE, units.SPY:
		return true
	}
	return false
}

// Drafted records a draft order and what came of it.
type Drafted struct {
	Unit string `json:"unit"`
	// Asked is the quantity ordered and Drafted the quantity the entity
	// could supply.
	Asked   int `json:"asked"`
	Drafted int `json:"drafted"`
}

// Draft drafts up to qty of the unit with the given code from the colony's
// free population, and returns the quantity drafted: as many as it can.
//
//   - SLD: 1 USK each, which becomes an SLD.
//   - CWRK: 1 PRO and 1 USK each.
//   - TRNE: 1 USK each, and 1 PRO for every TraineesPerPRO in the cadre or
//     part of that, so 1 TRNE needs a whole PRO.
//   - SPY: 1 PRO and 1 SLD each.
//
// It drafts nothing of a unit that isn't Draftable.
func (c *Colony) Draft(code string, qty int) int {
	n := max(0, qty)
	switch code {
	case units.SLD:
		n = min(n, c.FreeUSK())
		c.Population.USK -= n
		c.Population.SLD += n
	case units.CWRK:
		n = min(n, c.FreePRO(), c.FreeUSK())
		c.Cadres.CWRK += n
	case units.TRNE:
		// The cadre can grow to what its trainers and the free PRO train.
		most := (trainers(c.Cadres.TRNE) + c.FreePRO()) * TraineesPerPRO
		n = max(0, min(n, c.FreeUSK(), most-c.Cadres.TRNE))
		c.Cadres.TRNE += n
	case units.SPY:
		n = min(n, c.FreePRO(), c.FreeSLD())
		c.Cadres.SPY += n
	default:
		return 0
	}
	return n
}

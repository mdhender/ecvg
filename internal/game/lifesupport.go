// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"math"

	"github.com/mdhender/ecvg/internal/units"
)

// MaxSolarLifeSupportOrbit is the outermost orbit in which the LFSU of an
// orbital colony run on solar power (manual G6.1).
const MaxSolarLifeSupportOrbit = 5

// LifeSupportDeathFactor is how hard a shortage of life support raises an
// entity's death rate: the rate is multiplied by 1 plus this times the
// fraction of the population that has no life support (manual G6.2).
const LifeSupportDeathFactor = 10

// LifeSupported returns the population units that qty LFSU of tech level tl
// support: TL^2 each (manual G6.1).
func LifeSupported(tl, qty int) int {
	return tl * tl * max(0, qty)
}

// LifeSupportFuel returns the FUEL that qty LFSU of tech level tl burn in a
// turn: 1 x TL each. See LifeSupportSolar for the ones that burn none.
func LifeSupportFuel(tl, qty int) int {
	return tl * max(0, qty)
}

// LifeSupportFuelled returns the most LFSU of tech level tl that fuel FUEL
// can run for a turn.
func LifeSupportFuelled(tl, fuel int) int {
	if tl < 1 {
		return 0
	}
	return max(0, fuel) / tl
}

// LifeSupportSolar reports whether the LFSU of an entity of type t in orbit
// orbitNo run without FUEL: the ones of an orbital colony in orbits
// 1..MaxSolarLifeSupportOrbit are solar powered.
func LifeSupportSolar(t EntityType, orbitNo int) bool {
	return t == CORB && 1 <= orbitNo && orbitNo <= MaxSolarLifeSupportOrbit
}

// LifeSupportCapacity returns the population units that the colony's
// assembled LFSU support when all of them run. LFSU in storage support
// nobody.
func (c *Colony) LifeSupportCapacity() int {
	n := 0
	for _, u := range c.Operational {
		if u.Type == units.LFSU {
			n += LifeSupported(u.TechLevel, u.Qty)
		}
	}
	return n
}

// LifeSupport is what a colony's life support did in a turn.
type LifeSupport struct {
	// Units is the number of assembled LFSU, Worked how many of them ran,
	// and Unfuelled how many had no FUEL to run on.
	Units     int `json:"units"`
	Worked    int `json:"worked"`
	Unfuelled int `json:"unfuelled,omitzero"`
	// Fuel is the FUEL the units burned.
	Fuel int `json:"fuel"`
	// Needed is the population units that needed life support, and Supported
	// the population units that the LFSU that ran support. The colony's
	// death rate goes up when Needed is the greater.
	Needed    int `json:"needed"`
	Supported int `json:"supported"`
	// Died is the population that died in the turn because the colony's
	// death rate was above normal.
	Died Population `json:"died,omitzero"`
}

// Short returns the population units that had no life support.
func (l LifeSupport) Short() int {
	return max(0, l.Needed-l.Supported)
}

// RunLifeSupport runs the colony's life support for a turn (manual G6.0),
// and returns what happened. The colony is in orbit orbitNo.
//
// An open-air colony needs no life support, and its LFSU don't run. In any
// other entity every assembled LFSU runs, whether the population needs it or
// not, in the order the units are listed; each burns LifeSupportFuel, from
// the cargo and then the depot, unless it is solar powered
// (LifeSupportSolar), and the ones the FUEL doesn't reach don't run.
//
// Then the death rate changes (manual G6.2). If the population needs more
// life support than the LFSU that ran supply, the rate becomes
// ((P x LifeSupportDeathFactor) + 1) x the rate, where P is the fraction of
// the population without life support; it is never more than 1. Otherwise
// it is halved, and never less than DefaultDeathRate.
func (c *Colony) RunLifeSupport(orbitNo int) LifeSupport {
	var l LifeSupport
	if c.Type.Enclosed() {
		l.Needed = c.Population.LifeSupport(c.Type)
		solar := LifeSupportSolar(c.Type, orbitNo)
		for _, u := range c.Operational {
			if u.Type != units.LFSU || u.Qty <= 0 {
				continue
			}
			n := u.Qty
			if !solar {
				n = min(n, LifeSupportFuelled(u.TechLevel, c.Stock(units.FUEL)))
				fuel := LifeSupportFuel(u.TechLevel, n)
				c.Spend(units.FUEL, fuel)
				l.Fuel += fuel
			}
			l.Units += u.Qty
			l.Worked += n
			l.Unfuelled += u.Qty - n
			l.Supported += LifeSupported(u.TechLevel, n)
		}
	}
	if short := l.Short(); short > 0 {
		p := float64(short) / float64(l.Needed)
		c.DeathRate = min(1, (p*LifeSupportDeathFactor+1)*max(c.DeathRate, 0))
	} else if c.DeathRate > DefaultDeathRate {
		c.DeathRate = max(DefaultDeathRate, c.DeathRate/2)
	}
	return l
}

// ExcessDeaths returns the population units of units that die in a turn at
// the death rate rate, over and above the deaths of the normal rate,
// DefaultDeathRate, which births make up for. The total is rounded up to a
// whole unit, and at a rate of 1 everybody dies.
func ExcessDeaths(units int, rate float64) int {
	switch {
	case units <= 0 || rate <= DefaultDeathRate:
		return 0
	case rate >= 1:
		return units
	}
	// The small allowance keeps the noise of the arithmetic from rounding a
	// whole number of deaths up to the next.
	return min(units, int(math.Ceil(float64(units)*(rate-DefaultDeathRate)-1e-9)))
}

// Deaths returns how deaths deaths are shared among the types of unit of the
// population p: in proportion to their numbers, each share rounded down, and
// the deaths left over taken first from UEM, then SLD, then USK, then PRO.
func (p Population) Deaths(deaths int) Population {
	units := p.Units()
	deaths = max(0, min(deaths, units))
	if units == 0 || deaths == 0 {
		return Population{}
	}
	var s Population
	left := deaths
	for _, t := range []struct{ dead, alive *int }{
		{&s.UEM, &p.UEM}, {&s.USK, &p.USK}, {&s.PRO, &p.PRO}, {&s.SLD, &p.SLD},
	} {
		*t.dead = *t.alive * deaths / units
		left -= *t.dead
	}
	for _, t := range []struct{ dead, alive *int }{
		{&s.UEM, &p.UEM}, {&s.SLD, &p.SLD}, {&s.USK, &p.USK}, {&s.PRO, &p.PRO},
	} {
		n := min(left, *t.alive-*t.dead)
		*t.dead += n
		left -= n
	}
	return s
}

// Less returns the population p without the units of q.
func (p Population) Less(q Population) Population {
	return Population{UEM: p.UEM - q.UEM, USK: p.USK - q.USK, PRO: p.PRO - q.PRO, SLD: p.SLD - q.SLD}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import "github.com/mdhender/ecvg/internal/game"

// LifeSupport is the life support phase of the turn numbered turn, the last
// steps of the manual's POP increase stage (B3.20.3, B3.20.5, B3.20.6). It
// comes after Demographics. For each colony in d, in this order:
//
//  1. Deaths. If the colony's death rate is above game.DefaultDeathRate,
//     which the lack of life support in an earlier turn made it, part of its
//     population dies (game.ExcessDeaths, shared by game.Population.Deaths),
//     and its cadres are cut down to the people left.
//  2. The LFSU run, burning FUEL that production left, and the death rate
//     changes: up if the population has more units than the LFSU that ran
//     support, and otherwise halfway back to normal
//     (game.Colony.RunLifeSupport).
//
// So the population dies the turn after its life support falls short. The
// colony's LifeSupport records the turn. Nothing happens in turn SetupTurn.
//
// Births, and the deaths of the normal death rate, are not built: they
// cancel out at the default rates, so only the deaths of the rate above
// normal are taken.
func LifeSupport(d *game.Data, turn int) {
	orbits := map[int]int{} // orbit number by orbit ID
	for _, o := range d.Orbits {
		orbits[o.ID] = o.OrbitNo
	}
	orbitNo := map[int]int{} // by planet ID
	for _, p := range d.Planets {
		orbitNo[p.ID] = orbits[p.OrbitID]
	}
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.LifeSupport = game.LifeSupport{}
		if turn <= SetupTurn {
			continue
		}
		died := c.Population.Deaths(game.ExcessDeaths(c.Population.Units(), c.DeathRate))
		c.Population = c.Population.Less(died)
		c.Cadres = c.Cadres.Fit(c.Population)
		c.LifeSupport = c.RunLifeSupport(orbitNo[c.PlanetID])
		c.LifeSupport.Died = died
	}
}

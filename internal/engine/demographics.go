// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package engine processes a game's turns. A turn is processed in phases, each
// of which changes the game's data in memory.
package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// TurnsPerYear is the number of turns in a game year. A turn whose number
// divides by it is an end-of-year turn.
const TurnsPerYear = 4

// SetupTurn is the number of a game's first turn, the players' setup turn.
const SetupTurn = 0

// RunTurn processes the game's current turn, d.Game.Turn: it runs each phase
// in order on the state in d, which becomes the state as of the next turn, and
// then advances d.Game.Turn. The phases, in order: ApplyOrders, with the
// orders the empires returned for the turn, AllocateResources, Farming, Mining, Assembly, Draft, Rations,
// PopulationChanges, and Demographics. It returns what became of the orders. If it fails, d is not
// changed.
func RunTurn(d *game.Data, files []orders.File) (OrdersResult, error) {
	result, err := ApplyOrders(d, files)
	if err != nil {
		return OrdersResult{}, err
	}
	allocs := AllocateResources(d, d.Game.Turn)
	Farming(d, d.Game.Turn, allocs)
	Mining(d, d.Game.Turn, allocs)
	Assembly(d)
	Draft(d, result.Drafts)
	meals := Rations(d, d.Game.Turn)
	PopulationChanges(d, meals)
	Demographics(d, d.Game.Turn)
	d.Game.Turn++
	return result, nil
}

// Demographics is the demographic changes phase of the turn numbered turn. It
// applies ChangePopulation to the population of every entity in d, promotes
// the trainees of the TRNE cadre the turn began with (game.Colony.Train; not
// in the setup turn), and then cuts the entity's cadres down to what its population can still make up.
func Demographics(d *game.Data, turn int) {
	for i := range d.Colonies {
		c := &d.Colonies[i]
		c.Population = ChangePopulation(c.Population, turn)
		// The trainees the turn began with are promoted and their cadre is
		// disbanded. The ones drafted this turn train in the next.
		c.Trained = game.Trained{}
		if turn > SetupTurn {
			c.Trained = c.Train()
		}
		c.Cadres.NewTRNE = 0
		// A cadre can't hold people the colony no longer has.
		c.Cadres = c.Cadres.Fit(c.Population)
	}
}

// ChangePopulation returns the population p after the demographic changes of
// the turn numbered turn, made in this order:
//
//  1. If UEM is more than 30% of the population, 2% of UEM (rounded up to a
//     whole unit) become USK.
//  2. If the prior turn, turn-1, was an end-of-year turn (it is greater than 0
//     and divides by TurnsPerYear), 5% of SLD (rounded up) become PRO.
//
// The total population does not change. Nothing changes in turn SetupTurn.
func ChangePopulation(p game.Population, turn int) game.Population {
	if turn <= SetupTurn {
		return p
	}
	if p.UEM*10 > p.Units()*3 {
		n := percentUp(p.UEM, 2)
		p.UEM -= n
		p.USK += n
	}
	if prior := turn - 1; prior > 0 && prior%TurnsPerYear == 0 {
		n := percentUp(p.SLD, 5)
		p.SLD -= n
		p.PRO += n
	}
	return p
}

// percentUp returns pct percent of n, rounded up to a whole number.
func percentUp(n, pct int) int {
	return (n*pct + 99) / 100
}

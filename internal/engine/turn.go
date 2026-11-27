// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"log"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// A Stage is one stage of the manual's turn sequence (B3.0).
type Stage struct {
	No   int    // the stage's number in the manual: B3.No
	Name string // the stage's name in the manual

	// run does the stage's work. It is nil for a stage that isn't built.
	run func(t *turn)
}

// Implemented reports whether the stage does anything yet.
func (s Stage) Implemented() bool {
	return s.run != nil
}

// turn is what the stages of one turn share.
type turn struct {
	d      *game.Data
	no     int          // the number of the turn being processed
	orders OrdersResult // what became of the orders, from ApplyOrders
}

// stages is the turn sequence, in the manual's order (B3.1 to B3.22).
var stages = []Stage{
	{No: 1, Name: "Combat"},
	{No: 2, Name: "Pre Set-Up"},
	{No: 3, Name: "Set-Up", run: func(t *turn) { Setups(t.d) }},
	{No: 4, Name: "Dis-Assembly", run: func(t *turn) { Disassembly(t.d) }},
	{No: 5, Name: "Pre-Transfer"},
	{No: 6, Name: "Transfer"},
	{No: 7, Name: "Build Change"},
	{No: 8, Name: "Assembly", run: func(t *turn) { Assembly(t.d) }},
	{No: 9, Name: "Market"},
	{No: 10, Name: "Survey"},
	{No: 11, Name: "Spy Move"},
	{No: 12, Name: "Draft", run: func(t *turn) { Draft(t.d, t.orders.Drafts) }},
	{No: 13, Name: "Pay Change"},
	{No: 14, Name: "Naming"},
	{No: 15, Name: "Ship Movement"},
	{No: 16, Name: "Restart"},
	{No: 17, Name: "Production", run: production},
	{No: 18, Name: "Espionage"},
	{No: 19, Name: "Rebellion"},
	{No: 20, Name: "POP Increase", run: popIncrease},
	{No: 21, Name: "News Release"},
	{No: 22, Name: "Probe"},
}

// Stages returns the turn sequence, in the order RunTurn runs it.
func Stages() []Stage {
	return append([]Stage(nil), stages...)
}

// RunTurn processes the game's current turn, d.Game.Turn, on the state in d,
// which becomes the state as of the next turn, and then advances d.Game.Turn.
//
// First comes the orders phase, ApplyOrders, with the orders the empires
// returned for the turn: every order is checked, and its work started, before
// any stage runs. Then the stages of the manual's turn sequence (B3.0) run in
// order, each for every empire before the next begins. A stage that isn't
// built logs "stage n: name: not implemented" and does nothing.
//
// It returns what became of the orders. If it fails, d is not changed.
func RunTurn(d *game.Data, files []orders.File) (OrdersResult, error) {
	result, err := ApplyOrders(d, files)
	if err != nil {
		return OrdersResult{}, err
	}
	t := &turn{d: d, no: d.Game.Turn, orders: result}
	for _, s := range stages {
		if !s.Implemented() {
			log.Printf("stage %d: %s: not implemented", s.No, s.Name)
			continue
		}
		s.run(t)
	}
	d.Game.Turn++
	return result, nil
}

// production is the Production stage (B3.17): AllocateResources, Farming,
// Rations, PopulationChanges (the deaths by starvation), Mining, and Research.
func production(t *turn) {
	allocs := AllocateResources(t.d, t.no)
	Farming(t.d, t.no, allocs)
	meals := Rations(t.d, t.no)
	PopulationChanges(t.d, meals)
	Mining(t.d, t.no, allocs)
	Research(t.d, t.no, allocs)
}

// popIncrease is the POP Increase stage (B3.20): Demographics, then
// LifeSupport.
func popIncrease(t *turn) {
	Demographics(t.d, t.no)
	LifeSupport(t.d, t.no)
}

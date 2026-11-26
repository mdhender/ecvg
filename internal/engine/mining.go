// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Mining is the mining production phase of the turn numbered turn. In every
// colony in d, the MINE units in its mining groups that were allocated staff
// and fuel burn the fuel and extract refined product from the group's deposit
// (game.MineOutput, with the deposit's yield): GOLD, FUEL, MTL, or NMTL, as
// the deposit is. The product goes into the colony's depot, or its cargo if
// it has no depot, and comes out of the deposit's size; a deposit can't give
// more than it has left. Product that the cargo has no room for is returned
// to the deposit: the empire pays to extract it and gains nothing. Each group's Mined records what it did. Allocs is
// what AllocateResources returned for the turn. Nothing is mined in turn
// SetupTurn.
//
// A group whose deposit is missing or exhausted still burns its fuel.
func Mining(d *game.Data, turn int, allocs []Allocation) {
	deposits := map[int]*game.Deposit{}
	for i := range d.Deposits {
		deposits[d.Deposits[i].ID] = &d.Deposits[i]
	}
	for i := range d.Colonies {
		c := &d.Colonies[i]
		for j := range c.MiningGroups {
			g := &c.MiningGroups[j]
			g.Mined = game.Mined{}
			if turn <= SetupTurn || i >= len(allocs) || j >= len(allocs[i].Mines) {
				continue
			}
			dep := deposits[g.DepositID]
			product := 0
			for k, u := range g.Units {
				if k >= len(allocs[i].Mines[j]) {
					break
				}
				w := allocs[i].Mines[j][k]
				g.Mined.Worked += w.Worked
				g.Mined.Fuel += w.Fuel
				if dep != nil {
					product += game.MineOutput(u.TechLevel, w.Worked, dep.Yield)
				}
			}
			c.Spend(units.FUEL, g.Mined.Fuel)
			if dep == nil {
				continue
			}
			// The deposit can't give more than it has, and it loses only what
			// the colony has room to store: the rest is returned to it.
			product = min(product, max(0, dep.Size))
			g.Mined.Product = c.StoreResource(string(dep.Type), product)
			g.Mined.Returned = product - g.Mined.Product
			dep.Size -= g.Mined.Product
		}
	}
}

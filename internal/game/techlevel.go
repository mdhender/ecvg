// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/units"
)

// WorkTechLevel is the work that raises an entity's tech level. Its Unit has
// no type: its TechLevel is the level the entity is raised to, and its Qty is
// 1 until the work is done.
const WorkTechLevel Work = "tech-level"

const (
	// ErrTechLevelNotHigher is the error for an order that would not raise
	// the entity's tech level.
	ErrTechLevelNotHigher = cerrs.Error("the tech level asked for is not higher than the entity's")
	// ErrTechLevelTooHigh is the error for an order to go past
	// units.MaxTechLevel.
	ErrTechLevelTooHigh = cerrs.Error("there is no tech level that high")
	// ErrTechLevelInProgress is the error for an order given while the
	// entity's CWRK are still raising its tech level.
	ErrTechLevelInProgress = cerrs.Error("the entity is already raising its tech level")
	// ErrNotEnoughResearch is the error for an order that the entity's RSCH,
	// or the PRTO units the order names, don't pay for in full.
	ErrNotEnoughResearch = cerrs.Error("not enough research to pay for every tech level asked for")
)

// TechLevelCost returns the research points it takes to raise an entity from
// tech level tl-1 to tl: 100,000 for level 2, and twice as many for each
// level after it (manual J4.1). It is 0 for a level that can't be reached.
func TechLevelCost(tl int) int {
	if tl < units.MinTechLevel+1 || tl > units.MaxTechLevel {
		return 0
	}
	return 100_000 << (tl - 2)
}

// TechLevelsCost returns the research points it takes to raise an entity
// from tech level from to tech level to, one level at a time.
func TechLevelsCost(from, to int) int {
	cost := 0
	for tl := from + 1; tl <= to; tl++ {
		cost += TechLevelCost(tl)
	}
	return cost
}

// PrototypeResearch returns the research points that a PRTO unit of tech
// level tl holds: 50,000 for a PRTO-1, and twice as many for each level
// after it, so that a PRTO-n holds exactly TechLevelCost(n) for n of 2 or
// more.
func PrototypeResearch(tl int) int {
	if tl < units.MinTechLevel || tl > units.MaxTechLevel {
		return 0
	}
	return 50_000 << (tl - 1)
}

// RaisingTechLevel returns the tech level that the colony's CWRK are working
// to raise it to, or 0 if they aren't.
func (c *Colony) RaisingTechLevel() int {
	for _, a := range c.Assembly {
		if a.Action == WorkTechLevel && a.Unit.Qty > 0 {
			return a.Unit.TechLevel
		}
	}
	return 0
}

// StartTechLevel starts the work that raises the colony's tech level to the
// level to, or by one level if to is 0. The research every level costs
// (TechLevelsCost) is paid at once and in full: from the colony's RSCH, or,
// if prtoQty is not 0, from up to prtoQty of its PRTO units of tech level
// prtoTL, each of which holds PrototypeResearch points. PRTO units are used
// up whole, as few of them as pay the cost. What is left over in them is
// stored as RSCH if the colony has assembled laboratories (HasLabs), and is
// lost if it has none. The colony's tech level changes only when the work is
// done.
//
// The work is the colony's enclosed space (its Capacity, rounded up), taken
// as EMU, for each level, and its CWRK do it in the assembly phase like any
// other assembly work; maxCWRK caps the CWRK that work on it in a turn, if it
// is not 0.
//
// Nothing changes if it fails: ErrTechLevelInProgress, ErrTechLevelNotHigher,
// ErrTechLevelTooHigh, or ErrNotEnoughResearch.
func (c *Colony) StartTechLevel(to, prtoTL, prtoQty, maxCWRK int) error {
	if tl := c.RaisingTechLevel(); tl != 0 {
		return fmt.Errorf("to TL-%d: %w", tl, ErrTechLevelInProgress)
	}
	if to == 0 {
		to = c.TechLevel + 1
	}
	switch {
	case to > units.MaxTechLevel:
		return fmt.Errorf("TL-%d: %w", to, ErrTechLevelTooHigh)
	case to <= c.TechLevel:
		return fmt.Errorf("TL-%d, and the entity is TL-%d: %w", to, c.TechLevel, ErrTechLevelNotHigher)
	}
	cost := TechLevelsCost(c.TechLevel, to)
	if prtoQty == 0 {
		if have := c.Stock(units.RSCH); have < cost {
			return fmt.Errorf("TL-%d takes %d RSCH and the entity has %d: %w", to, cost, have, ErrNotEnoughResearch)
		}
		c.Spend(units.RSCH, cost)
	} else {
		each := PrototypeResearch(prtoTL)
		have := max(0, min(prtoQty, c.Stored(units.PRTO, prtoTL)-c.Claimed(units.PRTO, prtoTL)))
		if each <= 0 || have < (cost+each-1)/each {
			return fmt.Errorf("TL-%d takes %d RSCH and %d PRTO-%d hold %d: %w", to, cost, have, prtoTL, have*each, ErrNotEnoughResearch)
		}
		n := (cost + each - 1) / each
		c.takeStored(units.PRTO, prtoTL, n)
		if left := n*each - cost; left > 0 && c.HasLabs() {
			c.AddResearch(left)
		}
	}
	c.Assembly = append(c.Assembly, Assembly{
		Action:  WorkTechLevel,
		Unit:    Unit{TechLevel: to, Qty: 1},
		Need:    c.Capacity().Ceil() * (to - c.TechLevel),
		MaxCWRK: maxCWRK,
	})
	return nil
}

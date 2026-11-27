// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"math"
	"slices"
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// EMUPerCWRK is the work one CWRK does in a turn, in EMU (equivalent mass
// units): it assembles up to that many MU.
const EMUPerCWRK = 500

// Work is what a piece of assembly work does with its units.
type Work string

const (
	// WorkAssemble moves units from storage into use. It is the zero value.
	WorkAssemble Work = ""
	// WorkDisassemble moves assembled units back into storage.
	WorkDisassemble Work = "disassemble"
	// WorkScrap breaks units in storage up for their MTL and NMTL.
	WorkScrap Work = "scrap"
)

// Verb returns the word that orders use for the work.
func (w Work) Verb() string {
	if w == WorkAssemble {
		return "assemble"
	}
	return string(w)
}

// ScrapPercent is the percentage of the MTL and NMTL that went into a unit
// that scrapping it recovers, rounded down.
const ScrapPercent = 95

// Assembly is a piece of work for a colony's CWRK: assembling units, into one
// of the colony's groups or on their own, disassembling them, or scrapping
// them. An order starts it, and it lasts until it has dealt with every unit.
type Assembly struct {
	// Action is what the work does: assemble, if it is empty.
	Action Work `json:"action,omitzero"`
	// Kind and GroupNo are the group that assembled units join, if any.
	Kind    GroupKind `json:"kind,omitzero"`
	GroupNo int       `json:"group_no,omitzero"`
	// Unit is the units still to do. They stay where they are until then.
	Unit Unit `json:"unit"`
	// EMU is the work done toward them. A unit is done when the work reaches
	// its mass, so a partly done unit keeps its EMU between turns.
	EMU int `json:"emu"`
	// MaxCWRK is the most CWRK the order lets work on it in a turn, or 0 for
	// as many as it can use. SharesCap says that the work is of the same
	// order as the work before it, and the cap is on them together.
	MaxCWRK   int  `json:"max_cwrk,omitzero"`
	SharesCap bool `json:"shares_cap,omitzero"`
	// Assembled is the number of units done in the prior turn, and Waiting
	// says why units that had the work done weren't: "room" or "units".
	Assembled int    `json:"assembled,omitzero"`
	Waiting   string `json:"waiting,omitzero"`
	// MTL and NMTL are what scrapping recovered in the prior turn, and Lost
	// is the part of it that the colony had no room to store.
	MTL  int `json:"mtl,omitzero"`
	NMTL int `json:"nmtl,omitzero"`
	Lost int `json:"lost,omitzero"`
}

// Construction is what a colony's CWRK did in a turn.
type Construction struct {
	// Pool is the EMU the CWRK could do, EMUPerCWRK each, and Used the EMU
	// that went into assembly. The rest is lost.
	Pool int `json:"pool"`
	Used int `json:"used"`
}

// Stored returns the quantity of the unit with the given code and tech level
// in the colony's storage: its cargo and its depot together.
func (c *Colony) Stored(code string, tl int) int {
	return count(code, tl, c.Cargo, c.Depot)
}

// InUse returns the quantity of the unit with the given code and tech level
// that is assembled, in the colony or in its depot.
func (c *Colony) InUse(code string, tl int) int {
	return count(code, tl, c.Operational, c.DepotOperational)
}

func count(code string, tl int, lists ...[]Unit) int {
	n := 0
	for _, list := range lists {
		for _, u := range list {
			if u.Type == code && u.TechLevel == tl {
				n += u.Qty
			}
		}
	}
	return n
}

// Claimed returns the quantity of the unit with the given code and tech level
// in storage that the colony's work is still to assemble or scrap.
func (c *Colony) Claimed(code string, tl int) int {
	return c.BeingAssembled(code, tl) + c.working(WorkScrap, code, tl)
}

// BeingAssembled returns the quantity of the unit that the colony's work is
// still to assemble.
func (c *Colony) BeingAssembled(code string, tl int) int {
	return c.working(WorkAssemble, code, tl)
}

// BeingDisassembled returns the quantity of the unit that the colony's work
// is still to disassemble.
func (c *Colony) BeingDisassembled(code string, tl int) int {
	return c.working(WorkDisassemble, code, tl)
}

func (c *Colony) working(w Work, code string, tl int) int {
	n := 0
	for _, a := range c.Assembly {
		if a.Action == w && a.Unit.Type == code && a.Unit.TechLevel == tl {
			n += a.Unit.Qty
		}
	}
	return n
}

// Assemble does a turn of the colony's assembly work. The colony's CWRK make
// a pool of EMUPerCWRK EMU each, and each piece of work, oldest first, takes
// as much of it as it can use, or as much as its order's cap on CWRK allows.
// EMU nothing took is lost. Work that finished in the prior turn is dropped
// first. What happens to the units whose mass the work has reached depends on
// the work:
//
//   - Assembled units move from storage into use, in the depot if the colony
//     has one and the unit can be there, and otherwise in the colony itself,
//     and join their group if they have one.
//   - Disassembled units move out of use and back into storage: from the
//     depot into the depot, and from the colony into its cargo.
//   - Scrapped units leave storage, and ScrapPercent percent of the MTL and
//     NMTL that built them is stored; what there is no room for is lost.
//
// Units move only if the colony has room for them where they are going; the
// rest wait.
func (c *Colony) Assemble() {
	jobs := c.Assembly[:0]
	// When the first work of an order is dropped, the next of the order
	// stands in for it.
	promote := false
	for _, a := range c.Assembly {
		if a.Unit.Qty <= 0 {
			promote = promote || !a.SharesCap
			continue
		}
		if promote {
			a.SharesCap = false
		}
		promote = false
		a.Assembled, a.Waiting, a.MTL, a.NMTL, a.Lost = 0, "", 0, 0, 0
		jobs = append(jobs, a)
	}
	c.Assembly = jobs
	pool := max(0, c.Cadres.CWRK) * EMUPerCWRK
	left := pool
	budget := 0 // what the order's cap leaves of the pool
	for i := range c.Assembly {
		a := &c.Assembly[i]
		if !a.SharesCap {
			budget = math.MaxInt
			if a.MaxCWRK > 0 {
				budget = a.MaxCWRK * EMUPerCWRK
			}
		}
		d, ok := units.Lookup(a.Unit.Type)
		if !ok {
			continue
		}
		tl := a.Unit.TechLevel
		take := max(0, min(left, budget, d.Mass(tl, a.Unit.Qty)-a.EMU))
		a.EMU += take
		left -= take
		budget -= take
		// The most units whose mass the work has reached.
		n := sort.Search(a.Unit.Qty, func(k int) bool { return d.Mass(tl, k+1) > a.EMU })
		have := c.Stored(a.Unit.Type, tl)
		if a.Action == WorkDisassemble {
			have = c.InUse(a.Unit.Type, tl)
		}
		if have < n {
			n, a.Waiting = have, "units"
		}
		if fit := c.fit(a.Action, d, tl, n); fit < n {
			n, a.Waiting = fit, "room"
		}
		if n <= 0 {
			continue
		}
		switch a.Action {
		case WorkDisassemble:
			c.dismantle(d, tl, n)
		case WorkScrap:
			a.MTL, a.NMTL, a.Lost = c.scrap(d, tl, n)
		default:
			c.install(d, tl, n)
			c.addToGroup(a.Kind, a.GroupNo, tl, n)
		}
		a.EMU -= d.Mass(tl, n)
		a.Unit.Qty -= n
		a.Assembled = n
	}
	c.Construction = Construction{Pool: pool, Used: pool - left}
}

// fit returns the most of n units that the colony has room to assemble or
// disassemble. They fit if the colony is within its capacity afterwards, or
// no further over it than it was. Both ways the space can go either way:
// most units are bigger assembled, and structure takes space only when it is
// stowed and makes space only when it is assembled.
func (c *Colony) fit(w Work, d units.Def, tl, n int) int {
	if w == WorkScrap || (w == WorkAssemble && c.HasDepot() && d.DepotAllowed()) {
		return n
	}
	before := c.SpaceFree()
	return sort.Search(n, func(k int) bool {
		t := *c
		t.Operational, t.Cargo = slices.Clone(c.Operational), slices.Clone(c.Cargo)
		t.DepotOperational, t.Depot = slices.Clone(c.DepotOperational), slices.Clone(c.Depot)
		if w == WorkDisassemble {
			t.dismantle(d, tl, k+1)
		} else {
			t.install(d, tl, k+1)
		}
		after := t.SpaceFree()
		return after < 0 && after < before
	})
}

// install moves qty units from storage into use: in the depot, if the colony
// has one and the unit can be there, and otherwise in the colony itself.
func (c *Colony) install(d units.Def, tl, qty int) {
	c.takeStored(d.Code, tl, qty)
	if c.HasDepot() && d.DepotAllowed() {
		c.DepotOperational = addUnits(c.DepotOperational, d.Code, tl, qty)
	} else {
		c.Operational = addUnits(c.Operational, d.Code, tl, qty)
	}
}

// dismantle moves qty units out of use and into storage: the ones in use in
// the depot first, which go into the depot, and then the ones in the colony,
// which go into its cargo.
func (c *Colony) dismantle(d units.Def, tl, qty int) {
	n := takeUnits(c.DepotOperational, d.Code, tl, qty)
	if n > 0 {
		c.Depot = addUnits(c.Depot, d.Code, tl, n)
	}
	if n = takeUnits(c.Operational, d.Code, tl, qty-n); n > 0 {
		c.Cargo = addUnits(c.Cargo, d.Code, tl, n)
	}
}

// scrap removes qty units from storage and stores ScrapPercent percent of the
// MTL and NMTL that built them, each total rounded down. It returns what was
// recovered, and how much of that the colony had no room for.
func (c *Colony) scrap(d units.Def, tl, qty int) (mtl, nmtl, lost int) {
	c.takeStored(d.Code, tl, qty)
	perLotMTL, perLotNMTL, _ := d.Inputs(tl, 1)
	div := 100 * d.LotSize(tl)
	mtl, nmtl = qty*perLotMTL*ScrapPercent/div, qty*perLotNMTL*ScrapPercent/div
	lost = mtl - c.StoreResource(units.MTL, mtl)
	lost += nmtl - c.StoreResource(units.NMTL, nmtl)
	return mtl, nmtl, lost
}

// takeStored removes qty of a unit from the colony's storage, cargo first.
func (c *Colony) takeStored(code string, tl, qty int) {
	qty -= takeUnits(c.Cargo, code, tl, qty)
	takeUnits(c.Depot, code, tl, qty)
}

// takeUnits removes up to qty of a unit from list and returns the number
// removed.
func takeUnits(list []Unit, code string, tl, qty int) int {
	taken := 0
	for i := range list {
		if u := &list[i]; u.Type == code && u.TechLevel == tl {
			n := max(0, min(qty-taken, u.Qty))
			u.Qty -= n
			taken += n
		}
	}
	return taken
}

// addUnits adds qty of a unit to list.
func addUnits(list []Unit, code string, tl, qty int) []Unit {
	for i := range list {
		if u := &list[i]; u.Type == code && u.TechLevel == tl {
			u.Qty += qty
			return list
		}
	}
	return append(list, Unit{Type: code, TechLevel: tl, Qty: qty})
}

// GroupUnitsAt returns the entry of us for the tech level tl, adding an empty
// one if there is none, and the list.
func GroupUnitsAt(us []GroupUnits, tl int) (*GroupUnits, []GroupUnits) {
	for i := range us {
		if us[i].TechLevel == tl {
			return &us[i], us
		}
	}
	us = append(us, GroupUnits{TechLevel: tl})
	return &us[len(us)-1], us
}

// addToGroup adds qty units of tech level tl to the colony's group of the
// given kind numbered no.
func (c *Colony) addToGroup(kind GroupKind, no, tl, qty int) {
	add := func(us *[]GroupUnits) {
		var e *GroupUnits
		e, *us = GroupUnitsAt(*us, tl)
		e.Qty += qty
	}
	switch kind {
	case KindFarming:
		for i := range c.FarmingGroups {
			if c.FarmingGroups[i].No == no {
				add(&c.FarmingGroups[i].Units)
			}
		}
	case KindFactory:
		for i := range c.FactoryGroups {
			if c.FactoryGroups[i].No == no {
				add(&c.FactoryGroups[i].Units)
			}
		}
	case KindMining:
		for i := range c.MiningGroups {
			if c.MiningGroups[i].No == no {
				add(&c.MiningGroups[i].Units)
			}
		}
	}
}

// NextGroupNo gives out the number for a new group of the colony: one more
// than the highest that any of its groups, of any kind, has had.
func (c *Colony) NextGroupNo() int {
	last := c.LastGroupNo
	for _, g := range c.FarmingGroups {
		last = max(last, g.No)
	}
	for _, g := range c.MiningGroups {
		last = max(last, g.No)
	}
	for _, g := range c.FactoryGroups {
		last = max(last, g.No)
	}
	c.LastGroupNo = last + 1
	return c.LastGroupNo
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"sort"

	"github.com/mdhender/ecvg/internal/units"
)

// MaxShipsPerLocation is the most ships an empire may have set up at one
// location (manual F4.3.1).
const MaxShipsPerLocation = 99

// SetupShuttleMU is the transport capacity that a CWRK of the establishing
// entity takes for each turn it works on a set up: twice its mass, for the
// trips to the job site and back.
const SetupShuttleMU = 2 * 2 * PopulationUnitMass

// Setup is a set up that a colony is doing: a new entity that it is building
// (manual F4.0). The colony still owns everything that goes into it, and its
// people still live in the colony, until the work is done.
type Setup struct {
	// Entity is the new entity as it will be when the work is done. It has
	// its ID from the start.
	Entity Colony `json:"entity"`
	// BOM is the bill of material: what the colony has reserved for the
	// entity, and hands over when the work is done. People are listed by
	// their unit (UEM, USK, PRO, SLD) or their cadre (CWRK, SPY).
	BOM []Unit `json:"bom"`
	// Need is the EMU of assembly work that the entity takes, and EMU the
	// work done so far.
	Need int `json:"need"`
	EMU  int `json:"emu"`
	// MaxCWRK is the most CWRK of the colony that may be shuttled to the job
	// in a turn, or 0 for as many as it can use. Set ups with the same Batch,
	// if it isn't 0, share that crew: they are built one after another.
	MaxCWRK int `json:"max_cwrk,omitzero"`
	Batch   int `json:"batch,omitzero"`
	// Crew is the CWRK that were shuttled to the job in the prior turn, and
	// Waiting says why the job isn't done: "work" or "population".
	Crew    int    `json:"crew,omitzero"`
	Waiting string `json:"waiting,omitzero"`
}

// SetupItem is one kind of unit that a set up installs in the new entity.
type SetupItem struct {
	// Unit is the units to install. Its Type may be a population unit (UEM,
	// USK, PRO, SLD) or a cadre (CWRK, SPY), which takes its people along.
	Unit Unit
	// Stowed is set if units that can be assembled are to go into the new
	// entity's storage instead.
	Stowed bool
	// Kind and GroupNo are the group of the new entity that the assembled
	// units join, if any.
	Kind    GroupKind
	GroupNo int
}

// Reserved returns the quantity of the unit with the given code and tech
// level that the colony's set ups have reserved. For people, the code is the
// population unit or the cadre.
func (c *Colony) Reserved(code string, tl int) int {
	n := 0
	for _, s := range c.Setups {
		n += count(code, tl, s.BOM)
	}
	return n
}

// FreeCWRK and FreeSPY return the colony's cadres that no set up has
// reserved.
func (c *Colony) FreeCWRK() int { return max(0, c.Cadres.CWRK-c.Reserved(units.CWRK, 0)) }
func (c *Colony) FreeSPY() int  { return max(0, c.Cadres.SPY-c.Reserved(units.SPY, 0)) }

// PlanSetup adds the units of one line of a setup order to the set up s,
// which must be one of the colony's own, and returns how many it added. It
// reserves them in the colony (see Reserved) and puts them in s.Entity as
// they will be when the work is done. It takes what there is (manual F4.0):
//
//   - What the colony has: units in its storage that nothing has claimed or
//     reserved, never assembled ones; or people that no cadre holds and no
//     set up has reserved; or, for CWRK and SPY, the cadre, whose members go
//     along (a CWRK is a PRO and a USK, a SPY a PRO and an SLD).
//   - The room in the new entity (manual F3.2.1), if it is enclosed: units
//     take their assembled volume if they are assembled there and their
//     stowed volume if not, and people 1 VU each. Assembled structure takes
//     none, and makes the room for what is listed after it.
//
// A unit that requires assembly is assembled in the new entity, and joins
// the item's group, unless the item is stowed. Structure, life support, and
// the entity's first SDRV go up with the entity; the rest takes its mass in
// EMU of work, which is added to s.Need. Nothing goes to a depot: everything
// is in use in the entity or in its cargo.
func (c *Colony) PlanSetup(s *Setup, it SetupItem) int {
	code, tl, n := it.Unit.Type, it.Unit.TechLevel, max(0, it.Unit.Qty)
	d, _ := units.Lookup(code)
	to := &s.Entity
	if d.Population() {
		var p Population
		switch code {
		case units.UEM:
			p.UEM, n = 1, min(n, c.Population.UEM-c.Reserved(units.UEM, 0))
		case units.USK:
			p.USK, n = 1, min(n, c.FreeUSK())
		case units.PRO:
			p.PRO, n = 1, min(n, c.FreePRO())
		case units.SLD:
			p.SLD, n = 1, min(n, c.FreeSLD())
		case units.CWRK:
			p.PRO, p.USK, n = 1, 1, min(n, c.FreeCWRK())
		case units.SPY:
			p.PRO, p.SLD, n = 1, 1, min(n, c.FreeSPY())
		default:
			return 0
		}
		if to.Type.Enclosed() {
			n = min(n, max(0, to.SpaceFree().Floor())/p.Volume())
		}
		if n = max(0, n); n == 0 {
			return 0
		}
		to.Population = to.Population.Less(Population{UEM: -p.UEM * n, USK: -p.USK * n, PRO: -p.PRO * n, SLD: -p.SLD * n})
		switch code {
		case units.CWRK:
			to.Cadres.CWRK += n
		case units.SPY:
			to.Cadres.SPY += n
		}
		s.BOM = addUnits(s.BOM, code, 0, n)
		return n
	}
	n = min(n, c.Stored(code, tl)-c.Claimed(code, tl))
	assemble := d.RequiresAssembly && !it.Stowed
	switch {
	case assemble && (d.Encloses() > 0 || !to.Type.Enclosed()):
		// Structure needs no room, and neither does anything in the open.
	case assemble:
		free := to.SpaceFree()
		n = sort.Search(max(0, n), func(k int) bool { return d.Volume(tl, k+1).Cmp(free) > 0 })
	default:
		n = to.Room(code, tl, n)
	}
	if n = max(0, n); n == 0 {
		return 0
	}
	if assemble {
		// What goes up with the entity takes no work.
		work := n
		switch {
		case d.Encloses() > 0 || code == units.LFSU:
			work = 0
		case code == units.SDRV && !inUse(to, units.SDRV):
			work = n - 1
		}
		s.Need += workEMU(WorkAssemble, d, tl, work)
		to.Operational = addUnits(to.Operational, code, tl, n)
		to.addToGroup(it.Kind, it.GroupNo, tl, n)
	} else {
		to.Cargo = addUnits(to.Cargo, code, tl, n)
	}
	s.BOM = addUnits(s.BOM, code, tl, n)
	return n
}

// inUse reports whether c has any assembled unit with the given code.
func inUse(c *Colony, code string) bool {
	for _, u := range c.Operational {
		if u.Type == code && u.Qty > 0 {
			return true
		}
	}
	return false
}

// RunSetups does a turn of work on the colony's set ups, oldest first, and
// returns the entities that are done, which are no longer the colony's.
// StartTransport must have been called this turn.
//
// The CWRK that a set up lists work on it, EMUPerCWRK each, and on nothing
// else. The colony's other CWRK help: as many as the work can use, up to the
// order's cap (which set ups of one Batch share, so they are built in turn),
// are shuttled to the job, which takes SetupShuttleMU of the colony's TRNS
// each, and the FUEL and crew for that; the ones the TRNS can't carry don't
// work. A CWRK that is shuttled does nothing else in the turn. The EMU of all
// of them is recorded in Construction.Setup, which Disassemble takes out of
// the turn's pool.
//
// When the work is done, the colony hands over what it reserved, people
// last, and the entity is complete. People who have died since are missed:
// the entity gets the ones there are, and waits if there are none at all.
func (c *Colony) RunSetups() []Colony {
	helpers, shuttled := c.FreeCWRK(), 0
	budgets := map[int]int{}
	none := func(int) int { return 0 }
	var done []Colony
	jobs := c.Setups[:0:0]
	for _, s := range c.Setups {
		s.Crew, s.Waiting = 0, ""
		need := max(0, s.Need-s.EMU)
		own := min(need, min(count(units.CWRK, 0, s.BOM), max(0, c.Cadres.CWRK))*EMUPerCWRK)
		need -= own
		crew := min(helpers, (need+EMUPerCWRK-1)/EMUPerCWRK)
		if s.MaxCWRK > 0 {
			left, ok := budgets[s.Batch]
			if !ok || s.Batch == 0 {
				left = s.MaxCWRK
			}
			crew = min(crew, left)
		}
		crew = c.loadLimit(crew, func(k int) int { return k * SetupShuttleMU }, none, none)
		c.load(crew * SetupShuttleMU)
		if s.MaxCWRK > 0 && s.Batch != 0 {
			left, ok := budgets[s.Batch]
			if !ok {
				left = s.MaxCWRK
			}
			budgets[s.Batch] = left - crew
		}
		helpers -= crew
		shuttled += crew
		s.Crew = crew
		s.EMU += own + min(need, crew*EMUPerCWRK)
		if s.EMU < s.Need {
			s.Waiting = "work"
			jobs = append(jobs, s)
			continue
		}
		if !c.completeSetup(&s) {
			s.Waiting = "population"
			jobs = append(jobs, s)
			continue
		}
		done = append(done, s.Entity)
	}
	c.Setups = jobs
	if len(jobs) == 0 {
		c.Setups = nil
	}
	c.Construction.Setup = (c.Reserved(units.CWRK, 0) + shuttled) * EMUPerCWRK
	return done
}

// completeSetup hands what the set up s reserved over to its entity. It
// reports whether it did: it does nothing if the colony has none of the
// people left.
func (c *Colony) completeSetup(s *Setup) bool {
	var p Population
	var k Cadres
	left := c.Population
	take := func(have *int, n int) int {
		n = max(0, min(n, *have))
		*have -= n
		return n
	}
	for _, u := range s.BOM {
		switch u.Type {
		case units.UEM:
			p.UEM += take(&left.UEM, u.Qty)
		case units.USK:
			p.USK += take(&left.USK, u.Qty)
		case units.PRO:
			p.PRO += take(&left.PRO, u.Qty)
		case units.SLD:
			p.SLD += take(&left.SLD, u.Qty)
		case units.CWRK:
			n := min(u.Qty, max(0, c.Cadres.CWRK), left.PRO, left.USK)
			k.CWRK += take(&left.PRO, n)
			p.PRO, p.USK = p.PRO+n, p.USK+take(&left.USK, n)
		case units.SPY:
			n := min(u.Qty, max(0, c.Cadres.SPY), left.PRO, left.SLD)
			k.SPY += take(&left.PRO, n)
			p.PRO, p.SLD = p.PRO+n, p.SLD+take(&left.SLD, n)
		}
	}
	if p.Units() < 1 {
		return false
	}
	for _, u := range s.BOM {
		if d, _ := units.Lookup(u.Type); !d.Population() {
			c.takeStored(u.Type, u.TechLevel, u.Qty)
		}
	}
	c.Population = left
	c.Cadres.CWRK -= k.CWRK
	c.Cadres.SPY -= k.SPY
	s.Entity.Population, s.Entity.Cadres = p, k
	return true
}

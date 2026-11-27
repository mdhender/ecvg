// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"math"
	"slices"
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
	// Want is the bill of material: everything that goes into the entity,
	// in the order it is supplied. BOM is the part of it that the colony has
	// reserved so far, and hands over when the work is done. People are
	// listed by their unit (UEM, USK, PRO, SLD) or their cadre (CWRK, SPY).
	Want []Unit `json:"want"`
	BOM  []Unit `json:"bom"`
	// Built is set once all of the entity's structure has been supplied. No
	// assembly work is done before that, and what is supplied after it must
	// be carried by the colony's TRNS.
	Built bool `json:"built,omitzero"`
	// Need is the EMU of assembly work that the entity takes, and EMU the
	// work done so far.
	Need int `json:"need"`
	EMU  int `json:"emu"`
	// MaxCWRK is the most CWRK that may work on the job in a turn, the ones
	// in the bill of material and the ones shuttled from the colony
	// together, or 0 for as many as it can use. Set ups with the same Batch,
	// if it isn't 0, share that crew: they are built one after another.
	MaxCWRK int `json:"max_cwrk,omitzero"`
	Batch   int `json:"batch,omitzero"`
	// Crew is the CWRK that were shuttled to the job in the prior turn,
	// Hauled the MU of late supplies that the TRNS carried for it then, and
	// Waiting says why the job isn't done: "structure", "units", "work", or
	// "population".
	Crew    int    `json:"crew,omitzero"`
	Hauled  int    `json:"hauled,omitzero"`
	Waiting string `json:"waiting,omitzero"`
}

// Short returns what the set up still wants and the colony hasn't reserved.
func (s Setup) Short() []Unit {
	var short []Unit
	for _, u := range s.Want {
		if n := u.Qty - count(u.Type, u.TechLevel, s.BOM); n > 0 {
			short = append(short, Unit{Type: u.Type, TechLevel: u.TechLevel, Qty: n})
		}
	}
	return short
}

// structureShort reports whether any of the set up's structure is still to
// be supplied.
func (s Setup) structureShort() bool {
	for _, u := range s.Short() {
		if d, _ := units.Lookup(u.Type); d.Encloses() > 0 {
			return true
		}
	}
	return false
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
// puts them in s.Entity as they will be when the work is done, adds them to
// the bill of material, s.Want, and reserves the ones the colony has (see
// Supply).
//
// The line is cut down to the room in the new entity (manual F3.2.1), if it
// is enclosed: units take their assembled volume if they are assembled there
// and their stowed volume if not, and people 1 VU each. Assembled structure
// takes none, and makes the room for what is listed after it. It is not cut
// down to what the colony has: the set up waits for the rest.
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
		p, ok := peopleOf(code)
		if !ok {
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
		tl = 0
	} else {
		assemble := d.RequiresAssembly && !it.Stowed
		switch {
		case assemble && (d.Encloses() > 0 || !to.Type.Enclosed()):
			// Structure needs no room, and neither does anything in the open.
		case assemble:
			free := to.SpaceFree()
			n = sort.Search(n, func(k int) bool { return d.Volume(tl, k+1).Cmp(free) > 0 })
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
	}
	s.Want = addUnits(s.Want, code, tl, n)
	return n
}

// peopleOf returns the people that make up one unit with the given code: a
// population unit, or a cadre that a set up can take.
func peopleOf(code string) (Population, bool) {
	switch code {
	case units.UEM:
		return Population{UEM: 1}, true
	case units.USK:
		return Population{USK: 1}, true
	case units.PRO:
		return Population{PRO: 1}, true
	case units.SLD:
		return Population{SLD: 1}, true
	case units.CWRK:
		return Population{PRO: 1, USK: 1}, true
	case units.SPY:
		return Population{PRO: 1, SLD: 1}, true
	}
	return Population{}, false
}

// Supply reserves, for the set up s, which must be one of the colony's own,
// what s still wants and the colony can spare, in the order of the bill of
// material: units in its storage that nothing has claimed or reserved, never
// assembled ones; people that no cadre holds and no set up has reserved; and,
// for CWRK and SPY, the cadre, whose members go along. It returns the MU of
// it that the colony's TRNS carried.
//
// While the entity's structure is going up, supplies need no TRNS. Once all
// of the structure has been supplied (s.Built), what was short is late: the
// colony's TRNS must carry it, and only what they can still carry this turn
// is reserved (see Transfer; a population unit is 1 MU).
func (c *Colony) Supply(s *Setup) int {
	late, hauled := s.Built, 0
	none := func(int) int { return 0 }
	for _, u := range s.Short() {
		code, tl, n := u.Type, u.TechLevel, u.Qty
		d, _ := units.Lookup(code)
		mass, fuelOut, proOut := none, none, none
		if p, ok := peopleOf(code); ok {
			switch code {
			case units.UEM:
				n = min(n, c.Population.UEM-c.Reserved(units.UEM, 0))
			case units.USK:
				n = min(n, c.FreeUSK())
			case units.PRO:
				n, proOut = min(n, c.FreePRO()-c.Transport.Crew), func(k int) int { return k }
			case units.SLD:
				n = min(n, c.FreeSLD())
			case units.CWRK:
				n = min(n, c.FreeCWRK())
			case units.SPY:
				n = min(n, c.FreeSPY())
			}
			mass = func(k int) int { return k * p.Mass() }
		} else {
			n = min(n, c.Stored(code, tl)-c.Claimed(code, tl))
			if code != units.TRNS { // TRNS move themselves
				mass = func(k int) int { return d.Mass(tl, k).Ceil() }
			}
			if code == units.FUEL {
				fuelOut = func(k int) int { return k }
			}
		}
		if late {
			n = c.loadLimit(n, mass, fuelOut, proOut)
		}
		if n = max(0, n); n == 0 {
			continue
		}
		s.BOM = addUnits(s.BOM, code, tl, n)
		if late {
			c.load(mass(n))
			hauled += mass(n)
		}
	}
	if !s.structureShort() {
		s.Built = true
	}
	return hauled
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
// First the colony supplies what the set up is short of (see Supply). Once
// the structure is up, the CWRK work. The ones in the bill of material work
// on their own set up, EMUPerCWRK each, and on nothing else. The colony's
// other CWRK help: as many as the work can use are shuttled to the job, which
// takes SetupShuttleMU of the colony's TRNS each, and the FUEL and crew for
// that; the ones the TRNS can't carry don't work. A CWRK that is shuttled
// does nothing else in the turn. The order's cap is on all the CWRK that
// work on the set up, its own first; set ups of one Batch share it, so they
// are built in turn. The EMU of all of them is recorded in
// Construction.Setup, which Disassemble takes out of the turn's pool.
//
// When everything has been supplied and the work is done, the colony hands
// over what it reserved, and the entity is complete. People who have died
// since are missed: the entity gets the ones there are, and waits if there
// are none at all.
func (c *Colony) RunSetups() []Colony {
	helpers, shuttled := c.FreeCWRK(), 0
	budgets := map[int]int{}
	none := func(int) int { return 0 }
	var done []Colony
	jobs := slices.Clone(c.Setups)
	c.Setups = jobs
	var keep []Setup
	for i := range jobs {
		s := &jobs[i]
		s.Crew, s.Waiting = 0, ""
		s.Hauled = c.Supply(s)
		if !s.Built {
			s.Waiting = "structure"
			keep = append(keep, *s)
			continue
		}
		need := max(0, s.Need-s.EMU)
		// The CWRK the order's cap still allows.
		allowed := math.MaxInt
		if s.MaxCWRK > 0 {
			allowed = s.MaxCWRK
			if left, ok := budgets[s.Batch]; ok && s.Batch != 0 {
				allowed = left
			}
		}
		own := min(count(units.CWRK, 0, s.BOM), max(0, c.Cadres.CWRK), allowed, (need+EMUPerCWRK-1)/EMUPerCWRK)
		work := min(need, own*EMUPerCWRK)
		need -= work
		crew := min(helpers, allowed-own, (need+EMUPerCWRK-1)/EMUPerCWRK)
		crew = c.loadLimit(crew, func(k int) int { return k * SetupShuttleMU }, none, none)
		c.load(crew * SetupShuttleMU)
		if s.MaxCWRK > 0 && s.Batch != 0 {
			budgets[s.Batch] = allowed - own - crew
		}
		helpers -= crew
		shuttled += crew
		s.Crew = crew
		s.EMU += work + min(need, crew*EMUPerCWRK)
		switch {
		case len(s.Short()) > 0:
			s.Waiting = "units"
		case s.EMU < s.Need:
			s.Waiting = "work"
		case !c.completeSetup(s):
			s.Waiting = "population"
		default:
			done = append(done, s.Entity)
			// The reservation is gone: the next set up may have its units.
			s.BOM, s.Want = nil, nil
			continue
		}
		keep = append(keep, *s)
	}
	c.Setups = keep
	c.Construction.Setup = (c.Reserved(units.CWRK, 0) + shuttled) * EMUPerCWRK
	return done
}

// CancelSetup drops the colony's set up of the entity with the given ID, and
// reports whether it had one. What the set up had reserved is the colony's
// to use again; the work done on it is lost.
func (c *Colony) CancelSetup(id int) bool {
	at := slices.IndexFunc(c.Setups, func(s Setup) bool { return s.Entity.ID == id })
	if at < 0 {
		return false
	}
	c.Setups = slices.Delete(slices.Clone(c.Setups), at, at+1)
	if len(c.Setups) == 0 {
		c.Setups = nil
	}
	return true
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

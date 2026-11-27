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
	// Delivered is the part of the bill of material that add-ons have
	// carried to the job site: it is no entity's until the work is done.
	// (People that add-ons deliver join the colony instead, and its BOM.)
	// Pledged is the part that add-ons have still to deliver, which the
	// colony does not supply.
	Delivered []Unit `json:"delivered,omitempty"`
	Pledged   []Unit `json:"pledged,omitempty"`
	// Turns is the number of turns that the set up has been worked on.
	// Add-ons deliver nothing in its first.
	Turns int `json:"turns,omitzero"`
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

// Short returns what the set up still wants: what the colony hasn't
// reserved and no add-on has delivered.
func (s Setup) Short() []Unit {
	var short []Unit
	for _, u := range s.Want {
		if n := u.Qty - count(u.Type, u.TechLevel, s.BOM, s.Delivered); n > 0 {
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
// level that the colony's set ups and add-ons have reserved. For people, the code is the
// population unit or the cadre.
func (c *Colony) Reserved(code string, tl int) int {
	n := 0
	for _, s := range c.Setups {
		n += count(code, tl, s.BOM)
	}
	for _, a := range c.AddOns {
		n += count(code, tl, a.BOM)
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
		code, tl := u.Type, u.TechLevel
		// What add-ons have pledged is theirs to supply.
		n := min(u.Qty-count(code, tl, s.Pledged), c.spare(code, tl))
		mass, fuelOut, proOut := unitMass(code, tl), none, none
		switch code {
		case units.PRO:
			n, proOut = min(n, c.FreePRO()-c.Transport.Crew), func(k int) int { return k }
		case units.FUEL:
			fuelOut = func(k int) int { return k }
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

// spare returns the quantity of the unit with the given code and tech level
// that the colony can still reserve: units in its storage that nothing has
// claimed or reserved, people that no cadre holds and nothing has reserved,
// or cadres that nothing has reserved.
func (c *Colony) spare(code string, tl int) int {
	switch code {
	case units.UEM:
		return max(0, c.Population.UEM-c.Reserved(units.UEM, 0))
	case units.USK:
		return c.FreeUSK()
	case units.PRO:
		return c.FreePRO()
	case units.SLD:
		return c.FreeSLD()
	case units.CWRK:
		return c.FreeCWRK()
	case units.SPY:
		return c.FreeSPY()
	}
	return max(0, c.Stored(code, tl)-c.Claimed(code, tl))
}

// unitMass returns a function for the MU of TRNS capacity that k of the unit
// with the given code and tech level take: their mass, rounded up, 1 MU for
// each population unit, and nothing for TRNS, which move themselves.
func unitMass(code string, tl int) func(k int) int {
	if p, ok := peopleOf(code); ok {
		return func(k int) int { return k * p.Mass() }
	}
	d, _ := units.Lookup(code)
	if code == units.TRNS {
		return func(int) int { return 0 }
	}
	return func(k int) int { return d.Mass(tl, k).Ceil() }
}

// lessUnits takes qty of the unit with the given code and tech level off
// list, and drops the entry if none are left.
func lessUnits(list []Unit, code string, tl, qty int) []Unit {
	var out []Unit
	for _, u := range list {
		if u.Type == code && u.TechLevel == tl {
			n := min(qty, u.Qty)
			u.Qty -= n
			qty -= n
		}
		if u.Qty > 0 {
			out = append(out, u)
		}
	}
	return out
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
	// The CWRK that the colony's add-ons shuttled this turn are busy.
	prior := c.Construction.Setup / EMUPerCWRK
	helpers, shuttled := max(0, c.FreeCWRK()-prior), 0
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
		s.Turns++
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
	c.Construction.Setup = (c.Reserved(units.CWRK, 0) + shuttled + prior) * EMUPerCWRK
	return done
}

// CancelSetup drops the colony's set up of the entity with the given ID, and
// reports whether it had one. What the set up had reserved is the colony's
// to use again, and so is what add-ons had delivered to the job site, which
// goes into its storage whatever the room; the work done is lost.
func (c *Colony) CancelSetup(id int) bool {
	at := slices.IndexFunc(c.Setups, func(s Setup) bool { return s.Entity.ID == id })
	if at < 0 {
		return false
	}
	for _, u := range c.Setups[at].Delivered {
		if d, _ := units.Lookup(u.Type); c.HasDepot() && d.DepotAllowed() {
			c.Depot = addUnits(c.Depot, u.Type, u.TechLevel, u.Qty)
		} else {
			c.Cargo = addUnits(c.Cargo, u.Type, u.TechLevel, u.Qty)
		}
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

// AddOn is the help that a colony gives with a set up that another colony is
// doing (manual F5.0): units that the set up is short of, and CWRK.
type AddOn struct {
	// To is the ID of the entity that is being set up.
	To int `json:"to"`
	// Want is what the colony has still to deliver, and BOM the part of it
	// that it has reserved. People are listed as in Setup.
	Want []Unit `json:"want"`
	BOM  []Unit `json:"bom"`
	// MaxCWRK is the most CWRK of the colony that may be shuttled to the job
	// in a turn, or 0 for as many as it can use.
	MaxCWRK int `json:"max_cwrk,omitzero"`
	// Crew is the CWRK that were shuttled to the job in the prior turn,
	// Hauled the MU that the TRNS carried to it then, and Waiting says why
	// the add-on isn't done: "orbit" (the colony is somewhere else), "start"
	// (the set up has only just started), "units", or "transport".
	Crew    int    `json:"crew,omitzero"`
	Hauled  int    `json:"hauled,omitzero"`
	Waiting string `json:"waiting,omitzero"`
}

// Pledge adds to the add-on a, which must be one of the colony's own, up to
// qty of the unit with the given code and tech level for the set up s, which
// another colony is doing, and returns how many: as many as s is short of
// and no add-on has pledged. The colony reserves what it has of them (see
// ReserveAddOn).
func (c *Colony) Pledge(a *AddOn, s *Setup, code string, tl, qty int) int {
	n := 0
	for _, u := range s.Short() {
		if u.Type == code && u.TechLevel == tl {
			n = max(0, min(qty, u.Qty-count(code, tl, s.Pledged)))
		}
	}
	if n > 0 {
		a.Want = addUnits(a.Want, code, tl, n)
		s.Pledged = addUnits(s.Pledged, code, tl, n)
		c.ReserveAddOn(a)
	}
	return n
}

// ReserveAddOn reserves, for the add-on a, which must be one of the colony's
// own, what a still wants and the colony can spare, in the order listed.
func (c *Colony) ReserveAddOn(a *AddOn) {
	for _, u := range a.Want {
		n := min(u.Qty-count(u.Type, u.TechLevel, a.BOM), c.spare(u.Type, u.TechLevel))
		if n > 0 {
			a.BOM = addUnits(a.BOM, u.Type, u.TechLevel, n)
		}
	}
}

// RunAddOn does a turn of the add-on a, which must be one of the colony's
// own, for the set up s of the colony builder, and reports whether the
// add-on is done: it has delivered everything. StartTransport must have been
// called this turn, and builder.RunSetups must come after.
//
// The colony reserves what it can of what is wanted. Then its TRNS carry
// what is reserved to the job site, as much as they can still carry this
// turn (see Transfer; a population unit is 1 MU): units leave the colony and
// wait at the site (s.Delivered), and people join the builder, which reserves
// them for the set up. Nothing is delivered in the set up's first turn.
// If the structure is up, the colony's free CWRK, as many as the work can
// use and the add-on allows, are shuttled to the job like the builder's
// (SetupShuttleMU each), and do nothing else in the turn; their EMU is added
// to Construction.Setup.
//
// If the colony is not at the builder's planet, nothing happens: what is
// reserved stays reserved.
func (c *Colony) RunAddOn(a *AddOn, builder *Colony, s *Setup) bool {
	a.Crew, a.Hauled, a.Waiting = 0, 0, ""
	if c.PlanetID != builder.PlanetID {
		a.Waiting = "orbit"
		return false
	}
	c.ReserveAddOn(a)
	if s.Turns < 1 {
		a.Waiting = "start"
		return false
	}
	none := func(int) int { return 0 }
	for _, u := range slices.Clone(a.BOM) {
		code, tl := u.Type, u.TechLevel
		mass := unitMass(code, tl)
		n := c.loadLimit(u.Qty, mass, none, none)
		if n <= 0 {
			continue
		}
		if p, ok := peopleOf(code); ok {
			moved := Population{UEM: p.UEM * n, USK: p.USK * n, PRO: p.PRO * n, SLD: p.SLD * n}
			c.Population = c.Population.Less(moved)
			builder.Population = builder.Population.Less(Population{UEM: -moved.UEM, USK: -moved.USK, PRO: -moved.PRO, SLD: -moved.SLD})
			switch code {
			case units.CWRK:
				c.Cadres.CWRK -= n
				builder.Cadres.CWRK += n
			case units.SPY:
				c.Cadres.SPY -= n
				builder.Cadres.SPY += n
			}
			s.BOM = addUnits(s.BOM, code, tl, n)
		} else {
			c.takeStored(code, tl, n)
			s.Delivered = addUnits(s.Delivered, code, tl, n)
		}
		c.load(mass(n))
		a.Hauled += mass(n)
		a.BOM = lessUnits(a.BOM, code, tl, n)
		a.Want = lessUnits(a.Want, code, tl, n)
		s.Pledged = lessUnits(s.Pledged, code, tl, n)
	}
	if !s.structureShort() {
		s.Built = true
	}
	if need := max(0, s.Need-s.EMU); s.Built && need > 0 {
		crew := min(c.FreeCWRK()-c.Construction.Setup/EMUPerCWRK, (need+EMUPerCWRK-1)/EMUPerCWRK)
		if a.MaxCWRK > 0 {
			crew = min(crew, a.MaxCWRK)
		}
		crew = c.loadLimit(crew, func(k int) int { return k * SetupShuttleMU }, none, none)
		c.load(crew * SetupShuttleMU)
		a.Crew = crew
		c.Construction.Setup += crew * EMUPerCWRK
		s.EMU += min(need, crew*EMUPerCWRK)
	}
	switch {
	case len(a.Want) == 0:
		return true
	case len(a.BOM) > 0:
		a.Waiting = "transport"
	default:
		a.Waiting = "units"
	}
	return false
}

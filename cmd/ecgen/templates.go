// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

const errInvalidTemplate = cerrs.Error("invalid template")

// defaultHomeStarTemplate is the home star template used unless the game
// master names another: an Earth-like system with a planet in every orbit, the
// one in orbit 5 an asteroid belt.
//
//go:embed homestar-template.json
var defaultHomeStarTemplate []byte

// defaultEmpireTemplate is the empire template used unless the game master
// names another. Its numbers come from the sample turn 1 report in the 1978
// manual.
//
//go:embed empire-template.json
var defaultEmpireTemplate []byte

// loadTemplate returns the contents of the template file path, or def if path
// is empty.
func loadTemplate(path string, def []byte) ([]byte, error) {
	if path == "" {
		return def, nil
	}
	return os.ReadFile(path)
}

// HomeStarTemplate describes every home star in a game: what is in each of its
// orbits.
type HomeStarTemplate struct {
	// Orbits are the star's orbits, one for each of 1..OrbitsPerStar, in order.
	Orbits []TemplateOrbit `json:"orbits"`
}

// TemplateOrbit is an orbit in a HomeStarTemplate.
type TemplateOrbit struct {
	OrbitNo int `json:"orbit_no"`
	// Note describes the orbit for the game master. The generator ignores it.
	Note string `json:"note,omitempty"`
	// Planet is the planet in the orbit, or nil if the orbit is empty.
	Planet *TemplatePlanet `json:"planet"`
}

// TemplatePlanet is a planet in a HomeStarTemplate.
type TemplatePlanet struct {
	// HomeWorld marks the home world. It must be set on the planet in orbit
	// game.HomeWorldOrbitNo and on no other.
	HomeWorld bool `json:"homeworld,omitzero"`
	Habitable bool `json:"habitable,omitzero"`
	// Viability is 0 if the planet is not habitable, otherwise
	// game.MinViability..game.MaxViability.
	Viability int `json:"viability,omitzero"`
	// Deposits are the planet's deposits, numbered from 1 in this order.
	Deposits []TemplateDeposit `json:"deposits"`
}

// TemplateDeposit is a deposit in a HomeStarTemplate.
type TemplateDeposit struct {
	Type  game.DepositType `json:"type"`
	Size  int              `json:"size"`
	Yield int              `json:"yield"`
}

// parseHomeStarTemplate decodes data, rejecting unknown members, and
// validates the result.
func parseHomeStarTemplate(data []byte) (HomeStarTemplate, error) {
	var t HomeStarTemplate
	if err := json.Unmarshal(data, &t, json.RejectUnknownMembers(true)); err != nil {
		return HomeStarTemplate{}, err
	}
	if err := t.Validate(); err != nil {
		return HomeStarTemplate{}, err
	}
	return t, nil
}

// Validate reports every way t breaks the rules for a home star. It must have
// exactly game.OrbitsPerStar orbits, numbered from 1 in order, and the orbit
// game.HomeWorldOrbitNo must hold the home world: a planet marked homeworld,
// habitable, with viability game.MaxViability. No other planet may be marked
// homeworld. Every planet must have a viability of 0 if it is not habitable or
// game.MinViability..game.MaxViability if it is, and
// 1..game.MaxDepositsPerPlanet deposits whose sizes and yields are within the
// ranges for their types. The errors match errInvalidTemplate.
func (t HomeStarTemplate) Validate() error {
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{errInvalidTemplate}, args...)...))
	}
	if len(t.Orbits) != game.OrbitsPerStar {
		bad("%d orbits: want %d", len(t.Orbits), game.OrbitsPerStar)
	}
	for i, o := range t.Orbits {
		if o.OrbitNo != i+1 {
			bad("orbit %d: orbit_no %d: want %d", i+1, o.OrbitNo, i+1)
		}
		p := o.Planet
		if p == nil {
			if i+1 == game.HomeWorldOrbitNo {
				bad("orbit %d: empty: want the home world", i+1)
			}
			continue
		}
		isHome := i+1 == game.HomeWorldOrbitNo
		if p.HomeWorld && !isHome {
			bad("orbit %d: homeworld: only orbit %d holds the home world", i+1, game.HomeWorldOrbitNo)
		} else if isHome && !p.HomeWorld {
			bad("orbit %d: home world must be marked homeworld", i+1)
		}
		switch {
		case isHome && (!p.Habitable || p.Viability != game.MaxViability):
			bad("orbit %d: home world must be habitable with viability %d", i+1, game.MaxViability)
		case p.Habitable && (p.Viability < game.MinViability || p.Viability > game.MaxViability):
			bad("orbit %d: habitable with viability %d: want %d..%d", i+1, p.Viability, game.MinViability, game.MaxViability)
		case !p.Habitable && p.Viability != 0:
			bad("orbit %d: not habitable with viability %d: want 0", i+1, p.Viability)
		}
		if n := len(p.Deposits); n < 1 || n > game.MaxDepositsPerPlanet {
			bad("orbit %d: %d deposits: want 1..%d", i+1, n, game.MaxDepositsPerPlanet)
		}
		for j, d := range p.Deposits {
			a, ok := depositAmounts[d.Type]
			if !ok {
				bad("orbit %d: deposit %d: invalid type %q", i+1, j+1, d.Type)
				continue
			}
			if d.Size < a.minSize || d.Size > a.maxSize {
				bad("orbit %d: deposit %d: %s size %d: want %d..%d", i+1, j+1, d.Type, d.Size, a.minSize, a.maxSize)
			}
			if d.Yield < a.minYield || d.Yield > a.maxYield {
				bad("orbit %d: deposit %d: %s yield %d: want %d..%d", i+1, j+1, d.Type, d.Yield, a.minYield, a.maxYield)
			}
		}
	}
	return errors.Join(errs...)
}

// EmpireTemplate describes the initial assets of every empire: its colonies,
// all on its home world.
type EmpireTemplate struct {
	Colonies []TemplateColony `json:"colonies"`
}

// TemplateColony is a colony in an EmpireTemplate. It is a game.Colony without
// the IDs, which the generator supplies, and without mining groups: an empire
// starts with none and claims deposits for itself.
type TemplateColony struct {
	No               int             `json:"no"`
	Type             game.EntityType `json:"type"`
	TechLevel        int             `json:"tech_level"`
	Population       game.Population `json:"population"`
	Cadres           game.Cadres     `json:"cadres,omitzero"`
	Operational      []game.Unit     `json:"operational"`
	Cargo            []game.Unit     `json:"cargo"`
	Depot            []game.Unit     `json:"depot,omitempty"`
	DepotOperational []game.Unit     `json:"depot_operational,omitempty"`
	FarmingGroups    []game.Group    `json:"farming_groups"`
	FactoryGroups    []game.Group    `json:"factory_groups"`
}

// parseEmpireTemplate decodes data, rejecting unknown members, and validates
// the result.
func parseEmpireTemplate(data []byte) (EmpireTemplate, error) {
	var t EmpireTemplate
	if err := json.Unmarshal(data, &t, json.RejectUnknownMembers(true)); err != nil {
		return EmpireTemplate{}, err
	}
	if err := t.Validate(); err != nil {
		return EmpireTemplate{}, err
	}
	return t, nil
}

// Validate reports every way t breaks the rules for an empire's initial
// assets. There must be at least one colony, and colonies must be numbered
// from 1 in order, with one of the four entity types and a tech level of at
// least 1. No population
// may be negative. Units need a type from the unit catalogue (not population
// or research points), a quantity of at least 1, and a tech level that is
// valid for the unit; operational units must be of a kind that is assembled. Only an open-air
// surface colony may list units in a depot (depot for stored units and
// depot_operational for assembled ones), and not units of a class that must
// be kept in the colony. A colony may start with more than it has capacity
// for; that isn't checked here. A farming group's units are all
// of one tech level, so it has one entry of units. Groups of each kind must be numbered from 1 in
// order and hold at least one entry of units, each with a quantity and tech
// level of at least 1. The errors match errInvalidTemplate.
func (t EmpireTemplate) Validate() error {
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{errInvalidTemplate}, args...)...))
	}
	if len(t.Colonies) == 0 {
		bad("no colonies: want at least 1")
	}
	for i, c := range t.Colonies {
		if c.No != i+1 {
			bad("colony %d: no %d: want %d", i+1, c.No, i+1)
		}
		if c.Type == "" {
			bad("colony %d: missing type", i+1)
		}
		if c.TechLevel < 1 {
			bad("colony %d: tech_level %d: want at least 1", i+1, c.TechLevel)
		}
		if p := c.Population; p.UEM < 0 || p.USK < 0 || p.PRO < 0 || p.SLD < 0 {
			bad("colony %d: negative population", i+1)
		}
		if k := c.Cadres; k.CWRK < 0 || k.TRNE < 0 || k.SPY < 0 {
			bad("colony %d: negative cadres", i+1)
		}
		if k := c.Cadres; k.Fit(c.Population) != k {
			bad("colony %d: cadres: the population can't make them up (CWRK is 1 PRO and 1 USK each, TRNE 1 USK and 1 PRO per 100, SPY 1 PRO and 1 SLD)", i+1)
		}
		list := func(kind string, operational bool, us []game.Unit) {
			for j, u := range us {
				if u.Type == "" {
					bad("colony %d: %s %d: missing type", i+1, kind, j+1)
				}
				if u.Qty < 1 {
					bad("colony %d: %s %d: qty %d: want at least 1", i+1, kind, j+1, u.Qty)
				}
				if u.TechLevel < 0 {
					bad("colony %d: %s %d: tech_level %d: want at least 0", i+1, kind, j+1, u.TechLevel)
				}
				if u.Type == "" {
					continue
				}
				switch d, ok := units.Lookup(u.Type); {
				case !ok:
					bad("colony %d: %s %d: unknown unit %q", i+1, kind, j+1, u.Type)
				case d.Population() || d.Code == units.RP:
					bad("colony %d: %s %d: %s can't be listed as a unit", i+1, kind, j+1, u.Type)
				case !d.ValidTechLevel(u.TechLevel) && u.TechLevel >= 0:
					if d.HasTechLevel {
						bad("colony %d: %s %d: %s tech_level %d: want %d to %d", i+1, kind, j+1, u.Type, u.TechLevel, units.MinTechLevel, units.MaxTechLevel)
					} else {
						bad("colony %d: %s %d: %s tech_level %d: it has no tech level", i+1, kind, j+1, u.Type, u.TechLevel)
					}
				case operational && !d.RequiresAssembly:
					bad("colony %d: %s %d: %s is never assembled, so it belongs in cargo", i+1, kind, j+1, u.Type)
				}
			}
		}
		list("operational", true, c.Operational)
		list("cargo", false, c.Cargo)
		list("depot", false, c.Depot)
		list("depot_operational", true, c.DepotOperational)
		if len(c.Depot)+len(c.DepotOperational) > 0 && c.Type != game.COPN {
			bad("colony %d: depot: only an open-air surface colony (COPN) has a depot", i+1)
		}
		for _, l := range []struct {
			kind string
			us   []game.Unit
		}{{"depot", c.Depot}, {"depot_operational", c.DepotOperational}} {
			for j, u := range l.us {
				if d, ok := units.Lookup(u.Type); ok && !d.Population() && !d.DepotAllowed() {
					bad("colony %d: %s %d: %s (%s) can't be in a depot: it must be in the colony", i+1, l.kind, j+1, u.Type, d.Class)
				}
			}
		}
		// Group numbers are shared by every kind of group of a colony.
		groupNos := map[int]bool{}
		groups := func(kind string, gs []game.Group) {
			for j, g := range gs {
				switch {
				case g.No < 1:
					bad("colony %d: %s group %d: no %d: want 1 or more", i+1, kind, j+1, g.No)
				case groupNos[g.No]:
					bad("colony %d: %s group %d: no %d: another group of the colony has that number", i+1, kind, j+1, g.No)
				case j > 0 && g.No < gs[j-1].No:
					bad("colony %d: %s group %d: no %d: want the groups in order of their numbers", i+1, kind, j+1, g.No)
				}
				groupNos[g.No] = true
				if len(g.Units) == 0 {
					bad("colony %d: %s group %d: no units", i+1, kind, j+1)
				}
				for _, u := range g.Units {
					if u.Qty < 1 || u.TechLevel < 1 {
						bad("colony %d: %s group %d: qty %d at tech_level %d: want at least 1 of each", i+1, kind, j+1, u.Qty, u.TechLevel)
					}
				}
			}
		}
		groups("farming", c.FarmingGroups)
		for j, g := range c.FarmingGroups {
			if len(g.Units) > 1 {
				bad("colony %d: farming group %d: %d entries of units: every unit of a farming group is of the same tech level", i+1, j+1, len(g.Units))
			}
		}
		groups("factory", c.FactoryGroups)
	}
	return errors.Join(errs...)
}

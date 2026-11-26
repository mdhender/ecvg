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
	No            int             `json:"no"`
	Type          string          `json:"type"`
	TechLevel     int             `json:"tech_level"`
	Population    game.Population `json:"population"`
	Operational   []game.Unit     `json:"operational"`
	Cargo         []game.Unit     `json:"cargo"`
	FarmingGroups []game.Group    `json:"farming_groups"`
	FactoryGroups []game.Group    `json:"factory_groups"`
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
// from 1 in order, with a type and a tech level of at least 1. No population
// may be negative. Units need a type, a quantity of at least 1, and a tech
// level that is not negative. Groups of each kind must be numbered from 1 in
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
		units := func(kind string, us []game.Unit) {
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
			}
		}
		units("operational", c.Operational)
		units("cargo", c.Cargo)
		groups := func(kind string, gs []game.Group) {
			for j, g := range gs {
				if g.No != j+1 {
					bad("colony %d: %s group %d: no %d: want %d", i+1, kind, j+1, g.No, j+1)
				}
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
		groups("factory", c.FactoryGroups)
	}
	return errors.Join(errs...)
}

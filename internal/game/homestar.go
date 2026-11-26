// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	_ "embed"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// HomeStarTemplateFileName is the name of the file in a game's folder that
// holds the template for the game's home stars.
const HomeStarTemplateFileName = "homestar-template.json"

const (
	// HomeWorldOrbitNo is the orbit of a home star that holds its home world.
	HomeWorldOrbitNo = 3
	// HomeWorldViability is the viability of every home world.
	HomeWorldViability = MaxViability
)

// defaultHomeStarTemplate is the template CreateHomeStarTemplate writes: an
// Earth-like system of 9 planets in 10 orbits, with orbit 5 empty where the
// asteroid belt would be.
//
//go:embed homestar-template.json
var defaultHomeStarTemplate []byte

// HomeStarTemplate describes every home star in a game: what is in each of its
// orbits. The game master can edit it before the home stars are created.
type HomeStarTemplate struct {
	// Orbits are the star's orbits, one for each of 1..OrbitsPerStar, in order.
	Orbits []TemplateOrbit `json:"orbits"`
}

// TemplateOrbit is an orbit in a HomeStarTemplate.
type TemplateOrbit struct {
	OrbitNo int `json:"orbit_no"`
	// Note describes the orbit for the game master. The game ignores it.
	Note string `json:"note,omitempty"`
	// Planet is the planet in the orbit, or nil if the orbit is empty.
	Planet *TemplatePlanet `json:"planet"`
}

// TemplatePlanet is a planet in a HomeStarTemplate.
type TemplatePlanet struct {
	Habitable bool `json:"habitable"`
	// Viability is 0 if the planet is not habitable, otherwise
	// MinViability..MaxViability.
	Viability int `json:"viability"`
	// Deposits are the planet's deposits, numbered from 1 in this order.
	Deposits []TemplateDeposit `json:"deposits"`
}

// DefaultHomeStarTemplate returns the template that CreateHomeStarTemplate
// writes.
func DefaultHomeStarTemplate() HomeStarTemplate {
	t, err := parseHomeStarTemplate(defaultHomeStarTemplate)
	if err != nil {
		panic(fmt.Sprintf("game: default home star template: %v", err))
	}
	return t
}

// Validate reports every way t breaks the rules for a home star. It must have
// exactly OrbitsPerStar orbits, numbered from 1 in order, and the orbit
// HomeWorldOrbitNo must hold the home world: a habitable planet with viability
// HomeWorldViability. Every planet must have a viability of 0 if it is not
// habitable or MinViability..MaxViability if it is, and 1..MaxDepositsPerPlanet
// deposits whose sizes and yields are within the ranges for their types. The
// errors match ErrInvalidTemplate.
func (t HomeStarTemplate) Validate() error {
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{ErrInvalidTemplate}, args...)...))
	}
	if len(t.Orbits) != OrbitsPerStar {
		bad("%d orbits: want %d", len(t.Orbits), OrbitsPerStar)
	}
	for i, o := range t.Orbits {
		if o.OrbitNo != i+1 {
			bad("orbit %d: orbit_no %d: want %d", i+1, o.OrbitNo, i+1)
		}
		p := o.Planet
		if p == nil {
			if o.OrbitNo == HomeWorldOrbitNo {
				bad("orbit %d: empty: want the home world", o.OrbitNo)
			}
			continue
		}
		switch {
		case o.OrbitNo == HomeWorldOrbitNo && (!p.Habitable || p.Viability != HomeWorldViability):
			bad("orbit %d: home world must be habitable with viability %d", o.OrbitNo, HomeWorldViability)
		case p.Habitable && (p.Viability < MinViability || p.Viability > MaxViability):
			bad("orbit %d: habitable with viability %d: want %d..%d", o.OrbitNo, p.Viability, MinViability, MaxViability)
		case !p.Habitable && p.Viability != 0:
			bad("orbit %d: not habitable with viability %d: want 0", o.OrbitNo, p.Viability)
		}
		checkDeposits(p.Deposits, func(format string, args ...any) {
			bad("orbit %d: "+format, append([]any{o.OrbitNo}, args...)...)
		})
	}
	return errors.Join(errs...)
}

// checkDeposits calls bad for every way deposits break the rules for a
// planet's deposits: there must be 1..MaxDepositsPerPlanet of them, and each
// one's size and yield must be within the ranges for its type.
func checkDeposits(deposits []TemplateDeposit, bad func(format string, args ...any)) {
	if n := len(deposits); n < 1 || n > MaxDepositsPerPlanet {
		bad("%d deposits: want 1..%d", n, MaxDepositsPerPlanet)
	}
	for i, d := range deposits {
		a, ok := depositAmounts[d.Type]
		if !ok {
			bad("deposit %d: invalid type %q", i+1, d.Type)
			continue
		}
		if d.Size < a.minSize || d.Size > a.maxSize {
			bad("deposit %d: %s size %d: want %d..%d", i+1, d.Type, d.Size, a.minSize, a.maxSize)
		}
		if d.Yield < a.minYield || d.Yield > a.maxYield {
			bad("deposit %d: %s yield %d: want %d..%d", i+1, d.Type, d.Yield, a.minYield, a.maxYield)
		}
	}
}

// CreateHomeStarTemplate writes the default home star template to
// homestar-template.json in dir, which must already hold a game, and returns
// it. The file is indented, with one deposit per line, because the game
// master edits it by hand. It fails with ErrTemplateExists if the template
// already exists, so it never overwrites the game master's changes.
func CreateHomeStarTemplate(dir string) (HomeStarTemplate, error) {
	if _, err := Load(dir); err != nil {
		return HomeStarTemplate{}, err
	}
	t := DefaultHomeStarTemplate()
	path := filepath.Join(dir, HomeStarTemplateFileName)
	if err := writeNew(path, defaultHomeStarTemplate); errors.Is(err, ErrGameExists) {
		return HomeStarTemplate{}, fmt.Errorf("%s: %w", path, ErrTemplateExists)
	} else if err != nil {
		return HomeStarTemplate{}, err
	}
	return t, nil
}

// LoadHomeStarTemplate reads homestar-template.json from dir and validates it.
func LoadHomeStarTemplate(dir string) (HomeStarTemplate, error) {
	path := filepath.Join(dir, HomeStarTemplateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return HomeStarTemplate{}, err
	}
	t, err := parseHomeStarTemplate(data)
	if err != nil {
		return HomeStarTemplate{}, fmt.Errorf("%s: %w", path, err)
	}
	return t, nil
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

// Planets returns the number of planets in t.
func (t HomeStarTemplate) Planets() int {
	n := 0
	for _, o := range t.Orbits {
		if o.Planet != nil {
			n++
		}
	}
	return n
}

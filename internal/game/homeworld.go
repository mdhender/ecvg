// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// HomeWorldTemplateFileName is the name of the file in a game's folder that
// holds the template for the game's home worlds.
const HomeWorldTemplateFileName = "homeworld-template.json"

// HomeStarsPerStellium is the number of stars in a home Stellium. A game has
// one home Stellium, home star, and home world for each Stellium with this
// many stars, so at most 15 species.
const HomeStarsPerStellium = 2

// MaxDepositsPerPlanet is the most deposits a planet can have.
const MaxDepositsPerPlanet = 31

const (
	ErrInvalidTemplate = cerrs.Error("invalid home world template")
	ErrTemplateExists  = cerrs.Error("home world template already exists")
)

// HomeWorldTemplate describes every home world in a game. The game master can
// edit it before the home worlds are created. Home worlds are always
// habitable.
type HomeWorldTemplate struct {
	// OrbitNo is the orbit of its home star that a home world is in.
	OrbitNo int `json:"orbit_no"`
	// Viability is a home world's viability.
	Viability int `json:"viability"`
	// Deposits are a home world's deposits, numbered from 1 in this order.
	Deposits []TemplateDeposit `json:"deposits"`
}

// TemplateDeposit is a deposit in a HomeWorldTemplate.
type TemplateDeposit struct {
	Type  DepositType `json:"type"`
	Size  int         `json:"size"`
	Yield int         `json:"yield"`
}

// DefaultHomeWorldTemplate returns the template that CreateHomeWorldTemplate
// writes: orbit 3, viability 20, and 16 deposits (the average for a planet):
// 1 GOLD and 5 each of FUEL, MTL, and NMTL, each with the middle size and
// yield of its type.
func DefaultHomeWorldTemplate() HomeWorldTemplate {
	t := HomeWorldTemplate{OrbitNo: 3, Viability: 20}
	for _, dt := range []struct {
		typ   DepositType
		count int
	}{{GOLD, 1}, {FUEL, 5}, {MTL, 5}, {NMTL, 5}} {
		a := depositAmounts[dt.typ]
		for range dt.count {
			t.Deposits = append(t.Deposits, TemplateDeposit{
				Type:  dt.typ,
				Size:  (a.minSize + a.maxSize) / 2,
				Yield: (a.minYield + a.maxYield) / 2,
			})
		}
	}
	return t
}

// Validate reports every way t breaks the rules for a planet: its orbit must
// be 1..OrbitsPerStar, its viability MinViability..MaxViability, it must have
// 1..MaxDepositsPerPlanet deposits, and each deposit's size and yield must be
// within the ranges for its type. The errors match ErrInvalidTemplate.
func (t HomeWorldTemplate) Validate() error {
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{ErrInvalidTemplate}, args...)...))
	}
	if t.OrbitNo < 1 || t.OrbitNo > OrbitsPerStar {
		bad("orbit_no %d: want 1..%d", t.OrbitNo, OrbitsPerStar)
	}
	if t.Viability < MinViability || t.Viability > MaxViability {
		bad("viability %d: want %d..%d", t.Viability, MinViability, MaxViability)
	}
	if n := len(t.Deposits); n < 1 || n > MaxDepositsPerPlanet {
		bad("%d deposits: want 1..%d", n, MaxDepositsPerPlanet)
	}
	for i, d := range t.Deposits {
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
	return errors.Join(errs...)
}

// CreateHomeWorldTemplate writes DefaultHomeWorldTemplate to
// homeworld-template.json in dir, which must already hold a game. Unlike the
// other game files, the template is indented, because the game master edits
// it by hand. It fails with ErrGameExists if the template already exists, so
// it never overwrites the game master's changes.
func CreateHomeWorldTemplate(dir string) (HomeWorldTemplate, error) {
	if _, err := Load(dir); err != nil {
		return HomeWorldTemplate{}, err
	}
	t := DefaultHomeWorldTemplate()
	data, err := json.Marshal(t)
	if err != nil {
		return HomeWorldTemplate{}, err
	}
	data, err = indent(data)
	if err != nil {
		return HomeWorldTemplate{}, err
	}
	path := filepath.Join(dir, HomeWorldTemplateFileName)
	if err := writeNew(path, data); errors.Is(err, ErrGameExists) {
		return HomeWorldTemplate{}, fmt.Errorf("%s: %w", path, ErrTemplateExists)
	} else if err != nil {
		return HomeWorldTemplate{}, err
	}
	return t, nil
}

// indent formats JSON with two-space indentation and a final newline.
func indent(data []byte) ([]byte, error) {
	var v jsontext.Value = data
	if err := v.Indent(jsontext.WithIndent("  ")); err != nil {
		return nil, err
	}
	return append(v, '\n'), nil
}

// LoadHomeWorldTemplate reads homeworld-template.json from dir and validates
// it.
func LoadHomeWorldTemplate(dir string) (HomeWorldTemplate, error) {
	var t HomeWorldTemplate
	if err := loadJSON(dir, HomeWorldTemplateFileName, &t); err != nil {
		return HomeWorldTemplate{}, err
	}
	if err := t.Validate(); err != nil {
		return HomeWorldTemplate{}, fmt.Errorf("%s: %w", filepath.Join(dir, HomeWorldTemplateFileName), err)
	}
	return t, nil
}

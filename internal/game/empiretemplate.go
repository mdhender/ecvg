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
	"regexp"
	"slices"
)

// EmpireTemplateFileName is the name of the file in a game's folder that holds
// the template for every new empire's initial assets.
const EmpireTemplateFileName = "empire-template.json"

// The number of each kind of group every empire starts with.
const (
	EmpireFarmingGroups = 1
	EmpireMiningGroups  = 1
	EmpireFactoryGroups = 4
)

// PopulationType is a kind of population unit: a group (UEM, USK, PRO, SLD)
// or a cadre trained for a purpose (CNW, SPY).
type PopulationType string

const (
	UEM PopulationType = "UEM" // unemployed
	USK PopulationType = "USK" // unskilled
	PRO PopulationType = "PRO" // professional
	SLD PopulationType = "SLD" // soldier
	CNW PopulationType = "CNW" // construction worker
	SPY PopulationType = "SPY" // spy
)

// populationTypes is every PopulationType, in the order reports list them.
var populationTypes = []PopulationType{UEM, USK, PRO, SLD, CNW, SPY}

// CacheType is a kind of resource unit an empire starts with.
type CacheType string

// FOOD is food. The other cache types are the resources refined from the
// FUEL, MTL, and NMTL deposit types, and have the same names.
const FOOD CacheType = "FOOD"

// cacheTypes is every CacheType.
var cacheTypes = []CacheType{FOOD, CacheType(FUEL), CacheType(MTL), CacheType(NMTL)}

// defaultEmpireTemplate is the template CreateEmpireTemplate writes. Its
// numbers come from the sample turn 1 report in the 1978 manual.
//
//go:embed empire-template.json
var defaultEmpireTemplate []byte

// EmpireTemplate describes the initial assets of every new empire, all on its
// home world. The game master can edit it before empires are created.
type EmpireTemplate struct {
	Population    []TemplatePopulation   `json:"population"`
	FarmingGroups []TemplateFarmingGroup `json:"farming_groups"`
	MiningGroups  []TemplateMiningGroup  `json:"mining_groups"`
	FactoryGroups []TemplateFactoryGroup `json:"factory_groups"`
	Cache         []TemplateCache        `json:"cache"`
}

// TemplatePopulation is a number of population units of one type.
type TemplatePopulation struct {
	Type PopulationType `json:"type"`
	Qty  int            `json:"qty"`
}

// TemplateFarmingGroup is a group of farm (FRM) units.
type TemplateFarmingGroup struct {
	Units     int `json:"units"`
	TechLevel int `json:"tech_level"`
}

// TemplateMiningGroup is a group of mine (MIN) units working one deposit on
// the home world.
type TemplateMiningGroup struct {
	// DepositNo is the deposit's number on the home world, as listed in the
	// home star template.
	DepositNo int `json:"deposit_no"`
	Units     int `json:"units"`
	TechLevel int `json:"tech_level"`
}

// TemplateFactoryGroup is a group of factory (FCT) units.
type TemplateFactoryGroup struct {
	Units     int `json:"units"`
	TechLevel int `json:"tech_level"`
	// Orders is what the group builds: a unit code, with a tech level for
	// units that have one, such as CNGD, STUN, AUT-1, or MIN-1.
	Orders string `json:"orders"`
}

// TemplateCache is a quantity of one resource.
type TemplateCache struct {
	Type CacheType `json:"type"`
	Qty  int       `json:"qty"`
}

// ordersPattern matches a factory group's orders: a unit code of 2 to 5
// capital letters, with an optional tech level.
var ordersPattern = regexp.MustCompile(`^[A-Z]{2,5}(-[1-9][0-9]?)?$`)

// DefaultEmpireTemplate returns the template that CreateEmpireTemplate writes.
func DefaultEmpireTemplate() EmpireTemplate {
	t, err := parseEmpireTemplate(defaultEmpireTemplate)
	if err != nil {
		panic(fmt.Sprintf("game: default empire template: %v", err))
	}
	return t
}

// Validate reports every way t breaks the rules for an empire's initial
// assets. Population and cache entries must have known types, no type twice,
// and positive quantities. There must be exactly EmpireFarmingGroups farming
// groups, EmpireMiningGroups mining groups, and EmpireFactoryGroups factory
// groups, each with positive units and a tech level of at least 1. A mining
// group's deposit number must be 1..MaxDepositsPerPlanet, and a factory
// group's orders must look like a unit code. The errors match
// ErrInvalidTemplate.
func (t EmpireTemplate) Validate() error {
	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%w: "+format, append([]any{ErrInvalidTemplate}, args...)...))
	}
	group := func(kind string, i, units, techLevel int) {
		if units < 1 {
			bad("%s group %d: units %d: want at least 1", kind, i+1, units)
		}
		if techLevel < 1 {
			bad("%s group %d: tech_level %d: want at least 1", kind, i+1, techLevel)
		}
	}

	seenPop := map[PopulationType]bool{}
	for i, p := range t.Population {
		switch {
		case !slices.Contains(populationTypes, p.Type):
			bad("population %d: invalid type %q", i+1, p.Type)
		case seenPop[p.Type]:
			bad("population %d: %s listed twice", i+1, p.Type)
		}
		seenPop[p.Type] = true
		if p.Qty < 1 {
			bad("population %d: %s qty %d: want at least 1", i+1, p.Type, p.Qty)
		}
	}

	if n := len(t.FarmingGroups); n != EmpireFarmingGroups {
		bad("%d farming groups: want %d", n, EmpireFarmingGroups)
	}
	for i, g := range t.FarmingGroups {
		group("farming", i, g.Units, g.TechLevel)
	}
	if n := len(t.MiningGroups); n != EmpireMiningGroups {
		bad("%d mining groups: want %d", n, EmpireMiningGroups)
	}
	for i, g := range t.MiningGroups {
		group("mining", i, g.Units, g.TechLevel)
		if g.DepositNo < 1 || g.DepositNo > MaxDepositsPerPlanet {
			bad("mining group %d: deposit_no %d: want 1..%d", i+1, g.DepositNo, MaxDepositsPerPlanet)
		}
	}
	if n := len(t.FactoryGroups); n != EmpireFactoryGroups {
		bad("%d factory groups: want %d", n, EmpireFactoryGroups)
	}
	for i, g := range t.FactoryGroups {
		group("factory", i, g.Units, g.TechLevel)
		if !ordersPattern.MatchString(g.Orders) {
			bad("factory group %d: orders %q: want a unit code such as CNGD or AUT-1", i+1, g.Orders)
		}
	}

	seenCache := map[CacheType]bool{}
	for i, c := range t.Cache {
		switch {
		case !slices.Contains(cacheTypes, c.Type):
			bad("cache %d: invalid type %q", i+1, c.Type)
		case seenCache[c.Type]:
			bad("cache %d: %s listed twice", i+1, c.Type)
		}
		seenCache[c.Type] = true
		if c.Qty < 1 {
			bad("cache %d: %s qty %d: want at least 1", i+1, c.Type, c.Qty)
		}
	}
	return errors.Join(errs...)
}

// CheckHomeWorld reports whether t's mining groups fit the home world in h:
// each must work a deposit the home world has.
func (t EmpireTemplate) CheckHomeWorld(h HomeStarTemplate) error {
	deposits := len(h.Orbits[HomeWorldOrbitNo-1].Planet.Deposits)
	var errs []error
	for i, g := range t.MiningGroups {
		if g.DepositNo > deposits {
			errs = append(errs, fmt.Errorf("%w: mining group %d: deposit_no %d: the home world has %d deposits",
				ErrInvalidTemplate, i+1, g.DepositNo, deposits))
		}
	}
	return errors.Join(errs...)
}

// CreateEmpireTemplate writes the default empire template to
// empire-template.json in dir, which must already hold a game, and returns
// it. The file is indented, with one item per line, because the game master
// edits it by hand. It fails with ErrTemplateExists if the template already
// exists, so it never overwrites the game master's changes.
func CreateEmpireTemplate(dir string) (EmpireTemplate, error) {
	if _, err := Load(dir); err != nil {
		return EmpireTemplate{}, err
	}
	t := DefaultEmpireTemplate()
	path := filepath.Join(dir, EmpireTemplateFileName)
	if err := writeNew(path, defaultEmpireTemplate); errors.Is(err, ErrGameExists) {
		return EmpireTemplate{}, fmt.Errorf("%s: %w", path, ErrTemplateExists)
	} else if err != nil {
		return EmpireTemplate{}, err
	}
	return t, nil
}

// LoadEmpireTemplate reads empire-template.json from dir and validates it.
func LoadEmpireTemplate(dir string) (EmpireTemplate, error) {
	path := filepath.Join(dir, EmpireTemplateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return EmpireTemplate{}, err
	}
	t, err := parseEmpireTemplate(data)
	if err != nil {
		return EmpireTemplate{}, fmt.Errorf("%s: %w", path, err)
	}
	return t, nil
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

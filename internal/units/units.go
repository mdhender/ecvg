// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package units is the unit catalogue: what each kind of unit is, whether it
// carries a tech level and needs assembly, what a lot of it takes to build,
// and its mass and volume. The numbers come from unit-tables.md, which the
// tests check the catalogue against.
package units

import (
	"slices"
	"strings"
)

// The unit codes.
const (
	ACFT = "ACFT" // assault craft
	AMSL = "AMSL" // anti-missile
	AUTO = "AUTO" // automation
	AWPN = "AWPN" // assault weapons
	CNGD = "CNGD" // consumer goods
	CSUP = "CSUP" // combat supplies
	CWRK = "CWRK" // construction worker
	ESHD = "ESHD" // energy shield
	EWPN = "EWPN" // energy weapon
	FACT = "FACT" // factory
	FARM = "FARM" // farm
	FOOD = "FOOD" // food
	FUEL = "FUEL" // fuel
	GOLD = "GOLD" // gold
	HDRV = "HDRV" // hyper drive
	LAB  = "LAB"  // laboratory
	LFSU = "LFSU" // life support
	MINE = "MINE" // mine
	MLNC = "MLNC" // missile launcher
	MRBT = "MRBT" // military robot
	MSSL = "MSSL" // missile
	MTL  = "MTL"  // metallics
	NMTL = "NMTL" // non-metallics
	POL  = "POL"  // police
	POWR = "POWR" // power plant
	PRO  = "PRO"  // professionals
	PROB = "PROB" // probe
	PRTO = "PRTO" // prototype
	RBL  = "RBL"  // rebel
	RP   = "RP"   // research point
	SDRV = "SDRV" // space drive
	SENS = "SENS" // sensor
	SLD  = "SLD"  // soldiers
	SPAG = "SPAG" // special agent
	SPY  = "SPY"  // spy
	STRC = "STRC" // structure
	STRL = "STRL" // light structure
	TRNE = "TRNE" // trainee
	TRNS = "TRNS" // transport
	UEM  = "UEM"  // unemployables
	USK  = "USK"  // unskilled workers
	WRKR = "WRKR" // worker
)

// Class groups the units by what they are for.
type Class string

const (
	Ammunition     Class = "Ammunition"
	Cadre          Class = "Cadre"
	Commodity      Class = "Commodity"
	Infrastructure Class = "Infrastructure"
	Living         Class = "Living"
	Propulsion     Class = "Propulsion"
	Recon          Class = "Recon"
	Resource       Class = "Resource"
	Static         Class = "Static"
	Structural     Class = "Structural"
	Technology     Class = "Technology"
	Transportation Class = "Transportation"
	Weaponry       Class = "Weaponry"
)

// The tech levels of a unit that carries one.
const (
	MinTechLevel = 1
	MaxTechLevel = 10
)

// amount is a number that may depend on the tech level and may be a half:
// (perTL x TL + base) / div.
type amount struct {
	perTL, base, div int
}

// at returns the numerator of the amount at tech level tl, over a.div.
func (a amount) at(tl int) int {
	return a.perTL*tl + a.base
}

func fixed(n int) amount        { return amount{0, n, 1} }
func perTL(n int) amount        { return amount{n, 0, 1} }
func linear(n, base int) amount { return amount{n, base, 1} }
func halves(n, base int) amount { return amount{n, base, 2} }

// Def describes one kind of unit.
type Def struct {
	Code  string
	Name  string
	Class Class
	// HasTechLevel reports whether the unit carries a tech level,
	// MinTechLevel..MaxTechLevel. A unit that doesn't is at tech level 0.
	HasTechLevel bool
	// RequiresAssembly reports whether the unit must be assembled before it
	// can be used.
	RequiresAssembly bool
	// Manufactured reports whether factories build the unit from MTL and
	// NMTL. See Inputs.
	Manufactured bool

	lot, lotPerTL2       int // lot size: lot, or lotPerTL2 x TL^2
	mtl, nmtl            amount
	mass, volume, stowed amount
}

// Population reports whether the unit is people: Living or a Cadre. They are
// counted in population units and are never stowed.
func (d Def) Population() bool {
	return d.Class == Living || d.Class == Cadre
}

// DepotAllowed reports whether the unit can be kept in an open-air colony's
// depot. Weaponry, commodities, static units, and technology must be in the
// entity's cargo, and people aren't stored at all.
func (d Def) DepotAllowed() bool {
	switch d.Class {
	case Weaponry, Commodity, Static, Technology, Living, Cadre:
		return false
	}
	return true
}

// ValidTechLevel reports whether a unit of this kind can be at tech level tl:
// MinTechLevel..MaxTechLevel if it carries a tech level, otherwise only 0.
func (d Def) ValidTechLevel(tl int) bool {
	if !d.HasTechLevel {
		return tl == 0
	}
	return MinTechLevel <= tl && tl <= MaxTechLevel
}

// LotSize returns the number of units a single build adds to inventory.
func (d Def) LotSize(tl int) int {
	if d.lotPerTL2 != 0 {
		return d.lotPerTL2 * tl * tl
	}
	return d.lot
}

// Inputs returns the MTL and NMTL it takes to build lots lots of the unit.
// Units are built in whole lots. ok is false if the unit isn't manufactured,
// or if what it takes isn't settled yet (PRTO).
func (d Def) Inputs(tl, lots int) (mtl, nmtl int, ok bool) {
	if !d.Manufactured || d.mtl.div == 0 {
		return 0, 0, false
	}
	return d.mtl.at(tl) * lots, d.nmtl.at(tl) * lots, true
}

// Mass returns the mass of qty units, in MU, rounded up.
func (d Def) Mass(tl, qty int) int {
	return d.total(d.mass, tl, qty)
}

// Volume returns the space qty assembled (or never stowed) units take, in VU,
// rounded up.
func (d Def) Volume(tl, qty int) int {
	return d.total(d.volume, tl, qty)
}

// StowedVolume returns the space qty unassembled units take in storage, in
// VU, rounded up.
func (d Def) StowedVolume(tl, qty int) int {
	return d.total(d.stowed, tl, qty)
}

// total returns qty units' share of the per-lot amount a. Rounding is done
// once, on the total, and goes against the player: up.
func (d Def) total(a amount, tl, qty int) int {
	if a.div == 0 || qty <= 0 {
		return 0
	}
	div := a.div * d.LotSize(tl)
	return (qty*a.at(tl) + div - 1) / div
}

// Lookup returns the definition of the unit with the given code.
func Lookup(code string) (Def, bool) {
	i, ok := slices.BinarySearchFunc(catalogue, code, func(d Def, code string) int {
		return strings.Compare(d.Code, code)
	})
	if !ok {
		return Def{}, false
	}
	return catalogue[i], true
}

// All returns every unit's definition, in code order.
func All() []Def {
	return slices.Clone(catalogue)
}

var catalogue = build()

func build() []Def {
	// A unit built in lots of one whose mass, volume, and stowed volume are
	// each a multiple of its tech level.
	made := func(code, name string, class Class, asm bool, mtl, nmtl, mass, volume, stowed amount) Def {
		return Def{
			Code: code, Name: name, Class: class,
			HasTechLevel: true, RequiresAssembly: asm, Manufactured: true,
			lot: 1, mtl: mtl, nmtl: nmtl, mass: mass, volume: volume, stowed: stowed,
		}
	}
	// A resource or FOOD: not manufactured, no tech level.
	raw := func(code, name string, class Class, mass, volume amount) Def {
		return Def{Code: code, Name: name, Class: class, lot: 1, mass: mass, volume: volume, stowed: volume}
	}
	// People: 1 MU and 1 VU per population unit, never stowed.
	people := func(code, name string, class Class) Def {
		return Def{Code: code, Name: name, Class: class, lot: 1, mass: fixed(1), volume: fixed(1), stowed: fixed(1)}
	}
	lots := func(d Def, lot int) Def {
		d.lot = lot
		return d
	}
	noTL := func(d Def) Def {
		d.HasTechLevel = false
		return d
	}

	prob := made(PROB, "Probe", Recon, false, fixed(200), fixed(300), fixed(500), fixed(500), fixed(500))
	prob.lot, prob.lotPerTL2 = 0, 10

	// What a prototype takes to build isn't settled.
	prto := made(PRTO, "Prototype", Technology, false, amount{}, amount{}, perTL(1), perTL(3), perTL(3))

	defs := []Def{
		made(ACFT, "Assault Craft", Weaponry, false, perTL(3), perTL(2), perTL(5), perTL(5), halves(5, 0)),
		made(AMSL, "Anti-Missile", Ammunition, false, perTL(2), perTL(2), perTL(4), perTL(4), perTL(2)),
		made(AUTO, "Automation", Infrastructure, true, perTL(2), perTL(2), perTL(4), perTL(4), perTL(2)),
		made(AWPN, "Assault Weapons", Weaponry, false, perTL(1), perTL(1), perTL(2), perTL(2), perTL(1)),
		noTL(lots(made(CNGD, "Consumer Goods", Commodity, false, fixed(2), fixed(4), fixed(6), fixed(3), fixed(3)), 10)),
		noTL(lots(made(CSUP, "Combat Supplies", Ammunition, false, fixed(1), fixed(1), fixed(2), fixed(2), fixed(2)), 50)),
		people(CWRK, "Construction Worker", Cadre),
		made(ESHD, "Energy Shield", Weaponry, true, perTL(10), perTL(10), perTL(20), perTL(20), perTL(10)),
		made(EWPN, "Energy Weapon", Weaponry, true, perTL(5), perTL(5), perTL(10), perTL(10), perTL(5)),
		made(FACT, "Factory", Infrastructure, true, linear(1, 8), linear(1, 4), linear(2, 12), linear(2, 12), linear(1, 6)),
		made(FARM, "Farm", Infrastructure, true, linear(1, 4), linear(1, 2), linear(2, 6), linear(2, 6), linear(1, 3)),
		raw(FOOD, "Food", Commodity, fixed(6), fixed(3)),
		raw(FUEL, "Fuel", Resource, fixed(1), halves(0, 1)),
		// Gold is only for trading. It is stored like MTL.
		raw(GOLD, "Gold", Resource, fixed(1), halves(0, 1)),
		made(HDRV, "Hyper Drive", Propulsion, true, perTL(25), perTL(20), perTL(45), perTL(45), halves(45, 0)),
		made(LAB, "Laboratory", Infrastructure, true, linear(1, 5), linear(1, 3), linear(2, 8), linear(2, 8), linear(1, 4)),
		made(LFSU, "Life Support", Static, true, perTL(3), perTL(5), perTL(8), perTL(8), perTL(4)),
		made(MINE, "Mine", Infrastructure, true, linear(1, 5), linear(1, 5), linear(2, 10), linear(2, 10), linear(1, 5)),
		made(MLNC, "Missile Launcher", Weaponry, true, perTL(15), perTL(10), perTL(25), perTL(25), halves(25, 0)),
		made(MRBT, "Military Robot", Weaponry, false, linear(1, 10), linear(1, 10), linear(2, 20), linear(2, 20), linear(1, 10)),
		made(MSSL, "Missile", Ammunition, false, perTL(2), perTL(2), perTL(4), perTL(4), perTL(4)),
		raw(MTL, "Metallics", Resource, fixed(1), halves(0, 1)),
		raw(NMTL, "Non-Metallics", Resource, fixed(1), halves(0, 1)),
		people(POL, "Police", Cadre),
		made(POWR, "Power Plant", Infrastructure, true, linear(1, 5), linear(1, 5), linear(2, 10), linear(2, 10), linear(1, 5)),
		people(PRO, "Professionals", Living),
		prob,
		prto,
		people(RBL, "Rebel", Cadre),
		// A research point is bookkeeping, with no mass or volume.
		{Code: RP, Name: "Research Point", Class: Technology, lot: 1},
		made(SDRV, "Space Drive", Propulsion, true, perTL(15), perTL(10), perTL(25), perTL(25), halves(25, 0)),
		made(SENS, "Sensor", Recon, true, perTL(1000), perTL(2000), perTL(3000), perTL(3000), perTL(1500)),
		people(SLD, "Soldiers", Living),
		people(SPAG, "Special Agent", Cadre),
		people(SPY, "Spy", Cadre),
		lots(made(STRC, "Structure", Structural, true, perTL(7), perTL(3), perTL(10), perTL(10), perTL(10)), 10),
		lots(made(STRL, "Light Structure", Structural, true, perTL(7), perTL(3), perTL(10), perTL(10), perTL(10)), 100),
		people(TRNE, "Trainee", Cadre),
		made(TRNS, "Transport", Transportation, false, perTL(3), perTL(1), perTL(4), perTL(4), perTL(4)),
		people(UEM, "Unemployables", Living),
		people(USK, "Unskilled Workers", Living),
		people(WRKR, "Worker", Cadre),
	}
	slices.SortFunc(defs, func(a, b Def) int {
		return strings.Compare(a.Code, b.Code)
	})
	return defs
}

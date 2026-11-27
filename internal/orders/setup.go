// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package orders

import (
	"fmt"
	"strings"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

// Setup is the order that has an entity set up a new ship or colony at its
// own location (manual S5.3). It takes more than one line:
//
//	setup [Quantity] EntityType from EntityID [using Quantity CWRK [each]]
//	Quantity Unit [stowed]
//	Quantity FACT-n producing Unit
//	Quantity MINE-n working DepositNo
//	Quantity PopulationUnit
//	end
//
// The lines may be separated by ";" instead of the end of a line. EntityType
// is CSHP, COPN, CENC, or CORB. The Quantity of the first line is the number
// of identical ships to set up, and only a CSHP may have one. There is a line
// for each kind of unit to install, at least one, in the order of their
// importance; a unit that is assembled is assembled in the new entity unless
// its line ends with stowed.
type Setup struct {
	// Qty is the number of entities to set up: 1, unless the order says.
	Qty  int
	Type game.EntityType
	// From is the ID of the entity that does the set up.
	From int
	// Items are the units to install, in the order listed.
	Items []SetupItem
	// MaxCWRK is the most CWRK of the entity From that may help with the
	// assembly, or 0 if the order sets no cap. It is for each entity set up
	// if Each is set, and otherwise for all of them together.
	MaxCWRK int
	Each    bool
}

// SetupItem is one line of a Setup order.
type SetupItem struct {
	// Unit is the units to install. Its Type may be a population unit (UEM,
	// USK, PRO, SLD) or a cadre (CWRK, SPY).
	Unit game.Unit
	// Race is as for Draft, for people.
	Race int
	// Stowed is set if the units are to stay in storage.
	Stowed bool
	// Producing is what a new factory group made of the FACT units is to
	// make; its Qty is 0.
	Producing game.Unit
	// DepositNo is the deposit that a new mining group made of the MINE units
	// works.
	DepositNo int
}

func (i SetupItem) String() string {
	var sb strings.Builder
	if d, _ := units.Lookup(i.Unit.Type); d.Population() {
		fmt.Fprintf(&sb, "%d %s", i.Unit.Qty, popName(i.Unit.Type, i.Race))
	} else {
		fmt.Fprintf(&sb, "%d %s", i.Unit.Qty, unitName(i.Unit))
	}
	switch {
	case i.Stowed:
		sb.WriteString(" stowed")
	case i.Producing.Type != "":
		sb.WriteString(" producing " + unitName(i.Producing))
	case i.DepositNo != 0:
		fmt.Fprintf(&sb, " working %d", i.DepositNo)
	}
	return sb.String()
}

func (o Setup) String() string {
	var sb strings.Builder
	sb.WriteString("setup ")
	if o.Qty > 1 {
		fmt.Fprintf(&sb, "%d ", o.Qty)
	}
	fmt.Fprintf(&sb, "%s from %d%s", o.Type, o.From, using(o.MaxCWRK))
	if o.Each {
		sb.WriteString(" each")
	}
	for _, i := range o.Items {
		sb.WriteString("; " + i.String())
	}
	sb.WriteString("; end")
	return sb.String()
}

const setupUsage = `want:
  setup [Quantity] EntityType from EntityID [using Quantity CWRK [each]]
  Quantity Unit [stowed]          (a line for each kind of unit, at least one)
  end
EntityType is CSHP, COPN, CENC, or CORB; only CSHP may have a Quantity`

const setupItemUsage = `want one of:
  Quantity Unit [stowed]
  Quantity FACT-n producing Unit
  Quantity MINE-n working DepositNo
  Quantity PopulationUnit         (UEM, USK, PRO, SLD, CWRK, or SPY)`

// setupBlock is a setup order as it is read, line by line.
type setupBlock struct {
	// no is the number of the order's first line.
	no    int
	texts []string
	order Setup
	// err is the first thing found wrong with the order.
	err error
}

// fail records what is wrong with the order, if nothing was before.
func (b *setupBlock) fail(err error) {
	if b.err == nil {
		b.err = err
	}
}

// line returns the order as a Line.
func (b *setupBlock) line() Line {
	l := Line{No: b.no, Text: strings.Join(b.texts, "; "), Err: b.err}
	if l.Err == nil && len(b.order.Items) == 0 {
		l.Err = fmt.Errorf("%w: the order lists no units: %s", ErrInvalidOrder, setupUsage)
	}
	if l.Err == nil {
		l.Order = b.order
	}
	return l
}

// startsItem reports whether t can start a line that lists units: it starts
// with a digit, as a quantity does.
func startsItem(t Token) bool {
	return !t.Quoted && t.Text != "" && t.Text[0] >= '0' && t.Text[0] <= '9'
}

// header parses the order's first line.
func (b *setupBlock) header(tokens []Token) {
	b.texts = append(b.texts, joinTokens(tokens))
	usage := fmt.Errorf("%w: %s", ErrInvalidOrder, setupUsage)
	o := &b.order
	o.Qty = 1
	rest := tokens[1:]
	hasQty := len(rest) > 0 && startsItem(rest[0])
	if hasQty {
		var ok bool
		if o.Qty, ok = quantity(rest[0]); !ok || o.Qty < 1 {
			b.fail(fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, rest[0].Text))
			return
		}
		rest = rest[1:]
	}
	if len(rest) < 3 || !rest[1].Is("from") {
		b.fail(usage)
		return
	}
	switch t := game.EntityType(strings.ToUpper(rest[0].Text)); {
	case rest[0].Quoted, t != game.CSHP && t != game.COPN && t != game.CENC && t != game.CORB:
		b.fail(fmt.Errorf("%w: %q: want CSHP, COPN, CENC, or CORB", ErrInvalidEntityType, rest[0].Text))
		return
	default:
		o.Type = t
	}
	if hasQty && o.Type != game.CSHP {
		b.fail(fmt.Errorf("%w: only one colony is set up at a time: a quantity is for CSHP only", ErrInvalidQuantity))
		return
	}
	var err error
	if o.From, err = entityID(rest[2]); err != nil {
		b.fail(err)
		return
	}
	rest = rest[3:]
	if len(rest) == 0 {
		return
	}
	if n := len(rest); rest[n-1].Is("each") {
		o.Each = true
		rest = rest[:n-1]
	}
	if len(rest) != 3 || !rest[0].Is("using") {
		b.fail(usage)
		return
	}
	if o.MaxCWRK, _, err = usingClause(rest); err != nil {
		b.fail(err)
	}
}

// item parses a line that lists units, which is line no of the file.
func (b *setupBlock) item(no int, tokens []Token) {
	b.texts = append(b.texts, joinTokens(tokens))
	item, err := parseSetupItem(tokens)
	if err != nil {
		b.fail(fmt.Errorf("line %d: %w", no, err))
		return
	}
	b.order.Items = append(b.order.Items, item)
}

func parseSetupItem(tokens []Token) (SetupItem, error) {
	usage := fmt.Errorf("%w: %s", ErrInvalidOrder, setupItemUsage)
	if len(tokens) < 2 || len(tokens) > 4 {
		return SetupItem{}, usage
	}
	var item SetupItem
	// People: a population unit or one of the cadres that can be moved.
	u := tokens[1]
	if code, race, hasRace := strings.Cut(u.Text, "-"); strings.EqualFold(code, "CWKR") {
		u.Text = units.CWRK
		if hasRace {
			u.Text += "-" + race
		}
	}
	if code, race, ok := popCode(u); ok {
		qty, ok := quantity(tokens[0])
		if !ok || qty < 1 {
			return SetupItem{}, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, tokens[0].Text)
		}
		switch code {
		case units.UEM, units.USK, units.PRO, units.SLD, units.CWRK, units.SPY:
		default:
			return SetupItem{}, fmt.Errorf("%w: %q: want UEM, USK, PRO, SLD, CWRK, or SPY", ErrInvalidUnit, tokens[1].Text)
		}
		if len(tokens) != 2 {
			return SetupItem{}, fmt.Errorf("%w: people are never stowed: want Quantity %s", ErrInvalidOrder, code)
		}
		item.Unit, item.Race = game.Unit{Type: code, Qty: qty}, race
		return item, nil
	}
	var err error
	if item.Unit, err = quantityOf(tokens[0], tokens[1], ""); err != nil {
		return SetupItem{}, err
	}
	code := item.Unit.Type
	switch args := tokens[2:]; {
	case len(args) == 1 && args[0].Is("stowed"):
		item.Stowed = true
	case len(args) == 2 && args[0].Is("producing") && code == units.FACT:
		if item.Producing, err = unitOf(args[1], ""); err != nil {
			return SetupItem{}, err
		}
		if d, _ := units.Lookup(item.Producing.Type); !d.FactoryMade() {
			return SetupItem{}, fmt.Errorf("%w: %q: %s", ErrInvalidUnit, args[1].Text, NotFactoryMade(item.Producing.Type))
		}
	case len(args) == 2 && args[0].Is("working") && code == units.MINE:
		var ok bool
		if item.DepositNo, ok = number(args[1]); !ok || item.DepositNo < 1 {
			return SetupItem{}, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidDepositNo, args[1].Text)
		}
	case len(args) != 0:
		return SetupItem{}, usage
	case code == units.FACT:
		return SetupItem{}, fmt.Errorf("%w: say what the factory group is to make, or that the units stay in storage: Quantity FACT-n producing Unit, or Quantity FACT-n stowed", ErrInvalidOrder)
	case code == units.MINE:
		return SetupItem{}, fmt.Errorf("%w: say which deposit the mining group is to work, or that the units stay in storage: Quantity MINE-n working DepositNo, or Quantity MINE-n stowed", ErrInvalidOrder)
	}
	return item, nil
}

// CancelSetup is the order "cancel setup EntityID": the set up of the entity
// with ID EntityID, which is still being set up, is abandoned.
type CancelSetup struct {
	EntityID int
}

func (o CancelSetup) String() string {
	return fmt.Sprintf("cancel setup %d", o.EntityID)
}

func parseCancelSetup(tokens []Token) (Order, error) {
	if len(tokens) != 3 {
		return nil, fmt.Errorf("%w: want: cancel setup EntityID", ErrInvalidOrder)
	}
	id, err := entityID(tokens[2])
	if err != nil {
		return nil, err
	}
	return CancelSetup{EntityID: id}, nil
}

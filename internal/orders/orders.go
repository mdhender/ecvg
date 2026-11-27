// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package orders reads the orders players return for a turn.
//
// Orders are plain text, one order per line, and an order can't span lines.
// Keywords ignore case, white space between tokens is compressed, and a "#"
// or "//" that starts a token starts a comment that runs to the end of the
// line (see Lex). Blank lines are ignored.
//
// The orders:
//
//	set rations for EntityID to PercentageAmount
//	draft Quantity UnitOrCadre from EntityID
//	disband Quantity UnitOrCadre from EntityID
//	assemble factory-group on EntityID with Quantity FactoryUnit producing Unit
//	assemble farming-group on EntityID with Quantity FarmUnit
//	assemble mining-group on EntityID with Quantity MiningUnit working DepositNo
//	assemble factory-group GroupNo on EntityID add Quantity FactoryUnit
//	assemble mining-group GroupNo on EntityID add Quantity MiningUnit
//	assemble Quantity Unit on EntityID
//	disassemble Quantity Unit on EntityID
//	scrap Quantity Unit on EntityID
//	update factory-group GroupNo of EntityID retool
//	update factory-group GroupNo of EntityID produce Unit
//	update GroupKind-group GroupNo of EntityID shutdown (Quantity | PercentageAmount)
//	update GroupKind-group GroupNo of EntityID restart (Quantity | PercentageAmount)
//
// Any assemble, disassemble, or scrap order may end with "using Quantity CWRK", which caps the CWRK that work
// on it in a turn. CWKR is read as CWRK there, and it may have a race ID too.
//
// EntityID is the ID of an entity. PercentageAmount is a whole number with a
// "%" joined to it: "50%", not "50 %". For rations it is from game.MinRations
// to game.MaxRations, and for shutdown and restart, where it is a percentage
// of the units of the group, from 1 to 100. GroupKind is farming, mining, or factory, and GroupNo
// is the number of a group of that kind. Quantity is a whole number, 1 or
// more, which may have commas between its thousands: "3600" or "3,600".
// UnitOrCadre is the code of a population unit or cadre, such as SLD or CWRK.
// It may have a race ID joined to it by a hyphen, such as SLD-3; without one,
// or with race 0, it is of the race of the empire that gives the order.
// A unit that has a tech level is written with it, joined by a hyphen:
// FactoryUnit is FACT-1 to FACT-10, FarmUnit FARM-1 to FARM-10, and
// MiningUnit MINE-1 to MINE-10. A unit that has none is its code alone, such
// as CNGD. Every unit of a group is of one tech level.
package orders

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/units"
)

const (
	ErrUnknownOrder      = cerrs.Error("unknown order")
	ErrInvalidOrder      = cerrs.Error("invalid order")
	ErrInvalidEntityID   = cerrs.Error("invalid entity ID")
	ErrInvalidPercentage = cerrs.Error("invalid percentage")
	ErrInvalidGroupKind  = cerrs.Error("invalid group kind")
	ErrInvalidGroupNo    = cerrs.Error("invalid group number")
	ErrInvalidQuantity   = cerrs.Error("invalid quantity")
	ErrInvalidUnit       = cerrs.Error("invalid unit")
	ErrInvalidDepositNo  = cerrs.Error("invalid deposit number")
)

// Order is one order. Its type says which: SetRations, UpdateGroup, Draft,
// Disband, AssembleGroup, or UnitWork.
type Order interface {
	// String returns the order as it would be written.
	String() string
}

// SetRations is the order "set rations for EntityID to PercentageAmount". It
// sets the rations of the entity with ID EntityID to Percent percent.
type SetRations struct {
	EntityID int
	Percent  int
}

func (o SetRations) String() string {
	return fmt.Sprintf("set rations for %d to %d%%", o.EntityID, o.Percent)
}

// UpdateGroup is the order "update GroupKind-group GroupNo of EntityID
// Action": it changes what the group of that kind numbered GroupNo in the
// entity with ID EntityID does. A factory group retools, or is told what to
// produce. A group of any kind shuts down some of its units, or restarts
// some that are shut down.
type UpdateGroup struct {
	EntityID int
	Kind     game.GroupKind
	GroupNo  int
	Action   game.GroupAction
	// Producing is what a produce order has the group make; its Qty is 0.
	Producing game.Unit
	// A shutdown or restart order has either Qty, a number of units, or
	// Percent, a percentage of the units of the group; the other is 0.
	Qty     int
	Percent int
}

func (o UpdateGroup) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "update %s-group %d of %d %s", o.Kind, o.GroupNo, o.EntityID, o.Action)
	switch {
	case o.Action == game.ActionProduce:
		sb.WriteString(" " + unitName(o.Producing))
	case o.Percent != 0:
		fmt.Fprintf(&sb, " %d%%", o.Percent)
	case o.Qty != 0:
		fmt.Fprintf(&sb, " %d", o.Qty)
	}
	return sb.String()
}

// Draft is the order "draft Quantity UnitOrCadre from EntityID". It drafts
// Qty population units of the entity with ID EntityID into the unit or cadre
// with the code Unit.
type Draft struct {
	EntityID int
	Unit     string
	// Race is the ID of the race of the people, or 0 for the race of the
	// empire that gives the order. Nothing uses it yet.
	Race int
	Qty  int
}

func (o Draft) String() string {
	return fmt.Sprintf("draft %d %s from %d", o.Qty, popName(o.Unit, o.Race), o.EntityID)
}

// Disband is the order "disband Quantity UnitOrCadre from EntityID". It
// disbands Qty of the unit or cadre with the code Unit in the entity with ID
// EntityID.
type Disband struct {
	EntityID int
	Unit     string
	// Race is as for Draft.
	Race int
	Qty  int
}

func (o Disband) String() string {
	return fmt.Sprintf("disband %d %s from %d", o.Qty, popName(o.Unit, o.Race), o.EntityID)
}

// AssembleGroup is the order that assembles units into a factory, farming,
// or mining group of the entity with ID EntityID: a new group, if GroupNo is
// 0, or more units for the group numbered GroupNo.
type AssembleGroup struct {
	EntityID int
	Kind     game.GroupKind
	GroupNo  int
	// Unit is the units to assemble. Every unit of a group is of one tech
	// level.
	Unit game.Unit
	// DepositNo is the deposit a new mining group works: its number on the
	// entity's planet.
	DepositNo int
	// Producing is what a new factory group is to make, which the order must
	// say; its Qty is 0.
	Producing game.Unit
	// MaxCWRK is the most CWRK that may work on the order in a turn, or 0
	// if the order sets no cap.
	MaxCWRK int
}

func (o AssembleGroup) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "assemble %s-group ", o.Kind)
	if o.GroupNo != 0 {
		fmt.Fprintf(&sb, "%d on %d add ", o.GroupNo, o.EntityID)
	} else {
		fmt.Fprintf(&sb, "on %d with ", o.EntityID)
	}
	fmt.Fprintf(&sb, "%d %s", o.Unit.Qty, unitName(o.Unit))
	if o.Producing.Type != "" {
		fmt.Fprintf(&sb, " producing %s", unitName(o.Producing))
	}
	if o.DepositNo != 0 {
		fmt.Fprintf(&sb, " working %d", o.DepositNo)
	}
	sb.WriteString(using(o.MaxCWRK))
	return sb.String()
}

// UnitWork is the order "assemble Quantity Unit on EntityID", or the same
// with disassemble or scrap, as Action says. It has the CWRK of the entity
// with ID EntityID assemble units that are in its storage, disassemble units
// it has assembled, or scrap units in its storage for their MTL and NMTL.
type UnitWork struct {
	Action   game.Work
	EntityID int
	Unit     game.Unit
	// MaxCWRK is as for AssembleGroup.
	MaxCWRK int
}

func (o UnitWork) String() string {
	return fmt.Sprintf("%s %d %s on %d%s", o.Action.Verb(), o.Unit.Qty, unitName(o.Unit), o.EntityID, using(o.MaxCWRK))
}

// using returns the clause that caps an order's CWRK, if it has a cap.
func using(maxCWRK int) string {
	if maxCWRK == 0 {
		return ""
	}
	return fmt.Sprintf(" using %d CWRK", maxCWRK)
}

// popName returns a population unit or cadre as orders write it: SLD, or
// SLD-3 if the order names a race.
func popName(code string, race int) string {
	if race == 0 {
		return code
	}
	return fmt.Sprintf("%s-%d", code, race)
}

// popCode returns the code and race ID of a token that names a population
// unit or cadre: its code and, optionally, a race ID joined to it by a
// hyphen, such as SLD or SLD-3. Case is ignored. The race is 0 if the token
// has none, and 0 stands for the race of the empire that gives the order.
func popCode(t Token) (code string, race int, ok bool) {
	code, id, hasRace := strings.Cut(strings.ToUpper(t.Text), "-")
	if d, found := units.Lookup(code); t.Quoted || !found || !d.Population() {
		return "", 0, false
	}
	if hasRace {
		if race, ok = number(Token{Text: id}); !ok {
			return "", 0, false
		}
	}
	return code, race, true
}

// unitName returns the unit as orders write it: FACT-10, or CNGD.
func unitName(u game.Unit) string {
	if u.TechLevel == 0 {
		return u.Type
	}
	return fmt.Sprintf("%s-%d", u.Type, u.TechLevel)
}

// Line is one line of an orders file that holds an order.
type Line struct {
	// No is the line's number in the file, from 1.
	No int
	// Text is the line as written, without leading or trailing white space.
	Text string
	// Order is the line's order, or nil if Err is not.
	Order Order
	// Err says why the line is not a valid order.
	Err error
}

// maxLineLength bounds the length of a line of orders, in bytes.
const maxLineLength = 1 << 20

// Parse reads orders from r. It returns a Line for every line that is not
// blank or only a comment; a line that is not a valid order has its Err set.
// The error it returns is one from reading r.
func Parse(r io.Reader) ([]Line, error) {
	var lines []Line
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, maxLineLength)
	for no := 1; sc.Scan(); no++ {
		line := Line{No: no, Text: strings.TrimSpace(sc.Text())}
		tokens, err := Lex(line.Text)
		if err == nil && len(tokens) == 0 {
			continue
		}
		if err == nil {
			line.Order, err = parseOrder(tokens)
		}
		line.Err = err
		lines = append(lines, line)
	}
	return lines, sc.Err()
}

// parseOrder parses the tokens of one line, which has at least one.
func parseOrder(tokens []Token) (Order, error) {
	switch {
	case len(tokens) >= 2 && tokens[0].Is("set") && tokens[1].Is("rations"):
		return parseSetRations(tokens)
	case tokens[0].Is("update"):
		return parseUpdateGroup(tokens)
	case tokens[0].Is("draft"):
		return parseDraft(tokens, "draft")
	case tokens[0].Is("disband"):
		return parseDraft(tokens, "disband")
	case tokens[0].Is("assemble"):
		return parseAssemble(tokens)
	case tokens[0].Is("disassemble"):
		return parseUnitWork(tokens, game.WorkDisassemble)
	case tokens[0].Is("scrap"):
		return parseUnitWork(tokens, game.WorkScrap)
	}
	return nil, ErrUnknownOrder
}

const setRationsUsage = "want: set rations for EntityID to PercentageAmount"

func parseSetRations(tokens []Token) (Order, error) {
	if len(tokens) != 6 || !tokens[2].Is("for") || !tokens[4].Is("to") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOrder, setRationsUsage)
	}
	id, ok := number(tokens[3])
	if !ok || id < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidEntityID, tokens[3].Text)
	}
	pct, err := percentage(tokens[5], game.MinRations, game.MaxRations)
	if err != nil {
		return nil, err
	}
	return SetRations{EntityID: id, Percent: pct}, nil
}

const updateGroupUsage = `want one of:
  update factory-group GroupNo of EntityID retool
  update factory-group GroupNo of EntityID produce Unit
  update GroupKind-group GroupNo of EntityID shutdown Quantity
  update GroupKind-group GroupNo of EntityID shutdown PercentageAmount
  update GroupKind-group GroupNo of EntityID restart Quantity
  update GroupKind-group GroupNo of EntityID restart PercentageAmount
GroupKind is factory, farming, or mining`

func parseUpdateGroup(tokens []Token) (Order, error) {
	usage := fmt.Errorf("%w: %s", ErrInvalidOrder, updateGroupUsage)
	if len(tokens) < 2 {
		return nil, usage
	}
	o := UpdateGroup{}
	for k := range groupUnit {
		if tokens[1].Is(string(k) + "-group") {
			o.Kind = k
		}
	}
	if o.Kind == "" {
		return nil, fmt.Errorf("%w: %q: want factory-group, farming-group, or mining-group", ErrInvalidGroupKind, tokens[1].Text)
	}
	if len(tokens) < 6 || !tokens[3].Is("of") {
		return nil, usage
	}
	var ok bool
	if o.GroupNo, ok = number(tokens[2]); !ok || o.GroupNo < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidGroupNo, tokens[2].Text)
	}
	var err error
	if o.EntityID, err = entityID(tokens[4]); err != nil {
		return nil, err
	}
	args := tokens[6:]
	switch action := tokens[5]; {
	case action.Is("retool") || action.Is("produce"):
		if o.Kind != game.KindFactory {
			return nil, fmt.Errorf("%w: only a factory group can %s", ErrInvalidOrder, strings.ToLower(action.Text))
		}
		if action.Is("retool") {
			if len(args) != 0 {
				return nil, usage
			}
			o.Action = game.ActionRetool
			return o, nil
		}
		if len(args) != 1 {
			return nil, usage
		}
		o.Action = game.ActionProduce
		if o.Producing, err = unitOf(args[0], ""); err != nil {
			return nil, err
		}
		if d, _ := units.Lookup(o.Producing.Type); !d.FactoryMade() {
			return nil, fmt.Errorf("%w: %q: factories don't make it", ErrInvalidUnit, args[0].Text)
		}
		return o, nil
	case action.Is("shutdown") || action.Is("restart"):
		o.Action = game.ActionShutDown
		if action.Is("restart") {
			o.Action = game.ActionRestart
		}
		if len(args) != 1 {
			return nil, usage
		}
		if strings.HasSuffix(args[0].Text, "%") {
			if o.Percent, err = percentage(args[0], 1, 100); err != nil {
				return nil, err
			}
			return o, nil
		}
		if o.Qty, ok = quantity(args[0]); !ok || o.Qty < 1 {
			return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600, or a percentage, such as 50%%", ErrInvalidQuantity, args[0].Text)
		}
		return o, nil
	}
	return nil, usage
}

// parseDraft parses a draft or a disband order; verb says which.
func parseDraft(tokens []Token, verb string) (Order, error) {
	if len(tokens) != 5 || !tokens[3].Is("from") {
		return nil, fmt.Errorf("%w: want: %s Quantity UnitOrCadre from EntityID", ErrInvalidOrder, verb)
	}
	qty, ok := quantity(tokens[1])
	if !ok || qty < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, tokens[1].Text)
	}
	code, race, ok := popCode(tokens[2])
	if !ok {
		return nil, fmt.Errorf("%w: %q: want a population unit or cadre, such as SLD or CWRK, which may have a race ID joined to it, such as SLD-3", ErrInvalidUnit, tokens[2].Text)
	}
	id, ok := number(tokens[4])
	if !ok || id < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidEntityID, tokens[4].Text)
	}
	if verb == "disband" {
		return Disband{EntityID: id, Unit: code, Race: race, Qty: qty}, nil
	}
	return Draft{EntityID: id, Unit: code, Race: race, Qty: qty}, nil
}

// groupUnit is the unit that the groups of each kind are made of.
var groupUnit = map[game.GroupKind]string{
	game.KindFactory: units.FACT,
	game.KindFarming: units.FARM,
	game.KindMining:  units.MINE,
}

const assembleUsage = `want one of:
  assemble Quantity Unit on EntityID
  assemble factory-group on EntityID with Quantity FACT-n producing Unit
  assemble farming-group on EntityID with Quantity FARM-n
  assemble mining-group on EntityID with Quantity MINE-n working DepositNo
  assemble factory-group GroupNo on EntityID add Quantity FACT-n
  assemble mining-group GroupNo on EntityID add Quantity MINE-n
any of them may end with: using Quantity CWRK`

// usingClause takes "using Quantity CWRK" off the end of an order, if it is
// there, and returns the cap and the tokens before it. CWKR is read as CWRK.
func usingClause(tokens []Token) (int, []Token, error) {
	n := len(tokens)
	if n < 3 || !tokens[n-3].Is("using") {
		return 0, tokens, nil
	}
	maxCWRK, ok := quantity(tokens[n-2])
	if !ok || maxCWRK < 1 {
		return 0, nil, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, tokens[n-2].Text)
	}
	// CWKR is read as CWRK. The cadre may name a race, as CWRK-3; nothing
	// uses it yet.
	cadre := tokens[n-1]
	if code, race, hasRace := strings.Cut(cadre.Text, "-"); strings.EqualFold(code, "CWKR") {
		cadre.Text = units.CWRK
		if hasRace {
			cadre.Text += "-" + race
		}
	}
	if code, _, ok := popCode(cadre); !ok || code != units.CWRK {
		return 0, nil, fmt.Errorf("%w: %q: want CWRK", ErrInvalidUnit, tokens[n-1].Text)
	}
	return maxCWRK, tokens[:n-3], nil
}

// parseUnitWork parses "Verb Quantity Unit on EntityID [using Quantity
// CWRK]", where the verb is that of action.
func parseUnitWork(tokens []Token, action game.Work) (Order, error) {
	maxCWRK, tokens, err := usingClause(tokens)
	if err != nil {
		return nil, err
	}
	if len(tokens) != 5 || !tokens[3].Is("on") {
		return nil, fmt.Errorf("%w: want: %s Quantity Unit on EntityID [using Quantity CWRK]", ErrInvalidOrder, action.Verb())
	}
	unit, err := quantityOf(tokens[1], tokens[2], "")
	if err != nil {
		return nil, err
	}
	id, err := entityID(tokens[4])
	if err != nil {
		return nil, err
	}
	return UnitWork{Action: action, EntityID: id, Unit: unit, MaxCWRK: maxCWRK}, nil
}

func parseAssemble(tokens []Token) (Order, error) {
	usage := fmt.Errorf("%w: %s", ErrInvalidOrder, assembleUsage)
	if len(tokens) < 2 {
		return nil, usage
	}
	full := tokens
	maxCWRK, tokens, err := usingClause(tokens)
	if err != nil {
		return nil, err
	}
	var kind game.GroupKind
	for k := range groupUnit {
		if tokens[1].Is(string(k) + "-group") {
			kind = k
		}
	}
	if kind == "" {
		// assemble Quantity Unit on EntityID. People aren't assembled.
		if len(tokens) > 2 {
			if _, _, ok := popCode(tokens[2]); ok {
				return nil, fmt.Errorf("%w: %q: people are drafted, not assembled: draft Quantity UnitOrCadre from EntityID", ErrInvalidUnit, tokens[2].Text)
			}
		}
		return parseUnitWork(full, game.WorkAssemble)
	}

	o := AssembleGroup{Kind: kind, MaxCWRK: maxCWRK}
	rest := tokens[2:]
	// next takes the next token, or fails if there is none.
	next := func() (Token, bool) {
		if len(rest) == 0 {
			return Token{}, false
		}
		t := rest[0]
		rest = rest[1:]
		return t, true
	}
	t, ok := next()
	if ok && !t.Is("on") {
		// A group number: the order adds to a group. An entity has one
		// farming group of each tech level, so the units say which it is.
		if kind == game.KindFarming {
			return nil, fmt.Errorf("%w: a farming group takes no group number, its units' tech level says which group they join: assemble farming-group on EntityID with Quantity FARM-n", ErrInvalidOrder)
		}
		if o.GroupNo, ok = number(t); !ok || o.GroupNo < 1 {
			return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidGroupNo, t.Text)
		}
		t, ok = next()
	}
	if !ok || !t.Is("on") {
		return nil, usage
	}
	if t, ok = next(); !ok {
		return nil, usage
	}
	if o.EntityID, err = entityID(t); err != nil {
		return nil, err
	}
	link := "with"
	if o.GroupNo != 0 {
		link = "add"
	}
	if t, ok = next(); !ok || !t.Is(link) {
		return nil, usage
	}
	q, ok1 := next()
	u, ok2 := next()
	if !ok1 || !ok2 {
		return nil, usage
	}
	if o.Unit, err = quantityOf(q, u, groupUnit[kind]); err != nil {
		return nil, err
	}
	switch {
	case o.GroupNo != 0:
		// Nothing follows the units.
	case kind == game.KindMining:
		t, ok1 := next()
		id, ok2 := next()
		if !ok1 || !ok2 || !t.Is("working") {
			return nil, usage
		}
		if o.DepositNo, ok = number(id); !ok || o.DepositNo < 1 {
			return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidDepositNo, id.Text)
		}
	case kind == game.KindFactory:
		// A new factory group must be told what to make.
		t, ok1 := next()
		u, ok2 := next()
		if !ok1 || !ok2 || !t.Is("producing") {
			return nil, usage
		}
		if o.Producing, err = unitOf(u, ""); err != nil {
			return nil, err
		}
		if d, _ := units.Lookup(o.Producing.Type); !d.FactoryMade() {
			return nil, fmt.Errorf("%w: %q: factories don't make it", ErrInvalidUnit, u.Text)
		}
	}
	if len(rest) > 0 {
		return nil, usage
	}
	return o, nil
}

// entityID returns the value of a token that is an entity ID.
func entityID(t Token) (int, error) {
	id, ok := number(t)
	if !ok || id < 1 {
		return 0, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidEntityID, t.Text)
	}
	return id, nil
}

// quantityOf returns the unit that a quantity token and a unit token name.
// See unitOf for want.
func quantityOf(q, u Token, want string) (game.Unit, error) {
	qty, ok := quantity(q)
	if !ok || qty < 1 {
		return game.Unit{}, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, q.Text)
	}
	unit, err := unitOf(u, want)
	unit.Qty = qty
	return unit, err
}

// unitOf returns the unit that a token names: a code and, for a unit that
// has one, a tech level joined to it by a hyphen, such as FACT-10 or CNGD.
// Case is ignored. People aren't units to assemble or make. If want is not
// empty, the unit must have that code.
func unitOf(t Token, want string) (game.Unit, error) {
	bad := func(why string) (game.Unit, error) {
		return game.Unit{}, fmt.Errorf("%w: %q: %s", ErrInvalidUnit, t.Text, why)
	}
	code, level, hasLevel := strings.Cut(strings.ToUpper(t.Text), "-")
	d, ok := units.Lookup(code)
	switch {
	case t.Quoted || !ok || d.Population():
		if want != "" {
			return bad(fmt.Sprintf("want %s-1 to %s-%d", want, want, units.MaxTechLevel))
		}
		return bad("want a unit, such as FACT-1 or CNGD")
	case want != "" && code != want:
		return bad(fmt.Sprintf("want %s-1 to %s-%d", want, want, units.MaxTechLevel))
	case !d.HasTechLevel:
		if hasLevel {
			return bad(code + " has no tech level")
		}
		return game.Unit{Type: code}, nil
	}
	tl, ok := number(Token{Text: level})
	if !hasLevel || !ok || !d.ValidTechLevel(tl) {
		return bad(fmt.Sprintf("want %s-%d to %s-%d", code, units.MinTechLevel, code, units.MaxTechLevel))
	}
	return game.Unit{Type: code, TechLevel: tl}, nil
}

// quantity returns the value of a token that is a whole number, which may
// have commas between its thousands: "3600" or "3,600", but not "36,00".
func quantity(t Token) (int, bool) {
	groups := strings.Split(t.Text, ",")
	if len(groups) > 1 {
		if n := len(groups[0]); n < 1 || n > 3 {
			return 0, false
		}
		for _, g := range groups[1:] {
			if len(g) != 3 {
				return 0, false
			}
		}
	}
	return number(Token{Text: strings.Join(groups, ""), Quoted: t.Quoted})
}

// percentage returns the value of a token that is a whole number from least
// to most with a "%" joined to it.
func percentage(t Token, least, most int) (int, error) {
	digits, hasSign := strings.CutSuffix(t.Text, "%")
	pct, ok := number(Token{Text: digits, Quoted: t.Quoted})
	if !ok || !hasSign || pct < least || pct > most {
		return 0, fmt.Errorf("%w: %q: want a whole number from %d%% to %d%%, with the %% joined to it",
			ErrInvalidPercentage, t.Text, least, most)
	}
	return pct, nil
}

// number returns the value of a token that is only ASCII digits.
func number(t Token) (int, bool) {
	if t.Quoted || t.Text == "" || strings.Trim(t.Text, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(t.Text)
	return n, err == nil
}

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
//	set idle for GroupKind group GroupNo of EntityID to PercentageAmount
//	draft Quantity UnitOrCadre from EntityID
//	disband Quantity UnitOrCadre from EntityID
//	assemble Quantity CadreUnit on EntityID
//	assemble factory-group on EntityID with Quantity FactoryUnit (, Quantity FactoryUnit)* [producing Unit]
//	assemble farming-group on EntityID with Quantity FarmUnit
//	assemble mining-group on EntityID with Quantity MiningUnit working DepositNo
//	assemble factory-group GroupNo on EntityID add Quantity FactoryUnit (, Quantity FactoryUnit)*
//	assemble farming-group GroupNo on EntityID add Quantity FarmUnit
//	assemble mining-group GroupNo on EntityID add Quantity MiningUnit
//
// EntityID is the ID of an entity. PercentageAmount is a whole number with a
// "%" joined to it: "50%", not "50 %". For rations it is from game.MinRations
// to game.MaxRations, and for idle from game.MinIdlePercent to
// game.MaxIdlePercent. GroupKind is farming, mining, or factory, and GroupNo
// is the number of a group of that kind. Quantity is a whole number, 1 or
// more, which may have commas between its thousands: "3600" or "3,600".
// UnitOrCadre is the code of a population unit or cadre, such as SLD or CWRK.
// A unit that has a tech level is written with it, joined by a hyphen:
// FactoryUnit is FACT-1 to FACT-10, FarmUnit FARM-1 to FARM-10, and
// MiningUnit MINE-1 to MINE-10. A unit that has none is its code alone, such
// as CNGD. The comma between the units of a factory group may be joined to
// the unit before it or stand alone.
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

// Order is one order. Its type says which: SetRations, SetIdle, Draft,
// Disband, AssembleCadre, or AssembleGroup.
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

// SetIdle is the order "set idle for GroupKind group GroupNo of EntityID to
// PercentageAmount". It orders Percent percent of the units of a group of the
// entity with ID EntityID to idle; 0% puts them all back to work.
type SetIdle struct {
	EntityID int
	Kind     game.GroupKind
	GroupNo  int
	Percent  int
}

func (o SetIdle) String() string {
	return fmt.Sprintf("set idle for %s group %d of %d to %d%%", o.Kind, o.GroupNo, o.EntityID, o.Percent)
}

// Draft is the order "draft Quantity UnitOrCadre from EntityID". It drafts
// Qty population units of the entity with ID EntityID into the unit or cadre
// with the code Unit.
type Draft struct {
	EntityID int
	Unit     string
	Qty      int
}

func (o Draft) String() string {
	return fmt.Sprintf("draft %d %s from %d", o.Qty, o.Unit, o.EntityID)
}

// Disband is the order "disband Quantity UnitOrCadre from EntityID". It
// disbands Qty of the unit or cadre with the code Unit in the entity with ID
// EntityID.
type Disband struct {
	EntityID int
	Unit     string
	Qty      int
}

func (o Disband) String() string {
	return fmt.Sprintf("disband %d %s from %d", o.Qty, o.Unit, o.EntityID)
}

// AssembleCadre is the order "assemble Quantity CadreUnit on EntityID". It
// assembles Qty of the cadre with the code Unit, CWRK, TRNE, or SPY, on the
// entity with ID EntityID.
type AssembleCadre struct {
	EntityID int
	Unit     string
	Qty      int
}

func (o AssembleCadre) String() string {
	return fmt.Sprintf("assemble %d %s on %d", o.Qty, o.Unit, o.EntityID)
}

// AssembleGroup is the order that assembles units into a factory, farming,
// or mining group of the entity with ID EntityID: a new group, if GroupNo is
// 0, or more units for the group numbered GroupNo.
type AssembleGroup struct {
	EntityID int
	Kind     game.GroupKind
	GroupNo  int
	// Units is the units to assemble: one entry for a farming or mining
	// group, one or more for a factory group.
	Units []game.Unit
	// DepositNo is the deposit a new mining group works: its number on the
	// entity's planet.
	DepositNo int
	// Producing is what a new factory group is to make, if the order says;
	// its Qty is 0.
	Producing game.Unit
}

func (o AssembleGroup) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "assemble %s-group ", o.Kind)
	if o.GroupNo != 0 {
		fmt.Fprintf(&sb, "%d on %d add ", o.GroupNo, o.EntityID)
	} else {
		fmt.Fprintf(&sb, "on %d with ", o.EntityID)
	}
	for i, u := range o.Units {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%d %s", u.Qty, unitName(u))
	}
	if o.Producing.Type != "" {
		fmt.Fprintf(&sb, " producing %s", unitName(o.Producing))
	}
	if o.DepositNo != 0 {
		fmt.Fprintf(&sb, " working %d", o.DepositNo)
	}
	return sb.String()
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
	case len(tokens) >= 2 && tokens[0].Is("set") && tokens[1].Is("idle"):
		return parseSetIdle(tokens)
	case tokens[0].Is("draft"):
		return parseDraft(tokens, "draft")
	case tokens[0].Is("disband"):
		return parseDraft(tokens, "disband")
	case tokens[0].Is("assemble"):
		return parseAssemble(tokens)
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

const setIdleUsage = "want: set idle for GroupKind group GroupNo of EntityID to PercentageAmount"

func parseSetIdle(tokens []Token) (Order, error) {
	if len(tokens) != 10 || !tokens[2].Is("for") || !tokens[4].Is("group") || !tokens[6].Is("of") || !tokens[8].Is("to") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOrder, setIdleUsage)
	}
	var kind game.GroupKind
	for _, k := range []game.GroupKind{game.KindFarming, game.KindMining, game.KindFactory} {
		if tokens[3].Is(string(k)) {
			kind = k
		}
	}
	if kind == "" {
		return nil, fmt.Errorf("%w: %q: want farming, mining, or factory", ErrInvalidGroupKind, tokens[3].Text)
	}
	no, ok := number(tokens[5])
	if !ok || no < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidGroupNo, tokens[5].Text)
	}
	id, ok := number(tokens[7])
	if !ok || id < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidEntityID, tokens[7].Text)
	}
	pct, err := percentage(tokens[9], game.MinIdlePercent, game.MaxIdlePercent)
	if err != nil {
		return nil, err
	}
	return SetIdle{EntityID: id, Kind: kind, GroupNo: no, Percent: pct}, nil
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
	code := strings.ToUpper(tokens[2].Text)
	if d, ok := units.Lookup(code); tokens[2].Quoted || !ok || !d.Population() {
		return nil, fmt.Errorf("%w: %q: want a population unit or cadre, such as SLD or CWRK", ErrInvalidUnit, tokens[2].Text)
	}
	id, ok := number(tokens[4])
	if !ok || id < 1 {
		return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more", ErrInvalidEntityID, tokens[4].Text)
	}
	if verb == "disband" {
		return Disband{EntityID: id, Unit: code, Qty: qty}, nil
	}
	return Draft{EntityID: id, Unit: code, Qty: qty}, nil
}

// groupUnit is the unit that the groups of each kind are made of.
var groupUnit = map[game.GroupKind]string{
	game.KindFactory: units.FACT,
	game.KindFarming: units.FARM,
	game.KindMining:  units.MINE,
}

const assembleUsage = `want one of:
  assemble Quantity CadreUnit on EntityID (CWRK, TRNE, or SPY)
  assemble factory-group on EntityID with Quantity FACT-n, Quantity FACT-n ... [producing Unit]
  assemble farming-group on EntityID with Quantity FARM-n
  assemble mining-group on EntityID with Quantity MINE-n working DepositNo
  assemble factory-group GroupNo on EntityID add Quantity FACT-n, Quantity FACT-n ...
  assemble farming-group GroupNo on EntityID add Quantity FARM-n
  assemble mining-group GroupNo on EntityID add Quantity MINE-n`

func parseAssemble(tokens []Token) (Order, error) {
	usage := fmt.Errorf("%w: %s", ErrInvalidOrder, assembleUsage)
	if len(tokens) < 2 {
		return nil, usage
	}
	var kind game.GroupKind
	for k := range groupUnit {
		if tokens[1].Is(string(k) + "-group") {
			kind = k
		}
	}
	if kind == "" {
		// assemble Quantity CadreUnit on EntityID
		if len(tokens) != 5 || !tokens[3].Is("on") {
			return nil, usage
		}
		qty, ok := quantity(tokens[1])
		if !ok || qty < 1 {
			return nil, fmt.Errorf("%w: %q: want a whole number, 1 or more, such as 3600 or 3,600", ErrInvalidQuantity, tokens[1].Text)
		}
		code := strings.ToUpper(tokens[2].Text)
		if tokens[2].Quoted || (code != units.CWRK && code != units.TRNE && code != units.SPY) {
			return nil, fmt.Errorf("%w: %q: want CWRK, TRNE, or SPY", ErrInvalidUnit, tokens[2].Text)
		}
		id, err := entityID(tokens[4])
		if err != nil {
			return nil, err
		}
		return AssembleCadre{EntityID: id, Unit: code, Qty: qty}, nil
	}

	o := AssembleGroup{Kind: kind}
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
		// A group number: the order adds to a group.
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
	var err error
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
	for {
		q, ok1 := next()
		u, ok2 := next()
		if !ok1 || !ok2 {
			return nil, usage
		}
		// The comma that says another unit follows may end this one.
		more := !u.Quoted && len(u.Text) > 1 && strings.HasSuffix(u.Text, ",")
		if more {
			u.Text = strings.TrimSuffix(u.Text, ",")
		}
		unit, err := quantityOf(q, u, groupUnit[kind])
		if err != nil {
			return nil, err
		}
		o.Units = append(o.Units, unit)
		if !more && len(rest) > 0 && rest[0].Is(",") {
			rest, more = rest[1:], true
		}
		if !more {
			break
		}
	}
	if len(o.Units) > 1 && kind != game.KindFactory {
		return nil, fmt.Errorf("%w: a %s group takes units of one tech level", ErrInvalidOrder, kind)
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
	case kind == game.KindFactory && len(rest) > 0:
		t, ok1 := next()
		u, ok2 := next()
		if !ok1 || !ok2 || !t.Is("producing") {
			return nil, usage
		}
		if o.Producing, err = unitOf(u, ""); err != nil {
			return nil, err
		}
		if d, _ := units.Lookup(o.Producing.Type); !d.Manufactured {
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
		return bad("want a unit, such as STRC-1 or CNGD")
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

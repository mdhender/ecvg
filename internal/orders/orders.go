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
//
// EntityID is the ID of an entity. PercentageAmount is a whole number with a
// "%" joined to it: "50%", not "50 %". For rations it is from game.MinRations
// to game.MaxRations, and for idle from game.MinIdlePercent to
// game.MaxIdlePercent. GroupKind is farming, mining, or factory, and GroupNo
// is the number of a group of that kind. Quantity is a whole number, 1 or
// more, which may have commas between its thousands: "3600" or "3,600".
// UnitOrCadre is the code of a population unit or cadre, such as SLD or CWRK.
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
)

// Order is one order. Its type says which: SetRations, SetIdle, or Draft.
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
		return parseDraft(tokens)
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

const draftUsage = "want: draft Quantity UnitOrCadre from EntityID"

func parseDraft(tokens []Token) (Order, error) {
	if len(tokens) != 5 || !tokens[3].Is("from") {
		return nil, fmt.Errorf("%w: %s", ErrInvalidOrder, draftUsage)
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
	return Draft{EntityID: id, Unit: code, Qty: qty}, nil
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

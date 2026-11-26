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
//
// EntityID is the ID of an entity. PercentageAmount is a whole number from
// game.MinRations to game.MaxRations with a "%" joined to it: "50%", not
// "50 %".
package orders

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
)

const (
	ErrUnknownOrder      = cerrs.Error("unknown order")
	ErrInvalidOrder      = cerrs.Error("invalid order")
	ErrInvalidEntityID   = cerrs.Error("invalid entity ID")
	ErrInvalidPercentage = cerrs.Error("invalid percentage")
)

// Order is one order. Its type says which: SetRations.
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
	digits, hasSign := strings.CutSuffix(tokens[5].Text, "%")
	pct, ok := number(Token{Text: digits, Quoted: tokens[5].Quoted})
	if !ok || !hasSign || pct < game.MinRations || pct > game.MaxRations {
		return nil, fmt.Errorf("%w: %q: want a whole number from %d%% to %d%%, with the %% joined to it",
			ErrInvalidPercentage, tokens[5].Text, game.MinRations, game.MaxRations)
	}
	return SetRations{EntityID: id, Percent: pct}, nil
}

// number returns the value of a token that is only ASCII digits.
func number(t Token) (int, bool) {
	if t.Quoted || t.Text == "" || strings.Trim(t.Text, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(t.Text)
	return n, err == nil
}

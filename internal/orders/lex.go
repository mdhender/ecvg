// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package orders

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/mdhender/ecvg/internal/cerrs"
)

const (
	ErrInvalidUTF8       = cerrs.Error("line is not valid UTF-8")
	ErrUnterminatedQuote = cerrs.Error("quoted text has no closing quote")
	ErrQuoteJoined       = cerrs.Error("quoted text must be followed by a space or the end of the line")
)

// Token is one word or one piece of quoted text of a line of orders.
type Token struct {
	// Text is the token's text; for quoted text, it is the text between the
	// quotes.
	Text string
	// Quoted is true if the token was written in double quotes.
	Quoted bool
}

// Is reports whether the token is the keyword word, ignoring case. Quoted
// text is never a keyword.
func (t Token) Is(word string) bool {
	return !t.Quoted && strings.EqualFold(t.Text, word)
}

// Lex splits a line into its tokens.
//
// Tokens are separated by any amount of white space. Text in double quotes is
// one token, white space and all. A token that starts with "#" or "//" starts
// a comment, which runs to the end of the line; those characters do not start
// a comment inside quoted text or when joined to the end of a token, so
// "set rations#" is two tokens.
func Lex(line string) ([]Token, error) {
	if !utf8.ValidString(line) {
		return nil, ErrInvalidUTF8
	}
	var tokens []Token
	rest := line
	for {
		rest = strings.TrimLeftFunc(rest, unicode.IsSpace)
		if rest == "" || strings.HasPrefix(rest, "#") || strings.HasPrefix(rest, "//") {
			return tokens, nil
		}
		if quoted, ok := strings.CutPrefix(rest, `"`); ok {
			text, after, ok := strings.Cut(quoted, `"`)
			if !ok {
				return nil, ErrUnterminatedQuote
			}
			if r, _ := utf8.DecodeRuneInString(after); after != "" && !unicode.IsSpace(r) {
				return nil, ErrQuoteJoined
			}
			tokens = append(tokens, Token{Text: text, Quoted: true})
			rest = after
			continue
		}
		end := strings.IndexFunc(rest, unicode.IsSpace)
		if end < 0 {
			end = len(rest)
		}
		tokens = append(tokens, Token{Text: rest[:end]})
		rest = rest[end:]
	}
}

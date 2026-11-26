// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package orders_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

func TestLex(t *testing.T) {
	w := func(words ...string) []orders.Token {
		var ts []orders.Token
		for _, word := range words {
			ts = append(ts, orders.Token{Text: word})
		}
		return ts
	}
	for _, tc := range []struct {
		name string
		line string
		want []orders.Token
		err  error
	}{
		{name: "empty", line: ""},
		{name: "white space only", line: " \t "},
		{name: "words", line: "set rations for 7 to 50%", want: w("set", "rations", "for", "7", "to", "50%")},
		{name: "white space is compressed", line: "  set\t\trations   for　 7 ", want: w("set", "rations", "for", "7")},
		{name: "case is kept", line: "SET Rations", want: w("SET", "Rations")},
		{name: "utf-8 words", line: "naïve 帝国", want: w("naïve", "帝国")},
		{name: "hash comment line", line: "# set rations"},
		{name: "slash comment line", line: "  // set rations"},
		{name: "hash comment after tokens", line: "set rations # for 7", want: w("set", "rations")},
		{name: "slash comment after tokens", line: "set rations // for 7", want: w("set", "rations")},
		{name: "comment joined to its text", line: "set #rations", want: w("set")},
		{name: "hash joined to a token is not a comment", line: "set rations# for", want: w("set", "rations#", "for")},
		{name: "slashes joined to a token are not a comment", line: "set rations// for", want: w("set", "rations//", "for")},
		{name: "one slash is not a comment", line: "a / b", want: w("a", "/", "b")},
		{name: "quoted text", line: `name 7 "The  Big # One // really"`,
			want: []orders.Token{{Text: "name"}, {Text: "7"}, {Text: "The  Big # One // really", Quoted: true}}},
		{name: "empty quoted text", line: `name ""`, want: []orders.Token{{Text: "name"}, {Text: "", Quoted: true}}},
		{name: "comment after quoted text", line: `"a" # b`, want: []orders.Token{{Text: "a", Quoted: true}}},
		{name: "quote inside a word", line: `ab"c d`, want: w(`ab"c`, "d")},
		{name: "unterminated quote", line: `name "The Big One`, err: orders.ErrUnterminatedQuote},
		{name: "quoted text joined to a token", line: `"a"b`, err: orders.ErrQuoteJoined},
		{name: "quoted text joined to a comment", line: `"a"#b`, err: orders.ErrQuoteJoined},
		{name: "invalid utf-8", line: "set \xff", err: orders.ErrInvalidUTF8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := orders.Lex(tc.line)
			if !errors.Is(err, tc.err) || (tc.err == nil && err != nil) {
				t.Fatalf("Lex(%q) error = %v, want %v", tc.line, err, tc.err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Lex(%q) = %+v, want %+v", tc.line, got, tc.want)
			}
		})
	}
}

func TestParseSetRations(t *testing.T) {
	for _, tc := range []struct {
		line string
		want orders.Order
		err  error
	}{
		{line: "set rations for 7 to 50%", want: orders.SetRations{EntityID: 7, Percent: 50}},
		{line: "SET Rations FOR 7 To 50%", want: orders.SetRations{EntityID: 7, Percent: 50}},
		{line: "\tset   rations for 1500   to 1%  ", want: orders.SetRations{EntityID: 1500, Percent: 1}},
		{line: "set rations for 7 to 500% // feast", want: orders.SetRations{EntityID: 7, Percent: 500}},
		{line: "set rations for 007 to 050% # leading zeros", want: orders.SetRations{EntityID: 7, Percent: 50}},
		{line: "set rations for 7 to 50 %", err: orders.ErrInvalidOrder},
		{line: "set rations for 7 to 50", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 0%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 501%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to -5%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to +5%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 12.5%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to %", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 50%%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to ５０%", err: orders.ErrInvalidPercentage},
		{line: `set rations for 7 to "50%"`, err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 50%# comment", err: orders.ErrInvalidOrder},
		{line: "set rations for 7 to 50%#", err: orders.ErrInvalidPercentage},
		{line: "set rations for 0 to 50%", err: orders.ErrInvalidEntityID},
		{line: "set rations for -7 to 50%", err: orders.ErrInvalidEntityID},
		{line: "set rations for seven to 50%", err: orders.ErrInvalidEntityID},
		{line: "set rations for 99999999999999999999 to 50%", err: orders.ErrInvalidEntityID},
		{line: "set rations 7 50%", err: orders.ErrInvalidOrder},
		{line: "set rations for 7 at 50%", err: orders.ErrInvalidOrder},
		{line: "set rations for 7 to 50% now", err: orders.ErrInvalidOrder},
		{line: "set rations", err: orders.ErrInvalidOrder},
		{line: "set rations# for 7 to 50%", err: orders.ErrUnknownOrder},
		{line: `"set" rations for 7 to 50%`, err: orders.ErrUnknownOrder},
		{line: "set", err: orders.ErrUnknownOrder},
		{line: "launch missiles", err: orders.ErrUnknownOrder},
		{line: `name "unfinished`, err: orders.ErrUnterminatedQuote},
	} {
		t.Run(tc.line, func(t *testing.T) {
			lines, err := orders.Parse(strings.NewReader(tc.line))
			if err != nil || len(lines) != 1 {
				t.Fatalf("Parse = %+v, %v; want one line", lines, err)
			}
			got := lines[0]
			if !errors.Is(got.Err, tc.err) || (tc.err == nil && got.Err != nil) {
				t.Fatalf("error = %v, want %v", got.Err, tc.err)
			}
			if got.Order != tc.want {
				t.Errorf("order = %+v, want %+v", got.Order, tc.want)
			}
			if got.Text != strings.TrimSpace(tc.line) || got.No != 1 {
				t.Errorf("line = %d %q, want 1 %q", got.No, got.Text, strings.TrimSpace(tc.line))
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	const text = "# orders for turn 0\r\n" +
		"\n" +
		"set rations for 7 to 50%\r\n" +
		"   // nothing here\n" +
		"set rations for 7 to 50% set rations for 8 to 50%\n" +
		"set rations for 8 to 125%"
	lines, err := orders.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		no    int
		order string
		bad   bool
	}
	var got []row
	for _, l := range lines {
		r := row{no: l.No, bad: l.Err != nil}
		if l.Order != nil {
			r.order = l.Order.String()
		}
		got = append(got, r)
	}
	want := []row{{3, "set rations for 7 to 50%", false}, {5, "", true}, {6, "set rations for 8 to 125%", false}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse = %+v, want %+v", got, want)
	}
}

func TestEmpireID(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int
		ok   bool
	}{
		{"empire-001.txt", 1, true},
		{"empire-7.txt", 7, true},
		{"empire-150.txt", 150, true},
		{orders.FileName(42), 42, true},
		{"empire-000.txt", 0, false},
		{"empire-.txt", 0, false},
		{"empire-1.text", 0, false},
		{"empire--1.txt", 0, false},
		{"Empire-001.txt", 0, false},
		{"notes.txt", 0, false},
	} {
		got, err := orders.EmpireID(tc.name)
		if got != tc.want || (err == nil) != tc.ok || (err != nil && !errors.Is(err, orders.ErrInvalidFileName)) {
			t.Errorf("EmpireID(%q) = %d, %v; want %d, ok %v", tc.name, got, err, tc.want, tc.ok)
		}
	}
}

func TestLoadTurn(t *testing.T) {
	write := func(t *testing.T, dir string, files map[string]string) {
		t.Helper()
		ordersDir := filepath.Join(dir, "turns", "3", "orders")
		if err := os.MkdirAll(filepath.Join(ordersDir, "old"), 0o755); err != nil {
			t.Fatal(err)
		}
		for name, text := range files {
			if err := os.WriteFile(filepath.Join(ordersDir, name), []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Run("no orders folder", func(t *testing.T) {
		files, err := orders.LoadTurn(t.TempDir(), 3)
		if err != nil || len(files) != 0 {
			t.Errorf("LoadTurn = %+v, %v; want none", files, err)
		}
	})
	t.Run("files in empire order", func(t *testing.T) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"empire-010.txt": "set rations for 10 to 50%\nfeed everyone\n",
			"empire-2.txt":   "# nothing this turn\n",
			".DS_Store":      "junk",
		})
		files, err := orders.LoadTurn(dir, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 2 || files[0].EmpireID != 2 || files[1].EmpireID != 10 || files[1].Name != "empire-010.txt" {
			t.Fatalf("LoadTurn = %+v, want empires 2 and 10", files)
		}
		if len(files[0].Lines) != 0 || len(files[1].Lines) != 2 || files[1].Lines[1].Err == nil {
			t.Errorf("lines = %+v and %+v, want none and two with the second invalid", files[0].Lines, files[1].Lines)
		}
	})
	for name, tc := range map[string]struct {
		files map[string]string
		err   error
	}{
		"stray file":         {map[string]string{"notes.txt": ""}, orders.ErrInvalidFileName},
		"two for one empire": {map[string]string{"empire-1.txt": "", "empire-001.txt": ""}, orders.ErrDuplicateFile},
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			write(t, dir, tc.files)
			if _, err := orders.LoadTurn(dir, 3); !errors.Is(err, tc.err) {
				t.Errorf("LoadTurn error = %v, want %v", err, tc.err)
			}
		})
	}
}

func TestParseSetIdle(t *testing.T) {
	for _, tc := range []struct {
		line string
		want orders.Order
		err  error
	}{
		{line: "set idle for farming group 1 of 7 to 94%", want: orders.SetIdle{EntityID: 7, Kind: game.KindFarming, GroupNo: 1, Percent: 94}},
		{line: "SET IDLE FOR Mining GROUP 02 OF 007 TO 100% # all of it", want: orders.SetIdle{EntityID: 7, Kind: game.KindMining, GroupNo: 2, Percent: 100}},
		{line: "set idle for factory group 3 of 7 to 0%", want: orders.SetIdle{EntityID: 7, Kind: game.KindFactory, GroupNo: 3, Percent: 0}},
		{line: "set idle for farming group 1 of 7 to 101%", err: orders.ErrInvalidPercentage},
		{line: "set idle for farming group 1 of 7 to 50", err: orders.ErrInvalidPercentage},
		{line: "set idle for farming group 1 of 7 to -1%", err: orders.ErrInvalidPercentage},
		{line: "set idle for farm group 1 of 7 to 50%", err: orders.ErrInvalidGroupKind},
		{line: `set idle for "farming" group 1 of 7 to 50%`, err: orders.ErrInvalidGroupKind},
		{line: "set idle for farming group 0 of 7 to 50%", err: orders.ErrInvalidGroupNo},
		{line: "set idle for farming group one of 7 to 50%", err: orders.ErrInvalidGroupNo},
		{line: "set idle for farming group 1 of 0 to 50%", err: orders.ErrInvalidEntityID},
		{line: "set idle for farming group 1 of x to 50%", err: orders.ErrInvalidEntityID},
		{line: "set idle for farming group 1 in 7 to 50%", err: orders.ErrInvalidOrder},
		{line: "set idle for farming 1 of 7 to 50%", err: orders.ErrInvalidOrder},
		{line: "set idle for farming group 1 of 7 to 50% now", err: orders.ErrInvalidOrder},
		{line: "set idle", err: orders.ErrInvalidOrder},
	} {
		t.Run(tc.line, func(t *testing.T) {
			lines, err := orders.Parse(strings.NewReader(tc.line))
			if err != nil || len(lines) != 1 {
				t.Fatalf("Parse = %+v, %v; want one line", lines, err)
			}
			got := lines[0]
			if !errors.Is(got.Err, tc.err) || (tc.err == nil && got.Err != nil) {
				t.Fatalf("error = %v, want %v", got.Err, tc.err)
			}
			if got.Order != tc.want {
				t.Errorf("order = %+v, want %+v", got.Order, tc.want)
			}
			if tc.want != nil {
				// An order's String parses back to the order.
				again, err := orders.Parse(strings.NewReader(tc.want.String()))
				if err != nil || len(again) != 1 || again[0].Order != tc.want {
					t.Errorf("String() = %q, which parses to %+v, %v", tc.want.String(), again, err)
				}
			}
		})
	}
}

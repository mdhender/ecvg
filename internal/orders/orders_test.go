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
		{line: "\tset   rations for 1500   to 25%  ", want: orders.SetRations{EntityID: 1500, Percent: 25}},
		{line: "set rations for 7 to 100% // the most", want: orders.SetRations{EntityID: 7, Percent: 100}},
		{line: "set rations for 007 to 050% # leading zeros", want: orders.SetRations{EntityID: 7, Percent: 50}},
		{line: "set rations for 7 to 50 %", err: orders.ErrInvalidOrder},
		{line: "set rations for 7 to 50", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 0%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 24%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 101%", err: orders.ErrInvalidPercentage},
		{line: "set rations for 7 to 500%", err: orders.ErrInvalidPercentage},
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
		"set rations for 8 to 75%"
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
	want := []row{{3, "set rations for 7 to 50%", false}, {5, "", true}, {6, "set rations for 8 to 75%", false}}
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

func TestParseUpdateGroup(t *testing.T) {
	update := func(kind game.GroupKind, groupNo, entityID int, action game.GroupAction) orders.UpdateGroup {
		return orders.UpdateGroup{Kind: kind, GroupNo: groupNo, EntityID: entityID, Action: action}
	}
	qty := func(o orders.UpdateGroup, n int) orders.UpdateGroup { o.Qty = n; return o }
	pct := func(o orders.UpdateGroup, n int) orders.UpdateGroup { o.Percent = n; return o }
	producing := func(o orders.UpdateGroup, u game.Unit) orders.UpdateGroup { o.Producing = u; return o }
	for _, tc := range []struct {
		line string
		want orders.Order
		err  error
	}{
		{line: "update factory-group 8 of 16 retool", want: update(game.KindFactory, 8, 16, game.ActionRetool)},
		{line: "UPDATE Factory-Group 08 OF 016 RETOOL # all of it", want: update(game.KindFactory, 8, 16, game.ActionRetool)},
		{line: "update factory-group 8 of 16 produce EWPN-1", want: producing(update(game.KindFactory, 8, 16, game.ActionProduce), game.Unit{Type: "EWPN", TechLevel: 1})},
		{line: "update factory-group 8 of 16 produce rsch", err: orders.ErrInvalidUnit},
		{line: "update laboratory-group 9 of 16 shutdown 25%", want: pct(update(game.KindLaboratory, 9, 16, game.ActionShutDown), 25)},
		{line: "update Laboratory-Group 9 of 16 restart 3", want: qty(update(game.KindLaboratory, 9, 16, game.ActionRestart), 3)},
		{line: "update laboratory-group 9 of 16 retool", err: orders.ErrInvalidOrder},
		{line: "update laboratory-group 9 of 16 produce RSCH", err: orders.ErrInvalidOrder},
		{line: "update lab-group 9 of 16 shutdown 1", err: orders.ErrInvalidGroupKind},
		{line: "update factory-group 8 of 16 shutdown 500", want: qty(update(game.KindFactory, 8, 16, game.ActionShutDown), 500)},
		{line: "update factory-group 8 of 16 shutdown 2,500", want: qty(update(game.KindFactory, 8, 16, game.ActionShutDown), 2500)},
		{line: "update factory-group 8 of 16 shutdown 25%", want: pct(update(game.KindFactory, 8, 16, game.ActionShutDown), 25)},
		{line: "update factory-group 8 of 16 restart 200", want: qty(update(game.KindFactory, 8, 16, game.ActionRestart), 200)},
		{line: "update factory-group 8 of 16 restart 100%", want: pct(update(game.KindFactory, 8, 16, game.ActionRestart), 100)},
		{line: "update farming-group 1 of 7 shutdown 94%", want: pct(update(game.KindFarming, 1, 7, game.ActionShutDown), 94)},
		{line: "update mining-group 2 of 7 restart 10", want: qty(update(game.KindMining, 2, 7, game.ActionRestart), 10)},
		{line: "update factory-group 8 of 16 retool .", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 retool now", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 produce", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 produce EWPN-1 CNGD", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 produce EWPN", err: orders.ErrInvalidUnit},
		{line: "update factory-group 8 of 16 produce FUEL", err: orders.ErrInvalidUnit},
		{line: "update factory-group 8 of 16 produce SLD", err: orders.ErrInvalidUnit},
		{line: "update factory-group 8 of 16 produce RETL", err: orders.ErrInvalidUnit},
		{line: "update farming-group 1 of 7 retool", err: orders.ErrInvalidOrder},
		{line: "update mining-group 1 of 7 produce CNGD", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 shutdown", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 shutdown 5 10%", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16 shutdown 0", err: orders.ErrInvalidQuantity},
		{line: "update factory-group 8 of 16 shutdown all", err: orders.ErrInvalidQuantity},
		{line: "update factory-group 8 of 16 shutdown 36,00", err: orders.ErrInvalidQuantity},
		{line: "update factory-group 8 of 16 shutdown 0%", err: orders.ErrInvalidPercentage},
		{line: "update factory-group 8 of 16 restart 101%", err: orders.ErrInvalidPercentage},
		{line: "update factory-group 8 of 16 restart -1%", err: orders.ErrInvalidPercentage},
		{line: "update factory-group 8 of 16 idle 50%", err: orders.ErrInvalidOrder},
		{line: "update factory group 8 of 16 retool", err: orders.ErrInvalidGroupKind},
		{line: "update fact-group 8 of 16 retool", err: orders.ErrInvalidGroupKind},
		{line: `update "factory-group" 8 of 16 retool`, err: orders.ErrInvalidGroupKind},
		{line: "update factory-group 0 of 16 retool", err: orders.ErrInvalidGroupNo},
		{line: "update factory-group eight of 16 retool", err: orders.ErrInvalidGroupNo},
		{line: "update factory-group 8 of 0 retool", err: orders.ErrInvalidEntityID},
		{line: "update factory-group 8 of x retool", err: orders.ErrInvalidEntityID},
		{line: "update factory-group 8 on 16 retool", err: orders.ErrInvalidOrder},
		{line: "update factory-group 8 of 16", err: orders.ErrInvalidOrder},
		{line: "update", err: orders.ErrInvalidOrder},
		{line: "set idle for farming group 1 of 7 to 94%", err: orders.ErrUnknownOrder},
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

func TestParseDraft(t *testing.T) {
	for _, tc := range []struct {
		line string
		want orders.Order
		err  error
	}{
		{line: "draft 3,600 SLD from 13", want: orders.Draft{EntityID: 13, Unit: "SLD", Qty: 3600}},
		{line: "draft 16880 TRNE from 78", want: orders.Draft{EntityID: 78, Unit: "TRNE", Qty: 16880}},
		{line: "draft 5,000 CWRK from 99", want: orders.Draft{EntityID: 99, Unit: "CWRK", Qty: 5000}},
		{line: "DRAFT 1,234,567 cwrk FROM 099 // lots", want: orders.Draft{EntityID: 99, Unit: "CWRK", Qty: 1234567}},
		{line: "draft 1 spy from 1", want: orders.Draft{EntityID: 1, Unit: "SPY", Qty: 1}},
		// A race ID is optional; 0 is the empire's own race.
		{line: "draft 3,600 SLD-0 from 13", want: orders.Draft{EntityID: 13, Unit: "SLD", Qty: 3600}},
		{line: "draft 3,600 sld-3 from 13", want: orders.Draft{EntityID: 13, Unit: "SLD", Race: 3, Qty: 3600}},
		{line: "draft 5,000 CWRK-15 from 99", want: orders.Draft{EntityID: 99, Unit: "CWRK", Race: 15, Qty: 5000}},
		{line: "draft 3,600 SLD- from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 SLD-x from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 SLD--3 from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 SLD-3-1 from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 FARM-1 from 13", err: orders.ErrInvalidUnit},
		{line: "draft 0 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft -5 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft 36,00 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft 3,6000 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft ,600 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft 3,600, SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft 1234,567 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft 3.5 SLD from 13", err: orders.ErrInvalidQuantity},
		{line: "draft many SLD from 13", err: orders.ErrInvalidQuantity},
		{line: `draft "3,600" SLD from 13`, err: orders.ErrInvalidQuantity},
		{line: "draft 3,600 FARM from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 FOOD from 13", err: orders.ErrInvalidUnit},
		{line: "draft 3,600 troops from 13", err: orders.ErrInvalidUnit},
		{line: `draft 3,600 "SLD" from 13`, err: orders.ErrInvalidUnit},
		{line: "draft 3,600 SLD from 0", err: orders.ErrInvalidEntityID},
		{line: "draft 3,600 SLD from home", err: orders.ErrInvalidEntityID},
		{line: "draft 3,600 SLD in 13", err: orders.ErrInvalidOrder},
		{line: "draft 3,600 SLD from 13 .", err: orders.ErrInvalidOrder},
		{line: "draft 3,600 SLD", err: orders.ErrInvalidOrder},
		{line: "draft", err: orders.ErrInvalidOrder},
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
				again, err := orders.Parse(strings.NewReader(tc.want.String()))
				if err != nil || len(again) != 1 || again[0].Order != tc.want {
					t.Errorf("String() = %q, which parses to %+v, %v", tc.want.String(), again, err)
				}
			}
		})
	}
}

func TestParseDisbandAndAssemble(t *testing.T) {
	type unit = game.Unit
	group := func(kind game.GroupKind, no, entity int, u unit) orders.AssembleGroup {
		return orders.AssembleGroup{EntityID: entity, Kind: kind, GroupNo: no, Unit: u}
	}
	working := func(o orders.AssembleGroup, deposit int) orders.AssembleGroup {
		o.DepositNo = deposit
		return o
	}
	producing := func(o orders.AssembleGroup, u unit) orders.AssembleGroup {
		o.Producing = u
		return o
	}
	for _, tc := range []struct {
		line string
		want orders.Order
		err  error
	}{
		{line: "disband 5,000 CWRK from 99", want: orders.Disband{EntityID: 99, Unit: "CWRK", Qty: 5000}},
		{line: "Disband 40 spy From 7", want: orders.Disband{EntityID: 7, Unit: "SPY", Qty: 40}},
		{line: "disband 40 TRNE-2 from 7", want: orders.Disband{EntityID: 7, Unit: "TRNE", Race: 2, Qty: 40}},
		{line: "disband 40 SPY-0 from 7", want: orders.Disband{EntityID: 7, Unit: "SPY", Qty: 40}},
		{line: "disband 0 CWRK from 99", err: orders.ErrInvalidQuantity},
		{line: "disband 10 FARM from 99", err: orders.ErrInvalidUnit},
		{line: "disband 10 CWRK on 99", err: orders.ErrInvalidOrder},

		// Cadres are drafted, not assembled.
		{line: "assemble 5,000 CWRK on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 100 trne on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 7 SPY on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 100 SLD on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 5,000 CWRK-3 on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 5,000 HDRV-1 on 15 using 20 CWRK-3", want: orders.UnitWork{EntityID: 15, Unit: unit{Type: "HDRV", TechLevel: 1, Qty: 5000}, MaxCWRK: 20}},
		{line: "assemble 5,000 HDRV-1 on 15 using 20 cwkr-0", want: orders.UnitWork{EntityID: 15, Unit: unit{Type: "HDRV", TechLevel: 1, Qty: 5000}, MaxCWRK: 20}},
		{line: "assemble 5,000 HDRV-1 on 15 using 20 CWRK-x", err: orders.ErrInvalidUnit},
		{line: "assemble 5,000 HDRV-1 on 15 using 20 SLD-3", err: orders.ErrInvalidUnit},
		{line: "assemble 100 STRL on 5", want: orders.UnitWork{EntityID: 5, Unit: unit{Type: "STRL", Qty: 100}}},
		{line: "assemble 5,000 HDRV-1 on 15 using 20 CWKR", want: orders.UnitWork{EntityID: 15, Unit: unit{Type: "HDRV", TechLevel: 1, Qty: 5000}, MaxCWRK: 20}},
		{line: "assemble 10 cngd on 5", want: orders.UnitWork{EntityID: 5, Unit: unit{Type: "CNGD", Qty: 10}}},
		{line: "assemble 1 HDRV-1 using 20,000 CWKR", err: orders.ErrInvalidOrder},
		{line: "assemble 5 CWRK on 5 using 2 CWRK", err: orders.ErrInvalidUnit},
		{line: "assemble 5 HDRV on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 5 HDRV-11 on 5", err: orders.ErrInvalidUnit},

		{line: "Disassemble 3 lfsu-10 on 5 using 1,000 cwrk", want: orders.UnitWork{Action: game.WorkDisassemble, EntityID: 5, Unit: unit{Type: "LFSU", TechLevel: 10, Qty: 3}, MaxCWRK: 1000}},
		{line: "disassemble 3 LFSU-10 on 5", want: orders.UnitWork{Action: game.WorkDisassemble, EntityID: 5, Unit: unit{Type: "LFSU", TechLevel: 10, Qty: 3}}},
		{line: "disassemble 3 LFSU-10 from 5", err: orders.ErrInvalidOrder},
		{line: "disassemble 3 CWRK on 5", err: orders.ErrInvalidUnit},
		{line: "unassemble 3 LFSU-10 on 5", err: orders.ErrUnknownOrder},

		{line: "scrap 100 HDRV-1 on 15 using 200 CWRK", want: orders.UnitWork{Action: game.WorkScrap, EntityID: 15, Unit: unit{Type: "HDRV", TechLevel: 1, Qty: 100}, MaxCWRK: 200}},
		{line: "scrap 10 CNGD on 5", want: orders.UnitWork{Action: game.WorkScrap, EntityID: 5, Unit: unit{Type: "CNGD", Qty: 10}}},
		{line: "scrap 5 SLD on 5", err: orders.ErrInvalidUnit},
		{line: "scrap 0 HDRV-1 on 5", err: orders.ErrInvalidQuantity},
		{line: "scrap 5 HDRV-1 on five", err: orders.ErrInvalidEntityID},
		{line: "scrap 5 HDRV-1 on 5 using 0 CWRK", err: orders.ErrInvalidQuantity},
		{line: "scrap 5 HDRV-1 on 5 using 5 SLD", err: orders.ErrInvalidUnit},
		{line: "scrap 5 HDRV-1 on 5 using 5", err: orders.ErrInvalidOrder},
		{line: "scrap 5 HDRV-1 on 5 using 5 CWRK now", err: orders.ErrInvalidOrder},
		{line: "assemble 100 widgets on 5", err: orders.ErrInvalidUnit},
		{line: "assemble 0 STRN on 5", err: orders.ErrInvalidQuantity},
		{line: "assemble 100 STRN on home", err: orders.ErrInvalidEntityID},
		{line: "assemble 100 STRN from 5", err: orders.ErrInvalidOrder},
		{line: "assemble", err: orders.ErrInvalidOrder},

		{line: "assemble farming-group on 5 with 1,000 FARM-1", want: group(game.KindFarming, 0, 5, unit{Type: "FARM", TechLevel: 1, Qty: 1000})},
		{line: "ASSEMBLE Farming-Group ON 5 WITH 10 farm-10", want: group(game.KindFarming, 0, 5, unit{Type: "FARM", TechLevel: 10, Qty: 10})},
		{line: "assemble farming-group on 5 with 10 FARM-1, 10 FARM-2", err: orders.ErrInvalidUnit},
		{line: "assemble farming-group on 5 with 10 MINE-1", err: orders.ErrInvalidUnit},
		{line: "assemble farming-group on 5 with 10 FARM", err: orders.ErrInvalidUnit},
		{line: "assemble farming-group on 5 add 10 FARM-1", err: orders.ErrInvalidOrder},
		{line: "assemble farming-group 2 on 5 with 10 FARM-1", err: orders.ErrInvalidOrder},
		{line: "assemble farming-group 2 on 5 add 10 FARM-1", err: orders.ErrInvalidOrder},
		{line: "assemble mining-group 0 on 5 add 10 MINE-1", err: orders.ErrInvalidGroupNo},
		{line: "assemble farming-group on 5 with 10 FARM-1 working 3", err: orders.ErrInvalidOrder},
		{line: "assemble farming-group on 5 with", err: orders.ErrInvalidOrder},
		{line: "assemble farming-group on 5", err: orders.ErrInvalidOrder},
		{line: "assemble farming-group", err: orders.ErrInvalidOrder},

		{line: "assemble mining-group on 5 with 25,000 MINE-10 working 17", want: working(group(game.KindMining, 0, 5, unit{Type: "MINE", TechLevel: 10, Qty: 25000}), 17)},
		{line: "assemble mining-group 3 on 5 add 100 MINE-10", want: group(game.KindMining, 3, 5, unit{Type: "MINE", TechLevel: 10, Qty: 100})},
		{line: "assemble mining-group on 5 with 100 MINE-10", err: orders.ErrInvalidOrder},
		{
			line: "assemble mining-group on 5 with 100 MINE-10 working 17 using 50 CWRK",
			want: orders.AssembleGroup{EntityID: 5, Kind: game.KindMining, Unit: unit{Type: "MINE", TechLevel: 10, Qty: 100}, DepositNo: 17, MaxCWRK: 50},
		},
		{
			line: "assemble factory-group 4 on 5 add 100 FACT-1 using 7 CWKR",
			want: orders.AssembleGroup{EntityID: 5, Kind: game.KindFactory, GroupNo: 4, Unit: unit{Type: "FACT", TechLevel: 1, Qty: 100}, MaxCWRK: 7},
		},
		{line: "assemble mining-group on 5 with 100 MINE-10 using 50 CWRK working 17", err: orders.ErrInvalidOrder},
		{line: "assemble mining-group on 5 with 100 MINE-10 working gold", err: orders.ErrInvalidDepositNo},
		{line: "assemble mining-group on 5 with 100 MINE-10 working 0", err: orders.ErrInvalidDepositNo},
		{line: "assemble mining-group 3 on 5 add 100 MINE-10 working 17", err: orders.ErrInvalidOrder},
		{line: "assemble mining-group on 5 with 100 FACT-1 working 17", err: orders.ErrInvalidUnit},

		{line: "assemble factory-group on 5 with 100 FACT-1", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FACT-1 using 3 CWRK", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing rsch", err: orders.ErrInvalidUnit},
		{line: "assemble PRTO-3 on 5 from laboratory-group 2", want: orders.AssemblePrototype{EntityID: 5, TechLevel: 3, GroupNo: 2}},
		{line: "assemble PRTO-3 on 5 from lab-group 2", err: orders.ErrInvalidOrder},
		{line: "Assemble prto-10 ON 5 FROM Laboratory-Group 12 # for the fleet", want: orders.AssemblePrototype{EntityID: 5, TechLevel: 10, GroupNo: 12}},
		{line: "assemble PRTO on 5 from laboratory-group 2", err: orders.ErrInvalidUnit},
		{line: "assemble PRTO-11 on 5 from laboratory-group 2", err: orders.ErrInvalidUnit},
		{line: "assemble PRTO-3 on 5", err: orders.ErrInvalidOrder},
		{line: "assemble PRTO-3 on 5 from factory-group 2", err: orders.ErrInvalidOrder},
		{line: "assemble PRTO-3 on 5 from laboratory-group two", err: orders.ErrInvalidGroupNo},
		{line: "assemble PRTO-3 on 5 from laboratory-group 2 using 5 CWRK", err: orders.ErrInvalidOrder},
		{line: "assemble 2 PRTO-3 on 5 from laboratory-group 2", err: orders.ErrInvalidOrder},
		{line: "transfer 1,000 PRTO-3 from 5 to 6", want: orders.Transfer{From: 5, To: 6, Unit: unit{Type: "PRTO", TechLevel: 3, Qty: 1000}}},
		{line: "TRANSFER 20 cngd FROM 5 TO 6", want: orders.Transfer{From: 5, To: 6, Unit: unit{Type: "CNGD", Qty: 20}}},
		{line: "transfer 20 CNGD to 6 from 5", err: orders.ErrInvalidOrder},
		{line: "transfer 20 CNGD from 5", err: orders.ErrInvalidOrder},
		{line: "transfer CNGD from 5 to 6", err: orders.ErrInvalidOrder},
		{line: "transfer 0 CNGD from 5 to 6", err: orders.ErrInvalidQuantity},
		{line: "transfer 20 PRTO from 5 to 6", err: orders.ErrInvalidUnit},
		{line: "transfer 20 PRO from 5 to 6", err: orders.ErrInvalidUnit},
		{line: "transfer 20 CNGD from 5 to 0", err: orders.ErrInvalidEntityID},
		{line: "assemble tech-level on 5", want: orders.AssembleTechLevel{EntityID: 5}},
		{line: "Assemble Tech-Level ON 5 TO tl-3", want: orders.AssembleTechLevel{EntityID: 5, To: 3}},
		{line: "assemble tech-level on 5 with 2 PRTO-5", want: orders.AssembleTechLevel{EntityID: 5, PRTO: unit{Type: "PRTO", TechLevel: 5, Qty: 2}}},
		{
			line: "assemble tech-level on 5 to TL-10 with 1,000 prto-1 using 2,000 CWRK # all in",
			want: orders.AssembleTechLevel{EntityID: 5, To: 10, PRTO: unit{Type: "PRTO", TechLevel: 1, Qty: 1000}, MaxCWRK: 2000},
		},
		{line: "assemble tech-level on 5 using 20 CWRK", want: orders.AssembleTechLevel{EntityID: 5, MaxCWRK: 20}},
		{line: "assemble tech-level", err: orders.ErrInvalidOrder},
		{line: "assemble tech-level 5", err: orders.ErrInvalidOrder},
		{line: "assemble tech-level on five", err: orders.ErrInvalidEntityID},
		{line: "assemble tech-level on 5 to", err: orders.ErrInvalidOrder},
		{line: "assemble tech-level on 5 to 3", err: orders.ErrInvalidTechLevel},
		{line: "assemble tech-level on 5 to TL-11", err: orders.ErrInvalidTechLevel},
		{line: "assemble tech-level on 5 to TL-0", err: orders.ErrInvalidTechLevel},
		{line: "assemble tech-level on 5 with 2 RSCH", err: orders.ErrInvalidUnit},
		{line: "assemble tech-level on 5 with 2 PRTO", err: orders.ErrInvalidUnit},
		{line: "assemble tech-level on 5 with 0 PRTO-2", err: orders.ErrInvalidQuantity},
		{line: "assemble tech-level on 5 with 2 PRTO-2 to TL-3", err: orders.ErrInvalidOrder},
		{line: "assemble tech-level on 5 to TL-3 to TL-4", err: orders.ErrInvalidOrder},
		{line: "assemble laboratory-group on 5 with 100 LAB-1", want: group(game.KindLaboratory, 0, 5, unit{Type: "LAB", TechLevel: 1, Qty: 100})},
		{
			line: "Assemble Laboratory-Group ON 5 WITH 1,000 lab-10 using 20 CWRK",
			want: orders.AssembleGroup{EntityID: 5, Kind: game.KindLaboratory, Unit: unit{Type: "LAB", TechLevel: 10, Qty: 1000}, MaxCWRK: 20},
		},
		{line: "assemble laboratory-group 4 on 5 add 100 LAB-1", err: orders.ErrInvalidOrder},
		{line: "assemble laboratory-group on 5 with 100 FACT-1", err: orders.ErrInvalidUnit},
		{line: "assemble laboratory-group on 5 with 100 LAB", err: orders.ErrInvalidUnit},
		{line: "assemble laboratory-group on 5 with 100 LAB-1 producing RSCH", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing RSCH-1", err: orders.ErrInvalidUnit},
		{
			line: "assemble factory-group on 5 with 1,000 fact-10 producing CNGD",
			want: producing(group(game.KindFactory, 0, 5, unit{Type: "FACT", TechLevel: 10, Qty: 1000}), unit{Type: "CNGD"}),
		},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing farm-1", want: producing(group(game.KindFactory, 0, 5, unit{Type: "FACT", TechLevel: 1, Qty: 100}), unit{Type: "FARM", TechLevel: 1})},
		{line: "assemble factory-group 4 on 5 add 100 FACT-1", want: group(game.KindFactory, 4, 5, unit{Type: "FACT", TechLevel: 1, Qty: 100})},
		// Every unit of a group is of one tech level, so an order takes one.
		{line: "assemble factory-group 4 on 5 add 100 FACT-1, 5 FACT-2", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with 100 FACT-1 , 5 FACT-2", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group 4 on 5 add 100 FACT-1 producing CNGD", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing FARM", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing FUEL", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing SLD", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with 100 FACT-1 producing", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FACT-1, producing CNGD", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with 100 FACT-1 50 FACT-2", err: orders.ErrInvalidOrder},
		{line: "assemble factory-group on 5 with 100 FARM-1", err: orders.ErrInvalidUnit},
		{line: "assemble factory-group on 5 with lots FACT-1", err: orders.ErrInvalidQuantity},
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
			if !reflect.DeepEqual(got.Order, tc.want) {
				t.Fatalf("order = %+v, want %+v", got.Order, tc.want)
			}
			if tc.want != nil {
				text := tc.want.(interface{ String() string }).String()
				again, err := orders.Parse(strings.NewReader(text))
				if err != nil || len(again) != 1 || !reflect.DeepEqual(again[0].Order, tc.want) {
					t.Errorf("String() = %q, which parses to %+v, %v", text, again, err)
				}
			}
		})
	}
}

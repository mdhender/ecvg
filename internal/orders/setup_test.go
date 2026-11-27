// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package orders_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

func TestParseSetup(t *testing.T) {
	unit := func(code string, tl, qty int) game.Unit { return game.Unit{Type: code, TechLevel: tl, Qty: qty} }
	ship := orders.Setup{Qty: 6, Type: game.CSHP, From: 29, MaxCWRK: 20, Each: true, Items: []orders.SetupItem{
		{Unit: unit("STRN", 0, 50000)},
		{Unit: unit("PRO", 0, 5), Race: 2},
		{Unit: unit("CWRK", 0, 3)},
		{Unit: unit("SDRV", 1, 1)},
		{Unit: unit("FACT", 1, 10), Stowed: true},
	}}
	for _, tc := range []struct {
		name string
		text string
		want orders.Order
		err  error
		// no and line are the number and text of the order's Line.
		no   int
		line string
	}{
		{
			name: "one line for each part",
			text: "# a ship\nSetup 6 cshp From 29 Using 20 CWKR Each // comment\n50,000 STRN\n\n5 pro-2\n3 cwkr\n1 SDRV-1\n10 FACT-1 Stowed\nEND\n",
			want: ship, no: 2,
			line: "Setup 6 cshp From 29 Using 20 CWKR Each; 50,000 STRN; 5 pro-2; 3 cwkr; 1 SDRV-1; 10 FACT-1 Stowed; end",
		},
		{
			name: "semicolons instead of lines",
			text: "setup 6 CSHP from 29 using 20 CWRK each; 50000 STRN;5 PRO-2 ; 3 CWRK\n1 SDRV-1; 10 FACT-1 stowed; end",
			want: ship, no: 1,
			line: "setup 6 CSHP from 29 using 20 CWRK each; 50000 STRN; 5 PRO-2; 3 CWRK; 1 SDRV-1; 10 FACT-1 stowed; end",
		},
		{
			name: "a colony with groups",
			text: "setup CENC from 3\n100 STRN\n10 FACT-2 producing CNGD\n5 MINE-1 working 7\n4 FARM-6\n2 LAB-3\n9 UEM\nend",
			want: orders.Setup{Qty: 1, Type: game.CENC, From: 3, Items: []orders.SetupItem{
				{Unit: unit("STRN", 0, 100)},
				{Unit: unit("FACT", 2, 10), Producing: game.Unit{Type: "CNGD"}},
				{Unit: unit("MINE", 1, 5), DepositNo: 7},
				{Unit: unit("FARM", 6, 4)},
				{Unit: unit("LAB", 3, 2)},
				{Unit: unit("UEM", 0, 9)},
			}},
			no:   1,
			line: "setup CENC from 3; 100 STRN; 10 FACT-2 producing CNGD; 5 MINE-1 working 7; 4 FARM-6; 2 LAB-3; 9 UEM; end",
		},
		{name: "using without each", text: "setup CORB from 3 using 5 CWRK; 1 PRO; end",
			want: orders.Setup{Qty: 1, Type: game.CORB, From: 3, MaxCWRK: 5, Items: []orders.SetupItem{{Unit: unit("PRO", 0, 1)}}},
			no:   1, line: "setup CORB from 3 using 5 CWRK; 1 PRO; end"},
		{name: "a quantity is for ships only", text: "setup 2 COPN from 3; 1 PRO; end", err: orders.ErrInvalidQuantity, no: 1},
		{name: "a quantity of 0", text: "setup 0 CSHP from 3; 1 PRO; end", err: orders.ErrInvalidQuantity, no: 1},
		{name: "not an entity type", text: "setup SHIP from 3; 1 PRO; end", err: orders.ErrInvalidEntityType, no: 1},
		{name: "no entity", text: "setup CSHP; 1 PRO; end", err: orders.ErrInvalidOrder, no: 1},
		{name: "bad entity ID", text: "setup CSHP from x; 1 PRO; end", err: orders.ErrInvalidEntityID, no: 1},
		{name: "using something else", text: "setup CSHP from 3 using 5 PRO; 1 PRO; end", err: orders.ErrInvalidUnit, no: 1},
		{name: "no units", text: "setup CSHP from 3\nend", err: orders.ErrInvalidOrder, no: 1},
		{name: "no end", text: "setup CSHP from 3\n1 PRO\n", err: orders.ErrNoEnd, no: 1},
		{name: "bad unit line", text: "setup CSHP from 3\n1 PRO\n5 WIDGET\nend", err: orders.ErrInvalidUnit, no: 1},
		{name: "people are not stowed", text: "setup CSHP from 3; 1 PRO stowed; end", err: orders.ErrInvalidOrder, no: 1},
		{name: "trainees stay", text: "setup CSHP from 3; 1 TRNE; end", err: orders.ErrInvalidUnit, no: 1},
		{name: "factories must say", text: "setup CENC from 3; 1 PRO; 5 FACT-1; end", err: orders.ErrInvalidOrder, no: 1},
		{name: "mines must say", text: "setup CENC from 3; 1 PRO; 5 MINE-1; end", err: orders.ErrInvalidOrder, no: 1},
		{name: "only factories produce", text: "setup CENC from 3; 1 PRO; 5 MINE-1 producing CNGD; end", err: orders.ErrInvalidOrder, no: 1},
		{name: "factories don't make research", text: "setup CENC from 3; 5 FACT-1 producing RSCH; end", err: orders.ErrInvalidUnit, no: 1},
		{name: "end takes nothing", text: "setup CSHP from 3; 1 PRO; end now", err: orders.ErrInvalidOrder, no: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lines, err := orders.Parse(strings.NewReader(tc.text))
			if err != nil {
				t.Fatal(err)
			}
			if len(lines) != 1 {
				t.Fatalf("Parse returned %d lines, want 1: %+v", len(lines), lines)
			}
			l := lines[0]
			if l.No != tc.no {
				t.Errorf("line number %d, want %d", l.No, tc.no)
			}
			if !errors.Is(l.Err, tc.err) || (tc.err == nil && l.Err != nil) {
				t.Fatalf("error = %v, want %v", l.Err, tc.err)
			}
			if tc.err != nil {
				return
			}
			if !reflect.DeepEqual(l.Order, tc.want) {
				t.Errorf("order = %+v\nwant    %+v", l.Order, tc.want)
			}
			if l.Text != tc.line {
				t.Errorf("text = %q\nwant   %q", l.Text, tc.line)
			}
			// What the order writes parses back to the same order.
			again, err := orders.Parse(strings.NewReader(l.Order.String()))
			if err != nil || len(again) != 1 || !reflect.DeepEqual(again[0].Order, tc.want) {
				t.Errorf("String() = %q, which parses to %+v (%v)", l.Order.String(), again, err)
			}
		})
	}
}

// A setup order ends at its end line, and the orders around it are read as
// usual. One with no end line ends at the first line that can't be part of
// it, which is read as the next order.
func TestParseSetupAmongOrders(t *testing.T) {
	const text = "set rations for 7 to 50%\n" +
		"setup CSHP from 3\n1 PRO\nend; set rations for 8 to 50%\n" +
		"setup CSHP from 4\n2 PRO\n" +
		"set rations for 9 to 50%\n" +
		"setup CSHP from 5; 3 PRO; setup CSHP from 6; 4 PRO; end\n" +
		"end\n" +
		"5 PRO\n"
	lines, err := orders.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		no   int
		text string
		bad  bool
	}
	var got []row
	for _, l := range lines {
		got = append(got, row{l.No, l.Text, l.Err != nil})
	}
	want := []row{
		{1, "set rations for 7 to 50%", false},
		{2, "setup CSHP from 3; 1 PRO; end", false},
		{4, "set rations for 8 to 50%", false},
		{5, "setup CSHP from 4; 2 PRO", true},
		{7, "set rations for 9 to 50%", false},
		{8, "setup CSHP from 5; 3 PRO", true},
		{8, "setup CSHP from 6; 4 PRO; end", false},
		{9, "end", true},
		{10, "5 PRO", true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Parse =\n%+v\nwant\n%+v", got, want)
	}
}

func TestParseCancelSetup(t *testing.T) {
	for _, tc := range []struct {
		text string
		want orders.Order
		err  error
	}{
		{text: "Cancel Setup 31", want: orders.CancelSetup{EntityID: 31}},
		{text: "cancel setup", err: orders.ErrInvalidOrder},
		{text: "cancel setup x", err: orders.ErrInvalidEntityID},
		{text: "cancel setup 31 now", err: orders.ErrInvalidOrder},
	} {
		lines, err := orders.Parse(strings.NewReader(tc.text))
		if err != nil || len(lines) != 1 {
			t.Fatalf("%q: %v, %d lines", tc.text, err, len(lines))
		}
		if l := lines[0]; !errors.Is(l.Err, tc.err) || (tc.err == nil && l.Err != nil) || !reflect.DeepEqual(l.Order, tc.want) {
			t.Errorf("%q: %+v, %v; want %+v, %v", tc.text, l.Order, l.Err, tc.want, tc.err)
		}
	}
}

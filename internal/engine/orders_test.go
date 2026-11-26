// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// twoEmpires returns a game in which empire 1 has colonies 1 and 2 and
// empire 2 has colony 3, all on full rations.
func twoEmpires() game.Data {
	return game.Data{
		Empires: []game.Empire{{ID: 1}, {ID: 2}},
		Colonies: []game.Colony{
			{ID: 1, EmpireID: 1, Rations: game.DefaultRations},
			{ID: 2, EmpireID: 1, Rations: game.DefaultRations},
			{ID: 3, EmpireID: 2, Rations: game.DefaultRations},
		},
	}
}

func file(t *testing.T, empireID int, text string) orders.File {
	t.Helper()
	lines, err := orders.Parse(strings.NewReader(text))
	if err != nil {
		t.Fatal(err)
	}
	return orders.File{EmpireID: empireID, Name: orders.FileName(empireID), Lines: lines}
}

func TestSetRations(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		empireID, entityID, percent int
		err                         error
	}{
		{"own colony", 1, 2, 50, nil},
		{"least", 1, 2, 1, nil},
		{"most", 2, 3, 500, nil},
		{"another empire's colony", 1, 3, 50, engine.ErrNotControlled},
		{"no such entity", 1, 9, 50, engine.ErrNotControlled},
		{"too little", 1, 2, 0, orders.ErrInvalidPercentage},
		{"too much", 1, 2, 501, orders.ErrInvalidPercentage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := twoEmpires()
			err := engine.SetRations(&d, tc.empireID, tc.entityID, tc.percent)
			if !errors.Is(err, tc.err) || (tc.err == nil && err != nil) {
				t.Fatalf("error = %v, want %v", err, tc.err)
			}
			for _, c := range d.Colonies {
				want := game.DefaultRations
				if tc.err == nil && c.ID == tc.entityID {
					want = tc.percent
				}
				if c.Rations != want {
					t.Errorf("colony %d: rations %d, want %d", c.ID, c.Rations, want)
				}
			}
		})
	}
}

func TestApplyOrders(t *testing.T) {
	d := twoEmpires()
	result, err := engine.ApplyOrders(&d, []orders.File{
		file(t, 1, "set rations for 1 to 50%\n"+
			"set rations for 3 to 1%   # not ours\n"+
			"set rations for 1 to 75%\n"+
			"dance\n"),
		file(t, 2, "SET RATIONS FOR 3 TO 200%\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{d.Colonies[0].Rations, d.Colonies[1].Rations, d.Colonies[2].Rations}; got[0] != 75 || got[1] != 100 || got[2] != 200 {
		t.Errorf("rations = %v, want [75 100 200]", got)
	}
	if result.Accepted != 3 || len(result.Rejected) != 2 {
		t.Fatalf("result = %+v, want 3 accepted and 2 rejected", result)
	}
	wantRecords := []game.OrderResult{
		{EmpireID: 1, Line: 1, Text: "set rations for 1 to 50%", Accepted: true},
		{EmpireID: 1, Line: 2, Text: "set rations for 3 to 1%   # not ours", Reason: "entity 3: " + engine.ErrNotControlled.Error()},
		{EmpireID: 1, Line: 3, Text: "set rations for 1 to 75%", Accepted: true},
		{EmpireID: 1, Line: 4, Text: "dance", Reason: orders.ErrUnknownOrder.Error()},
		{EmpireID: 2, Line: 1, Text: "SET RATIONS FOR 3 TO 200%", Accepted: true},
	}
	if !reflect.DeepEqual(d.OrderResults, wantRecords) {
		t.Errorf("order results =\n%+v\nwant\n%+v", d.OrderResults, wantRecords)
	}
	if _, err := engine.ApplyOrders(&d, nil); err != nil || len(d.OrderResults) != 0 {
		t.Errorf("after a turn with no orders: %d order results, %v; want none", len(d.OrderResults), err)
	}
	r := result.Rejected[0]
	if r.EmpireID != 1 || r.Line != 2 || !errors.Is(r.Err, engine.ErrNotControlled) {
		t.Errorf("first rejection = %+v, want empire 1 line 2 ErrNotControlled", r)
	}
	if want := "empire-001.txt:2: set rations for 3 to 1%   # not ours: entity 3: "; !strings.HasPrefix(r.String(), want) {
		t.Errorf("rejection = %q, want prefix %q", r.String(), want)
	}
	if r := result.Rejected[1]; r.Line != 4 || !errors.Is(r.Err, orders.ErrUnknownOrder) {
		t.Errorf("second rejection = %+v, want line 4 ErrUnknownOrder", r)
	}
}

func TestApplyOrdersUnknownEmpire(t *testing.T) {
	d := twoEmpires()
	_, err := engine.ApplyOrders(&d, []orders.File{
		file(t, 1, "set rations for 1 to 50%\n"),
		file(t, 3, "set rations for 3 to 50%\n"),
	})
	if !errors.Is(err, engine.ErrNoSuchEmpire) {
		t.Fatalf("error = %v, want ErrNoSuchEmpire", err)
	}
	if d.Colonies[0].Rations != game.DefaultRations {
		t.Errorf("colony 1: rations %d, want them unchanged", d.Colonies[0].Rations)
	}
}

func TestRunTurnAppliesOrders(t *testing.T) {
	d := twoEmpires()
	result, err := engine.RunTurn(&d, []orders.File{file(t, 2, "set rations for 3 to 25%\n")})
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted != 1 || d.Colonies[2].Rations != 25 || d.Game.Turn != 1 {
		t.Errorf("accepted %d, rations %d, turn %d; want 1, 25, 1", result.Accepted, d.Colonies[2].Rations, d.Game.Turn)
	}
}

func TestSetIdle(t *testing.T) {
	groups := func() game.Data {
		d := twoEmpires()
		d.Colonies[1].FarmingGroups = []game.Group{{No: 1}, {No: 2, IdlePercent: 10}}
		d.Colonies[1].MiningGroups = []game.MiningGroup{{No: 1}}
		d.Colonies[1].FactoryGroups = []game.Group{{No: 1}}
		d.Colonies[2].FarmingGroups = []game.Group{{No: 1}}
		return d
	}
	for _, tc := range []struct {
		name               string
		empireID, entityID int
		kind               game.GroupKind
		groupNo, percent   int
		err                error
	}{
		{"farming", 1, 2, game.KindFarming, 2, 94, nil},
		{"all of it", 1, 2, game.KindFarming, 1, 100, nil},
		{"back to work", 1, 2, game.KindFarming, 2, 0, nil},
		{"mining", 1, 2, game.KindMining, 1, 50, nil},
		{"factory", 1, 2, game.KindFactory, 1, 50, nil},
		{"no such group", 1, 2, game.KindFarming, 3, 50, engine.ErrNoSuchGroup},
		{"no groups of that kind", 1, 1, game.KindMining, 1, 50, engine.ErrNoSuchGroup},
		{"unknown kind", 1, 2, game.GroupKind("fishing"), 1, 50, engine.ErrNoSuchGroup},
		{"another empire's colony", 1, 3, game.KindFarming, 1, 50, engine.ErrNotControlled},
		{"no such entity", 1, 9, game.KindFarming, 1, 50, engine.ErrNotControlled},
		{"too little", 1, 2, game.KindFarming, 1, -1, orders.ErrInvalidPercentage},
		{"too much", 1, 2, game.KindFarming, 1, 101, orders.ErrInvalidPercentage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := groups()
			err := engine.SetIdle(&d, tc.empireID, tc.entityID, tc.kind, tc.groupNo, tc.percent)
			if !errors.Is(err, tc.err) || (tc.err == nil && err != nil) {
				t.Fatalf("error = %v, want %v", err, tc.err)
			}
			want := groups()
			if tc.err == nil {
				c := &want.Colonies[tc.entityID-1]
				switch tc.kind {
				case game.KindFarming:
					c.FarmingGroups[tc.groupNo-1].IdlePercent = tc.percent
				case game.KindMining:
					c.MiningGroups[tc.groupNo-1].IdlePercent = tc.percent
				case game.KindFactory:
					c.FactoryGroups[tc.groupNo-1].IdlePercent = tc.percent
				}
			}
			if !reflect.DeepEqual(d, want) {
				t.Errorf("colonies = %+v, want %+v", d.Colonies, want.Colonies)
			}
		})
	}
	t.Run("as an order", func(t *testing.T) {
		d := groups()
		result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, "set idle for farming group 1 of 2 to 94%\nset idle for mining group 4 of 2 to 5%")})
		if err != nil {
			t.Fatal(err)
		}
		if d.Colonies[1].FarmingGroups[0].IdlePercent != 94 || len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, engine.ErrNoSuchGroup) {
			t.Errorf("idle %d%%, rejected %+v; want 94%% and one order rejected for its group", d.Colonies[1].FarmingGroups[0].IdlePercent, result.Rejected)
		}
	})
}

func TestDraftIsNotCarriedOutYet(t *testing.T) {
	d := twoEmpires()
	want := twoEmpires()
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, "draft 5,000 CWRK from 1")})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, engine.ErrNotImplemented) || !reflect.DeepEqual(d.Colonies, want.Colonies) {
		t.Errorf("rejected %+v, colonies %+v; want the order rejected as not implemented and nothing changed", result.Rejected, d.Colonies)
	}
}

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
		{"least", 1, 2, 25, nil},
		{"most", 2, 3, 100, nil},
		{"another empire's colony", 1, 3, 50, engine.ErrNotControlled},
		{"no such entity", 1, 9, 50, engine.ErrNotControlled},
		{"too little", 1, 2, 24, orders.ErrInvalidPercentage},
		{"too much", 1, 2, 101, orders.ErrInvalidPercentage},
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
			"set rations for 3 to 25%   # not ours\n"+
			"set rations for 1 to 75%\n"+
			"dance\n"),
		file(t, 2, "SET RATIONS FOR 3 TO 90%\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := []int{d.Colonies[0].Rations, d.Colonies[1].Rations, d.Colonies[2].Rations}; got[0] != 75 || got[1] != 100 || got[2] != 90 {
		t.Errorf("rations = %v, want [75 100 90]", got)
	}
	if result.Accepted != 3 || len(result.Rejected) != 2 {
		t.Fatalf("result = %+v, want 3 accepted and 2 rejected", result)
	}
	wantRecords := []game.OrderResult{
		{EmpireID: 1, Line: 1, Text: "set rations for 1 to 50%", Accepted: true},
		{EmpireID: 1, Line: 2, Text: "set rations for 3 to 25%   # not ours", Reason: "entity 3: " + engine.ErrNotControlled.Error()},
		{EmpireID: 1, Line: 3, Text: "set rations for 1 to 75%", Accepted: true},
		{EmpireID: 1, Line: 4, Text: "dance", Reason: orders.ErrUnknownOrder.Error()},
		{EmpireID: 2, Line: 1, Text: "SET RATIONS FOR 3 TO 90%", Accepted: true},
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
	if want := "empire-001.txt:2: set rations for 3 to 25%   # not ours: entity 3: "; !strings.HasPrefix(r.String(), want) {
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

func TestDraftOrders(t *testing.T) {
	d := twoEmpires()
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300}
	want := d.Colonies[0]
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, `draft 5,000 CWRK from 1
draft 10 POL from 1
draft 10 USK from 1
draft 10 SLD from 3
draft 100 SLD from 1`)})
	if err != nil {
		t.Fatal(err)
	}
	// The orders phase only queues the drafts.
	if !reflect.DeepEqual(d.Colonies[0], want) {
		t.Errorf("the orders phase changed the colony: %+v", d.Colonies[0])
	}
	wantDrafts := []engine.DraftOrder{{EntityID: 1, Unit: "CWRK", Qty: 5000}, {EntityID: 1, Unit: "SLD", Qty: 100}}
	if !reflect.DeepEqual(result.Drafts, wantDrafts) || result.Accepted != 2 || len(result.Rejected) != 3 {
		t.Fatalf("drafts %+v, accepted %d, rejected %+v", result.Drafts, result.Accepted, result.Rejected)
	}
	for i, e := range []error{engine.ErrNotDraftable, engine.ErrNotDraftable, engine.ErrNotControlled} {
		if !errors.Is(result.Rejected[i].Err, e) {
			t.Errorf("rejection %d: %v, want %v", i, result.Rejected[i].Err, e)
		}
	}
	d.Colonies[1].Drafted = []game.Drafted{{Unit: "SLD", Asked: 1, Drafted: 1}} // last turn's
	engine.Draft(&d, result.Drafts)
	c := d.Colonies[0]
	if c.Cadres != (game.Cadres{CWRK: 300}) || c.Population != (game.Population{USK: 900, PRO: 300, SLD: 100}) {
		t.Errorf("cadres %+v, population %+v", c.Cadres, c.Population)
	}
	if got := []game.Drafted{{Unit: "CWRK", Asked: 5000, Drafted: 300}, {Unit: "SLD", Asked: 100, Drafted: 100}}; !reflect.DeepEqual(c.Drafted, got) {
		t.Errorf("drafted %+v, want %+v", c.Drafted, got)
	}
	if d.Colonies[1].Drafted != nil {
		t.Errorf("last turn's record was kept: %+v", d.Colonies[1].Drafted)
	}
}

// A draft comes after production: the people drafted work the turn they are
// drafted in and are out of the worker cadre from the next.
func TestDraftedPeopleWorkThisTurn(t *testing.T) {
	d := twoEmpires()
	d.Game.Turn = 1
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300, SLD: 50}
	d.Colonies[0].Cargo = []game.Unit{{Type: "FOOD", Qty: 10_000}} // or they starve
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "draft 200 CWRK from 1\ndraft 150 TRNE from 1\ndraft 40 SPY from 1")}); err != nil {
		t.Fatal(err)
	}
	c := d.Colonies[0]
	if c.Workers.PRO != 300 || c.Workers.USK != 1000 || c.Cadres != (game.Cadres{CWRK: 200, TRNE: 150, SPY: 40}) {
		t.Fatalf("turn 1: workers %+v, cadres %+v", c.Workers, c.Cadres)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	// 200 + 2 + 40 PRO and 200 + 150 USK are held by the cadres.
	if c = d.Colonies[0]; c.Workers.PRO != 58 || c.Workers.USK != 650 || c.Drafted != nil {
		t.Errorf("turn 2: workers %+v, drafted %+v; want 58 PRO and 650 USK", c.Workers, c.Drafted)
	}
	// At the end of that turn 5% of the 150 trainees, rounded down, became
	// PRO, and the TRNE cadre disbanded. The other cadres stay.
	if c.Trained != (game.Trained{Trainees: 150, Graduated: 7}) || c.Cadres != (game.Cadres{CWRK: 200, SPY: 40}) ||
		c.Population != (game.Population{USK: 993, PRO: 307, SLD: 50}) {
		t.Errorf("turn 2: trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	if c = d.Colonies[0]; c.Workers.PRO != 67 || c.Workers.USK != 793 || c.Trained != (game.Trained{}) {
		t.Errorf("turn 3: workers %+v, trained %+v; want 67 PRO and 793 USK", c.Workers, c.Trained)
	}
}

func TestDisbandOrders(t *testing.T) {
	d := twoEmpires()
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300, SLD: 50}
	d.Colonies[0].Cadres = game.Cadres{CWRK: 200, SPY: 10}
	d.Colonies[0].Disbanded = []game.Drafted{{Unit: "SPY", Asked: 1, Drafted: 1}} // last turn's
	result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, `disband 150 CWRK from 1
draft 30 CWRK from 1
disband 500 SPY from 1
disband 20 SLD from 1
disband 5 POL from 1
disband 5 CWRK from 3
draft 25 SPY from 1`)})
	if err != nil {
		t.Fatal(err)
	}
	if result.Accepted != 5 || len(result.Rejected) != 2 || d.Colonies[0].Cadres != (game.Cadres{CWRK: 200, SPY: 10}) {
		t.Fatalf("accepted %d, rejected %+v, cadres %+v", result.Accepted, result.Rejected, d.Colonies[0].Cadres)
	}
	for i, e := range []error{engine.ErrNotDraftable, engine.ErrNotControlled} {
		if !errors.Is(result.Rejected[i].Err, e) {
			t.Errorf("rejection %d: %v, want %v", i, result.Rejected[i].Err, e)
		}
	}
	engine.Draft(&d, result.Drafts)
	c := d.Colonies[0]
	// 25 SPY are drafted after the 10 were disbanded.
	if c.Cadres != (game.Cadres{CWRK: 80, SPY: 25}) || c.Population != (game.Population{USK: 1020, PRO: 300, SLD: 30}) {
		t.Errorf("cadres %+v, population %+v", c.Cadres, c.Population)
	}
	want := []game.Drafted{{Unit: "CWRK", Asked: 150, Drafted: 150}, {Unit: "SPY", Asked: 500, Drafted: 10}, {Unit: "SLD", Asked: 20, Drafted: 20}}
	if !reflect.DeepEqual(c.Disbanded, want) || len(c.Drafted) != 2 {
		t.Errorf("disbanded %+v, drafted %+v; want %+v and two drafts", c.Disbanded, c.Drafted, want)
	}
}

// Disbanded people worked or not as they were at the start of the turn; they
// are back in the worker cadre from the next.
func TestDisbandTakesEffectNextTurn(t *testing.T) {
	d := twoEmpires()
	d.Game.Turn = 1
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300}
	d.Colonies[0].Cadres = game.Cadres{CWRK: 200}
	d.Colonies[0].Cargo = []game.Unit{{Type: "FOOD", Qty: 10_000}}
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "disband 200 CWRK from 1")}); err != nil {
		t.Fatal(err)
	}
	if c := d.Colonies[0]; c.Workers.PRO != 100 || c.Workers.USK != 800 || c.Cadres != (game.Cadres{}) {
		t.Fatalf("turn 1: workers %+v, cadres %+v", c.Workers, c.Cadres)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	if c := d.Colonies[0]; c.Workers.PRO != 300 || c.Workers.USK != 1000 || c.Disbanded != nil {
		t.Errorf("turn 2: workers %+v, disbanded %+v", c.Workers, c.Disbanded)
	}
}

// A trainee cadre must exist at the start of a turn to be promoted in its
// demographic changes; nobody is promoted in the setup turn.
func TestTraineesArePromotedInDemographics(t *testing.T) {
	d := game.Data{Colonies: []game.Colony{
		{ID: 1, Population: game.Population{USK: 1000, PRO: 10}, Cadres: game.Cadres{TRNE: 300, NewTRNE: 100}},
		{ID: 2, Population: game.Population{USK: 1000, PRO: 10}, Cadres: game.Cadres{TRNE: 300, NewTRNE: 100}},
	}}
	engine.Demographics(&d, 3)
	if c := d.Colonies[0]; c.Trained != (game.Trained{Trainees: 200, Graduated: 10}) || c.Cadres != (game.Cadres{TRNE: 100}) ||
		c.Population != (game.Population{USK: 990, PRO: 20}) {
		t.Errorf("trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
	// The setup turn: the trainees stay, and all of them train in turn 1.
	setup := game.Data{Colonies: []game.Colony{
		{ID: 1, Population: game.Population{USK: 1000, PRO: 10}, Cadres: game.Cadres{TRNE: 300, NewTRNE: 100}, Trained: game.Trained{Trainees: 5}},
	}}
	engine.Demographics(&setup, 0)
	if c := setup.Colonies[0]; c.Trained != (game.Trained{}) || c.Cadres != (game.Cadres{TRNE: 300}) || c.Population != (game.Population{USK: 1000, PRO: 10}) {
		t.Errorf("setup turn: trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
}

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
	// Set rations orders wait for the Pay Change stage: until it runs they
	// are neither accepted nor rejected, and nothing has changed.
	if result.Pending() != 4 || result.Accepted != 0 || len(result.Rejected) != 1 || d.Colonies[0].Rations != game.DefaultRations {
		t.Fatalf("before the stage: result = %+v, rations %d; want 4 pending, 1 rejected, rations unchanged", result, d.Colonies[0].Rations)
	}
	engine.Transfers(&d, &result) // not this stage's orders
	if result.Pending() != 4 {
		t.Fatalf("the Transfer stage settled set rations orders: %d pending, want 4", result.Pending())
	}
	engine.PayChanges(&d, &result)
	if result.Pending() != 0 {
		t.Fatalf("after the stage: %d pending, want 0", result.Pending())
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

func TestUpdateGroup(t *testing.T) {
	us := func(qty int) []game.GroupUnits { return []game.GroupUnits{{Qty: qty, TechLevel: 1}} }
	cngd, strn, strl := game.Unit{Type: "CNGD"}, game.Unit{Type: "STRN"}, game.Unit{Type: "STRL"}
	groups := func() game.Data {
		d := twoEmpires()
		d.Colonies[1].FarmingGroups = []game.Group{{No: 1, Units: us(10)}, {No: 2, ShutDown: 4, Units: us(10)}}
		d.Colonies[1].MiningGroups = []game.MiningGroup{{No: 3, ShutDown: 7, Units: us(7)}}
		d.Colonies[1].FactoryGroups = []game.Group{
			{No: 4, Producing: cngd, Units: us(1000)},
			{No: 5, Producing: strn, Units: us(10)},
			{No: 6, Producing: cngd, Units: us(10)},
			{No: 7, Retooled: true},
		}
		d.Colonies[1].LaboratoryGroups = []game.LaboratoryGroup{{No: 8, ShutDown: 2, Units: us(10)}}
		d.Colonies[2].FarmingGroups = []game.Group{{No: 1, Units: us(10)}}
		return d
	}
	// The groups of colony 2 that the tests change.
	farm := func(d *game.Data, no int) *game.Group { return &d.Colonies[1].FarmingGroups[no-1] }
	fact := func(d *game.Data, no int) *game.Group { return &d.Colonies[1].FactoryGroups[no-4] }
	for _, tc := range []struct {
		name string
		text string
		// want makes the changes the order should, if it has no err.
		want func(d *game.Data)
		err  error
	}{
		{name: "retool", text: "update factory-group 4 of 2 retool",
			want: func(d *game.Data) { *fact(d, 4) = game.Group{No: 4, Retooled: true, Units: us(1000)} }},
		{name: "retool again", text: "update factory-group 7 of 2 retool", want: func(d *game.Data) {}},
		{name: "produce after retooling", text: "update factory-group 7 of 2 produce EWPN-1",
			want: func(d *game.Data) { *fact(d, 7) = game.Group{No: 7, Producing: game.Unit{Type: "EWPN", TechLevel: 1}} }},
		{name: "retool and produce in one turn", text: "update factory-group 4 of 2 retool\nupdate factory-group 4 of 2 produce STRN",
			want: func(d *game.Data) { fact(d, 4).Producing = strn }},
		{name: "produce without retooling", text: "update factory-group 4 of 2 produce EWPN-1", err: engine.ErrNotRetooled},
		{name: "produce what it makes already", text: "update factory-group 4 of 2 produce CNGD", want: func(d *game.Data) {}},
		{name: "factories don't do research", text: "update factory-group 4 of 2 produce RSCH", err: orders.ErrInvalidUnit},
		{name: "nor after retooling", text: "update factory-group 7 of 2 produce RSCH", err: orders.ErrInvalidUnit},
		{name: "one structural unit to another", text: "update factory-group 5 of 2 produce STRL",
			want: func(d *game.Data) { fact(d, 5).Producing = strl }},
		{name: "to a structural unit from another unit", text: "update factory-group 4 of 2 produce STRL", err: engine.ErrNotRetooled},
		{name: "only factories retool", text: "update farming-group 1 of 2 retool", err: orders.ErrInvalidOrder},
		{name: "laboratories don't retool", text: "update laboratory-group 8 of 2 retool", err: orders.ErrInvalidOrder},
		{name: "or produce", text: "update laboratory-group 8 of 2 produce RSCH", err: orders.ErrInvalidOrder},

		{name: "shutdown a quantity", text: "update factory-group 4 of 2 shutdown 500",
			want: func(d *game.Data) { fact(d, 4).ShutDown = 500 }},
		{name: "shutdown more", text: "update farming-group 2 of 2 shutdown 3",
			want: func(d *game.Data) { farm(d, 2).ShutDown = 7 }},
		{name: "shutdown takes what is running", text: "update farming-group 2 of 2 shutdown 3,000",
			want: func(d *game.Data) { farm(d, 2).ShutDown = 10 }},
		{name: "shutdown a percentage of the group, rounded up", text: "update farming-group 2 of 2 shutdown 25%",
			want: func(d *game.Data) { farm(d, 2).ShutDown = 7 }},
		{name: "shutdown all of it", text: "update farming-group 1 of 2 shutdown 100%",
			want: func(d *game.Data) { farm(d, 1).ShutDown = 10 }},
		{name: "shutdown with none running", text: "update mining-group 3 of 2 shutdown 1", err: engine.ErrNoneRunning},
		{name: "shutdown a group with no units", text: "update factory-group 7 of 2 shutdown 50%", err: engine.ErrNoneRunning},
		{name: "restart a quantity", text: "update mining-group 3 of 2 restart 2",
			want: func(d *game.Data) { d.Colonies[1].MiningGroups[0].ShutDown = 5 }},
		{name: "restart takes what is shut down", text: "update farming-group 2 of 2 restart 9",
			want: func(d *game.Data) { farm(d, 2).ShutDown = 0 }},
		{name: "restart a percentage of the group, rounded down", text: "update farming-group 2 of 2 restart 25%",
			want: func(d *game.Data) { farm(d, 2).ShutDown = 2 }},
		{name: "restart all of it", text: "update mining-group 3 of 2 restart 100%",
			want: func(d *game.Data) { d.Colonies[1].MiningGroups[0].ShutDown = 0 }},
		{name: "restart with none shut down", text: "update farming-group 1 of 2 restart 1", err: engine.ErrNoneShutDown},
		{name: "shutdown laboratories", text: "update laboratory-group 8 of 2 shutdown 50%",
			want: func(d *game.Data) { d.Colonies[1].LaboratoryGroups[0].ShutDown = 7 }},
		{name: "restart laboratories", text: "update laboratory-group 8 of 2 restart 1",
			want: func(d *game.Data) { d.Colonies[1].LaboratoryGroups[0].ShutDown = 1 }},
		{name: "a laboratory group is not a factory group", text: "update factory-group 8 of 2 shutdown 1", err: engine.ErrNoSuchGroup},
		{name: "shutdown and restart in one turn", text: "update factory-group 4 of 2 shutdown 25%\nupdate factory-group 4 of 2 restart 25%",
			want: func(d *game.Data) {}},

		{name: "no such group", text: "update farming-group 3 of 2 shutdown 1", err: engine.ErrNoSuchGroup},
		{name: "a group of another kind", text: "update mining-group 4 of 2 shutdown 1", err: engine.ErrNoSuchGroup},
		{name: "no groups of that kind", text: "update mining-group 1 of 1 shutdown 1", err: engine.ErrNoSuchGroup},
		{name: "another empire's colony", text: "update farming-group 1 of 3 shutdown 1", err: engine.ErrNotControlled},
		{name: "no such entity", text: "update farming-group 1 of 9 shutdown 1", err: engine.ErrNotControlled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := groups()
			result, err := engine.ApplyOrders(&d, []orders.File{file(t, 1, tc.text)})
			if err != nil {
				t.Fatal(err)
			}
			// Update group orders wait for the Build Change stage.
			engine.BuildChanges(&d, &result)
			want := groups()
			if tc.err == nil {
				if len(result.Rejected) != 0 {
					t.Fatalf("rejected %+v, want none", result.Rejected)
				}
				tc.want(&want)
			} else if len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, tc.err) {
				t.Fatalf("rejected %+v, want one order rejected with %v", result.Rejected, tc.err)
			}
			if !reflect.DeepEqual(d.Colonies, want.Colonies) {
				t.Errorf("colonies = %+v, want %+v", d.Colonies, want.Colonies)
			}
		})
	}
	t.Run("the engine checks what the parser does", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			o    orders.UpdateGroup
			err  error
		}{
			{"a farming group can't retool", orders.UpdateGroup{EntityID: 2, Kind: game.KindFarming, GroupNo: 1, Action: game.ActionRetool}, engine.ErrOnlyFactories},
			{"a mining group can't produce", orders.UpdateGroup{EntityID: 2, Kind: game.KindMining, GroupNo: 3, Action: game.ActionProduce, Producing: cngd}, engine.ErrOnlyFactories},
			{"factories don't make it", orders.UpdateGroup{EntityID: 2, Kind: game.KindFactory, GroupNo: 7, Action: game.ActionProduce, Producing: game.Unit{Type: "FUEL"}}, orders.ErrInvalidUnit},
			{"no action", orders.UpdateGroup{EntityID: 2, Kind: game.KindFactory, GroupNo: 4}, orders.ErrInvalidOrder},
			{"no amount", orders.UpdateGroup{EntityID: 2, Kind: game.KindFactory, GroupNo: 4, Action: game.ActionShutDown}, orders.ErrInvalidOrder},
			{"two amounts", orders.UpdateGroup{EntityID: 2, Kind: game.KindFactory, GroupNo: 4, Action: game.ActionShutDown, Qty: 1, Percent: 1}, orders.ErrInvalidOrder},
			{"too much", orders.UpdateGroup{EntityID: 2, Kind: game.KindFactory, GroupNo: 4, Action: game.ActionShutDown, Percent: 101}, orders.ErrInvalidOrder},
			{"unknown kind", orders.UpdateGroup{EntityID: 2, Kind: game.GroupKind("fishing"), GroupNo: 1, Action: game.ActionShutDown, Qty: 1}, engine.ErrNoSuchGroup},
		} {
			d := groups()
			if err := engine.UpdateGroup(&d, 1, tc.o); !errors.Is(err, tc.err) {
				t.Errorf("%s: error = %v, want %v", tc.name, err, tc.err)
			}
			if want := groups(); !reflect.DeepEqual(d, want) {
				t.Errorf("%s: colonies = %+v, want %+v", tc.name, d.Colonies, want.Colonies)
			}
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

// A draft comes before production (manual B3.0): the people drafted are out
// of the worker cadre in the turn they are drafted in, and the trainees train
// in it.
func TestDraftedPeopleLeaveWorkThisTurn(t *testing.T) {
	d := twoEmpires()
	d.Game.Turn = 1
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300, SLD: 50}
	d.Colonies[0].Cargo = []game.Unit{{Type: "FOOD", Qty: 10_000}} // or they starve
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "draft 200 CWRK from 1\ndraft 150 TRNE from 1\ndraft 40 SPY from 1")}); err != nil {
		t.Fatal(err)
	}
	// 200 + 2 + 40 PRO and 200 + 150 USK were held by the cadres.
	c := d.Colonies[0]
	if c.Workers.PRO != 58 || c.Workers.USK != 650 {
		t.Fatalf("turn 1: workers %+v; want 58 PRO and 650 USK", c.Workers)
	}
	// At the end of the turn 5% of the 150 trainees, rounded down, became
	// PRO, and the TRNE cadre disbanded. The other cadres stay.
	if c.Trained != (game.Trained{Trainees: 150, Graduated: 7}) || c.Cadres != (game.Cadres{CWRK: 200, SPY: 40}) ||
		c.Population != (game.Population{USK: 993, PRO: 307, SLD: 50}) {
		t.Errorf("turn 1: trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	if c = d.Colonies[0]; c.Workers.PRO != 67 || c.Workers.USK != 793 || c.Trained != (game.Trained{}) || c.Drafted != nil {
		t.Errorf("turn 2: workers %+v, trained %+v, drafted %+v; want 67 PRO and 793 USK", c.Workers, c.Trained, c.Drafted)
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

// A disband comes before production (manual B3.0): the people disbanded are
// back in the worker cadre in the turn they are disbanded in.
func TestDisbandTakesEffectThisTurn(t *testing.T) {
	d := twoEmpires()
	d.Game.Turn = 1
	d.Colonies[0].Population = game.Population{USK: 1000, PRO: 300}
	d.Colonies[0].Cadres = game.Cadres{CWRK: 200}
	d.Colonies[0].Cargo = []game.Unit{{Type: "FOOD", Qty: 10_000}}
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "disband 200 CWRK from 1")}); err != nil {
		t.Fatal(err)
	}
	if c := d.Colonies[0]; c.Workers.PRO != 300 || c.Workers.USK != 1000 || c.Cadres != (game.Cadres{}) {
		t.Fatalf("turn 1: workers %+v, cadres %+v", c.Workers, c.Cadres)
	}
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	if c := d.Colonies[0]; c.Workers.PRO != 300 || c.Workers.USK != 1000 || c.Disbanded != nil {
		t.Errorf("turn 2: workers %+v, disbanded %+v", c.Workers, c.Disbanded)
	}
}

// A trainee cadre is promoted in the demographic changes of the turn it
// trains in; nobody is promoted in the setup turn.
func TestTraineesArePromotedInDemographics(t *testing.T) {
	d := game.Data{Colonies: []game.Colony{
		{ID: 1, Population: game.Population{USK: 1000, PRO: 10}, Cadres: game.Cadres{TRNE: 300}},
	}}
	engine.Demographics(&d, 3)
	if c := d.Colonies[0]; c.Trained != (game.Trained{Trainees: 300, Graduated: 15}) || c.Cadres != (game.Cadres{}) ||
		c.Population != (game.Population{USK: 985, PRO: 25}) {
		t.Errorf("trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
	// The setup turn: the trainees stay, and train in turn 1.
	setup := game.Data{Colonies: []game.Colony{
		{ID: 1, Population: game.Population{USK: 1000, PRO: 10}, Cadres: game.Cadres{TRNE: 300}, Trained: game.Trained{Trainees: 5}},
	}}
	engine.Demographics(&setup, 0)
	if c := setup.Colonies[0]; c.Trained != (game.Trained{}) || c.Cadres != (game.Cadres{TRNE: 300}) || c.Population != (game.Population{USK: 1000, PRO: 10}) {
		t.Errorf("setup turn: trained %+v, cadres %+v, population %+v", c.Trained, c.Cadres, c.Population)
	}
}

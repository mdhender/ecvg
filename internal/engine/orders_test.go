// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"errors"
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

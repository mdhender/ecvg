// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestDraft(t *testing.T) {
	type pop = game.Population
	type cad = game.Cadres
	for _, tc := range []struct {
		name    string
		pop     pop
		cadres  cad
		unit    string
		qty     int
		want    int
		wantPop pop
		wantCad cad
	}{
		{"soldiers come from USK", pop{USK: 5000, PRO: 10}, cad{}, "SLD", 3600, 3600, pop{USK: 1400, PRO: 10, SLD: 3600}, cad{}},
		{"soldiers, short of USK", pop{UEM: 900, USK: 50}, cad{}, "SLD", 3600, 50, pop{UEM: 900, SLD: 50}, cad{}},
		{"soldiers leave the cadres their USK", pop{USK: 100}, cad{TRNE: 30}, "SLD", 100, 70, pop{USK: 30, SLD: 70}, cad{TRNE: 30}},
		{"construction workers", pop{USK: 900, PRO: 600}, cad{}, "CWRK", 500, 500, pop{USK: 900, PRO: 600}, cad{CWRK: 500}},
		{"construction workers, short of PRO", pop{USK: 900, PRO: 200}, cad{}, "CWRK", 500, 200, pop{USK: 900, PRO: 200}, cad{CWRK: 200}},
		{"construction workers, short of USK", pop{USK: 90, PRO: 200}, cad{}, "CWRK", 500, 90, pop{USK: 90, PRO: 200}, cad{CWRK: 90}},
		{"construction workers, added to", pop{USK: 900, PRO: 600}, cad{CWRK: 550}, "CWRK", 500, 50, pop{USK: 900, PRO: 600}, cad{CWRK: 600}},
		{"100 trainees need 1 PRO", pop{USK: 500, PRO: 1}, cad{}, "TRNE", 100, 100, pop{USK: 500, PRO: 1}, cad{TRNE: 100}},
		{"1 trainee needs 1 PRO", pop{USK: 500, PRO: 1}, cad{}, "TRNE", 1, 1, pop{USK: 500, PRO: 1}, cad{TRNE: 1}},
		{"trainees, short of PRO", pop{USK: 50000, PRO: 3}, cad{}, "TRNE", 16880, 300, pop{USK: 50000, PRO: 3}, cad{TRNE: 300}},
		{"trainees, no PRO", pop{USK: 500}, cad{}, "TRNE", 1, 0, pop{USK: 500}, cad{}},
		{"trainees, short of USK", pop{USK: 150, PRO: 9}, cad{}, "TRNE", 400, 150, pop{USK: 150, PRO: 9}, cad{TRNE: 150}},
		{"trainees fill a trainer's class first", pop{USK: 500, PRO: 1}, cad{TRNE: 40}, "TRNE", 400, 60, pop{USK: 500, PRO: 1}, cad{TRNE: 100}},
		{"trainees, PRO held by another cadre", pop{USK: 500, PRO: 5}, cad{CWRK: 4}, "TRNE", 400, 100, pop{USK: 500, PRO: 5}, cad{CWRK: 4, TRNE: 100}},
		{"spies", pop{PRO: 50, SLD: 80}, cad{}, "SPY", 40, 40, pop{PRO: 50, SLD: 80}, cad{SPY: 40}},
		{"spies, short of SLD", pop{PRO: 50, SLD: 8}, cad{}, "SPY", 40, 8, pop{PRO: 50, SLD: 8}, cad{SPY: 8}},
		{"spies, short of PRO", pop{PRO: 5, SLD: 80}, cad{CWRK: 2}, "SPY", 40, 3, pop{PRO: 5, SLD: 80}, cad{CWRK: 2, SPY: 3}},
		{"not draftable", pop{USK: 500, PRO: 500}, cad{}, "POL", 40, 0, pop{USK: 500, PRO: 500}, cad{}},
		{"nothing asked", pop{USK: 500, PRO: 500}, cad{}, "CWRK", -4, 0, pop{USK: 500, PRO: 500}, cad{}},
	} {
		c := game.Colony{Population: tc.pop, Cadres: tc.cadres}
		if got := c.Draft(tc.unit, tc.qty); got != tc.want || c.Population != tc.wantPop || c.Cadres != tc.wantCad {
			t.Errorf("%s: drafted %d, leaving %+v and %+v; want %d, %+v and %+v", tc.name, got, c.Population, c.Cadres, tc.want, tc.wantPop, tc.wantCad)
		}
	}
}

func TestCadres(t *testing.T) {
	k := game.Cadres{CWRK: 200, TRNE: 150, SPY: 40}
	if k.PRO() != 242 || k.USK() != 350 || k.SLD() != 40 {
		t.Errorf("holds %d PRO, %d USK, %d SLD; want 242, 350, 40", k.PRO(), k.USK(), k.SLD())
	}
	c := game.Colony{Population: game.Population{USK: 1000, PRO: 300, SLD: 50}, Cadres: k}
	if c.FreePRO() != 58 || c.FreeUSK() != 650 || c.FreeSLD() != 10 {
		t.Errorf("free: %d PRO, %d USK, %d SLD", c.FreePRO(), c.FreeUSK(), c.FreeSLD())
	}
	if w := game.NewWorkers(c); w.PRO != 58 || w.USK != 650 {
		t.Errorf("workers %+v, want 58 PRO and 650 USK", w)
	}
	for _, tc := range []struct {
		pop  game.Population
		want game.Cadres
	}{
		{game.Population{USK: 1000, PRO: 300, SLD: 50}, k},
		{game.Population{USK: 1000, PRO: 300, SLD: 30}, game.Cadres{CWRK: 200, TRNE: 150, SPY: 30}},
		{game.Population{USK: 1000, PRO: 100, SLD: 50}, game.Cadres{CWRK: 60, SPY: 40}},
		{game.Population{USK: 1000, PRO: 241, SLD: 50}, game.Cadres{CWRK: 200, TRNE: 100, SPY: 40}},
		{game.Population{USK: 250, PRO: 300, SLD: 50}, game.Cadres{CWRK: 200, TRNE: 50, SPY: 40}},
		{game.Population{}, game.Cadres{}},
	} {
		if got := k.Fit(tc.pop); got != tc.want {
			t.Errorf("Fit(%+v) = %+v, want %+v", tc.pop, got, tc.want)
		}
	}
}

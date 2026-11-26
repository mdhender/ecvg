// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestChangePopulation(t *testing.T) {
	for _, tc := range []struct {
		name string
		turn int
		p    game.Population
		want game.Population
	}{
		{
			name: "UEM over 30 percent: 2 percent become USK",
			turn: 1,
			p:    game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
			want: game.Population{UEM: 588000, USK: 312000, PRO: 200000, SLD: 100000},
		},
		{
			name: "UEM at exactly 30 percent: no change",
			turn: 1,
			p:    game.Population{UEM: 30, USK: 40, PRO: 20, SLD: 10},
			want: game.Population{UEM: 30, USK: 40, PRO: 20, SLD: 10},
		},
		{
			name: "UEM just over 30 percent: rounds up to one unit",
			turn: 1,
			p:    game.Population{UEM: 31, USK: 39, PRO: 20, SLD: 10},
			want: game.Population{UEM: 30, USK: 40, PRO: 20, SLD: 10},
		},
		{
			name: "2 percent of 101 UEM rounds up to 3",
			turn: 1,
			p:    game.Population{UEM: 101, USK: 1},
			want: game.Population{UEM: 98, USK: 4},
		},
		{
			name: "turn 4 is an end-of-year turn, but the prior turn was not",
			turn: 4,
			p:    game.Population{USK: 100, SLD: 100},
			want: game.Population{USK: 100, SLD: 100},
		},
		{
			name: "after an end-of-year turn: 5 percent of SLD become PRO",
			turn: 5,
			p:    game.Population{USK: 100, PRO: 7, SLD: 100},
			want: game.Population{USK: 100, PRO: 12, SLD: 95},
		},
		{
			name: "5 percent of 101 SLD rounds up to 6",
			turn: 9,
			p:    game.Population{USK: 500, SLD: 101},
			want: game.Population{USK: 500, PRO: 6, SLD: 95},
		},
		{
			name: "turn 1 follows turn 0, which is not an end-of-year turn",
			turn: 1,
			p:    game.Population{USK: 100, SLD: 100},
			want: game.Population{USK: 100, SLD: 100},
		},
		{
			name: "both changes in one turn",
			turn: 5,
			p:    game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
			want: game.Population{UEM: 588000, USK: 312000, PRO: 205000, SLD: 95000},
		},
		{
			name: "turn 0 is the setup turn: no change",
			turn: 0,
			p:    game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
			want: game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
		},
		{
			name: "no population",
			turn: 5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := engine.ChangePopulation(tc.p, tc.turn)
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
			if got.Units() != tc.p.Units() {
				t.Errorf("total changed from %d to %d", tc.p.Units(), got.Units())
			}
		})
	}
}

func TestDemographics(t *testing.T) {
	d := game.Data{Colonies: []game.Colony{
		{ID: 1, Population: game.Population{UEM: 100}},
		{ID: 2, Population: game.Population{USK: 100, SLD: 40}},
	}}
	engine.Demographics(&d, 5)
	want := []game.Population{{UEM: 98, USK: 2}, {USK: 100, PRO: 2, SLD: 38}}
	for i, c := range d.Colonies {
		if c.Population != want[i] {
			t.Errorf("colony %d: got %+v, want %+v", c.ID, c.Population, want[i])
		}
	}
}

func TestRunTurn(t *testing.T) {
	d := game.Data{Colonies: []game.Colony{{ID: 1, Population: game.Population{UEM: 100, SLD: 100}}}}
	for _, want := range []struct {
		turn int
		p    game.Population
	}{
		{1, game.Population{UEM: 100, SLD: 100}},                // turn 0, the setup turn: no changes
		{2, game.Population{UEM: 98, USK: 2, SLD: 100}},         // turn 1
		{3, game.Population{UEM: 96, USK: 4, SLD: 100}},         // turn 2
		{4, game.Population{UEM: 94, USK: 6, SLD: 100}},         // turn 3
		{5, game.Population{UEM: 92, USK: 8, SLD: 100}},         // turn 4, the end of the year
		{6, game.Population{UEM: 90, USK: 10, PRO: 5, SLD: 95}}, // turn 5 follows it
	} {
		if _, err := engine.RunTurn(&d, nil); err != nil {
			t.Fatal(err)
		}
		if d.Game.Turn != want.turn || d.Colonies[0].Population != want.p {
			t.Fatalf("got turn %d, %+v; want turn %d, %+v", d.Game.Turn, d.Colonies[0].Population, want.turn, want.p)
		}
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/report"
)

// sample returns a game with two empires on one home world; only the first
// has a colony.
func sample() game.Data {
	return game.Data{
		Game:    game.Game{Code: "alpha"},
		Stellia: []game.Stellium{{ID: 4, X: -3, Y: 4, Z: 12, HomeStellium: true}},
		Stars:   []game.Star{{ID: 9, StelliumID: 4, SeqNo: 1, HomeStar: true}},
		Orbits:  []game.Orbit{{ID: 83, StarID: 9, OrbitNo: 3}, {ID: 84, StarID: 9, OrbitNo: 4}},
		Planets: []game.Planet{{ID: 40, OrbitID: 83, Habitable: true, Viability: 25}, {ID: 41, OrbitID: 84}},
		Deposits: []game.Deposit{
			{ID: 1, PlanetID: 40, DepositNo: 1, Type: game.GOLD, Size: 12000000, Yield: 1},
			{ID: 2, PlanetID: 40, DepositNo: 2, Type: game.FUEL, Size: 20000000, Yield: 13},
			{ID: 3, PlanetID: 41, DepositNo: 1, Type: game.NMTL, Size: 77000000, Yield: 9},
		},
		Races: []game.Race{{ID: 40, HomeWorldID: 40}},
		Empires: []game.Empire{
			{ID: 1, Handle: "abcdefgh", Name: "Empire #1", RaceID: 40, HomeWorldID: 40},
			{ID: 2, Name: "Empire #2", RaceID: 40, HomeWorldID: 40},
		},
		Colonies: []game.Colony{{
			ID: 7, EmpireID: 1, PlanetID: 40, No: 1, Type: game.COPN, TechLevel: 1,
			Population: game.Population{UEM: 600000, USK: 300000, PRO: 200000, SLD: 100000},
			Rations:    game.DefaultRations,
			SOL:        game.DefaultSOL, DeathRate: game.DefaultDeathRate, BirthRate: game.DefaultBirthRate,
			Operational:   []game.Unit{{Type: "FARM", TechLevel: 10, Qty: 100000}},
			Cargo:         []game.Unit{{Type: "FOOD", Qty: 4269990}},
			FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 100000, TechLevel: 10}}}},
		}},
	}
}

func TestEmpire(t *testing.T) {
	for _, tc := range []struct {
		name     string
		empireID int
		edit     func(d *game.Data)
		want     []string
		lack     []string
	}{
		{
			name:     "empire with an open-air colony",
			empireID: 1,
			want: []string{
				"EMPYREAN CHALLENGE   Game alpha   Turn 0",
				"Empire 1: Empire #1",
				"Player: abcdefgh",
				"Home world: planet 40 at (-3, 4, 12) star 1 orbit 3",
				"COLONY 1   COPN (open-air surface colony)   Tech level 1",
				"Entity ID: 7",
				"Rations 100%   Standard of living 1.00   Birth rate 0.625%   Death rate 0.625%",
				"  UEM             600,000      60,000,000",
				"  Total         1,200,000     120,000,000",
				"Food needed each turn: 300,000 FOOD",
				"  FARM       10        100,000",
				"  FOOD               4,269,990",
				"  Group 1          100,000 units at TL 10",
				"MINING GROUPS\n  None assembled",
				"FACTORY GROUPS\n  None assembled",
				"    1  GOLD      12,000,000     1%",
				"    2  FUEL      20,000,000    13%",
			},
			lack: []string{"77,000,000", "Life support", "ORDERS RECEIVED"},
		},
		{
			name:     "enclosed colony needs space and life support",
			empireID: 1,
			edit:     func(d *game.Data) { d.Colonies[0].Type = game.CENC },
			want:     []string{"CENC (enclosed surface colony)", "Space taken: 1,200,000 VU   Life support needed: 1,200,000 LU"},
		},
		{
			name:     "empire with no player or colonies",
			empireID: 2,
			want:     []string{"Player: (none)", "No colonies."},
			lack:     []string{"COLONY 1"},
		},
		{
			name:     "later turn with no orders",
			empireID: 1,
			edit:     func(d *game.Data) { d.Game.Turn = 7 },
			want:     []string{"Turn 7", "ORDERS RECEIVED FOR TURN 6", "  None\n"},
		},
		{
			name:     "orders received",
			empireID: 1,
			edit: func(d *game.Data) {
				d.Game.Turn = 1
				d.OrderResults = []game.OrderResult{
					{EmpireID: 1, Line: 2, Text: "set rations for 7 to 50%", Accepted: true},
					{EmpireID: 2, Line: 1, Text: "set rations for 7 to 1%", Reason: "not theirs"},
					{EmpireID: 1, Line: 5, Text: "dance", Reason: "unknown order"},
				}
			},
			want: []string{
				"ORDERS RECEIVED FOR TURN 0",
				"  Line 2: set rations for 7 to 50%\n    Accepted",
				"  Line 5: dance\n    REJECTED: unknown order",
			},
			lack: []string{"not theirs", "to 1%"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := sample()
			if tc.edit != nil {
				tc.edit(&d)
			}
			var sb strings.Builder
			if err := report.Empire(&sb, d, tc.empireID); err != nil {
				t.Fatal(err)
			}
			for _, want := range tc.want {
				if !strings.Contains(sb.String(), want) {
					t.Errorf("report lacks %q:\n%s", want, sb.String())
				}
			}
			for _, lack := range tc.lack {
				if strings.Contains(sb.String(), lack) {
					t.Errorf("report has %q", lack)
				}
			}
		})
	}
}

func TestEmpireErrors(t *testing.T) {
	for _, tc := range []struct {
		name     string
		empireID int
		edit     func(d *game.Data)
	}{
		{"no such empire", 9, func(d *game.Data) {}},
		{"home world missing", 1, func(d *game.Data) { d.Planets = nil }},
		{"star missing", 1, func(d *game.Data) { d.Stars = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := sample()
			tc.edit(&d)
			var sb strings.Builder
			if err := report.Empire(&sb, d, tc.empireID); err == nil {
				t.Error("got nil error, want one")
			}
		})
	}
}

func TestWriteAll(t *testing.T) {
	dir := t.TempDir()
	d := sample()
	if err := d.Write(dir); err != nil {
		t.Fatal(err)
	}
	for range 2 { // the second time replaces the reports
		out, n, err := report.WriteAll(dir, d)
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(dir, "turns", "0", "reports"); out != want || n != 2 {
			t.Errorf("WriteAll = %s, %d; want %s, 2", out, n, want)
		}
	}
	for id, want := range map[int]string{1: "Player: abcdefgh", 2: "Player: (none)"} {
		data, err := os.ReadFile(filepath.Join(dir, "turns", "0", "reports", report.FileName(id)))
		if err != nil || !strings.Contains(string(data), want) {
			t.Errorf("report %d: %v, lacks %q", id, err, want)
		}
	}
	if _, turns, err := game.Turns(dir); err != nil || len(turns) != 1 || turns[0].Reports != 2 {
		t.Errorf("Turns = %+v, %v; want one turn with 2 reports", turns, err)
	}
}

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
				"Food eaten each turn at 100% rations: 300,000 FOOD",
				"  FARM       10        100,000",
				"  FOOD               4,269,990",
				"  Group 1          100,000 units at TL 10",
				"MINING GROUPS\n  None assembled",
				"FACTORY GROUPS\n  None assembled",
				"    1  GOLD      12,000,000     1%",
				"    2  FUEL      20,000,000    13%",
			},
			lack: []string{"77,000,000", "Life support", "ORDERS RECEIVED", "STARVED"},
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
			name:     "short rations and starvation",
			empireID: 1,
			edit: func(d *game.Data) {
				d.Colonies[0].Rations = 20
				d.Colonies[0].Starved = game.Population{UEM: 1500, SLD: 20}
			},
			want: []string{
				"Food eaten each turn at 20% rations: 60,000 FOOD",
				"STARVED LAST TURN: 1,520 units (UEM 1,500, USK 0, PRO 0, SLD 20)",
			},
		},
		{
			name:     "harvest",
			empireID: 1,
			edit: func(d *game.Data) {
				d.Colonies[0].Harvest = game.Harvest{Units: 100000, Worked: 36000, Unstaffed: 4000, Unfuelled: 60000, Ruined: 1500, Fuel: 360000, Food: 1800000}
				d.Colonies[0].FarmingGroups = append(d.Colonies[0].FarmingGroups,
					game.Group{No: 2, IdlePercent: 30, Units: []game.GroupUnits{{Qty: 5000, TechLevel: 1, Pipeline: []int{0, 2000, 1000},
						Idle: game.Idle{Ordered: 1500, Unstaffed: 400, Unfuelled: 100}}}})
			},
			want: []string{
				"  Group 2   ORDERED IDLE: 30% of its units\n  Group 2            5,000 units at TL 1\n" +
					"                     1,500 of them were IDLE last turn: BY ORDER\n" +
					"                       400 of them were IDLE last turn: NO STAFF\n" +
					"                       100 of them were IDLE last turn: NO FUEL\n" +
					"                     2,000 of them deliver their crop after 2 more turns of work\n                     1,000 of them deliver their crop after 1 more turn of work\n",
				"  1,500 idle FARM-1 units LOST THE CROP they had begun",
				"  LAST TURN: 36,000 of 100,000 units worked, burned 360,000 FUEL, and made 1,800,000 FOOD",
				"  4,000 units idle: no staff (each needs 1 PRO and 3 USK)",
				"  60,000 units idle: no FUEL",
			},
		},
		{
			name:     "no harvest",
			empireID: 1,
			lack:     []string{"LAST TURN", "units idle", "IDLE"},
		},
		{
			name:     "mining and the depot",
			empireID: 1,
			edit: func(d *game.Data) {
				c := &d.Colonies[0]
				c.Depot = []game.Unit{{Type: "FUEL", Qty: 3250000}}
				c.DepotOperational = []game.Unit{{Type: "MINE", TechLevel: 10, Qty: 100000}}
				c.MiningGroups = []game.MiningGroup{{
					No: 1, DepositID: 2, IdlePercent: 25,
					Units: []game.GroupUnits{{Qty: 100000, TechLevel: 10, Idle: game.Idle{Ordered: 25000, Unstaffed: 50000}}},
					Mined: game.Mined{Worked: 25000, Fuel: 125000, Product: 812500, Returned: 1000},
				}}
			},
			want: []string{
				"DEPOT, STORED\n  Unit       TL            Qty\n  FUEL               3,250,000\n",
				"DEPOT, OPERATIONAL\n  Unit       TL            Qty\n  MINE       10        100,000\n",
				"            1,000 FUEL RETURNED TO THE DEPOSIT: no room to store it",
				// The sample colony has no structure.
				"SPACE                  VU\n  Capacity              0\n  Used         15,409,970\n  Over by      15,409,970   OVER CAPACITY: nothing more can be stored in cargo\n",
				"  Group 1   ORDERED IDLE: 25% of its units\n  Group 1          100,000 units at TL 10 working deposit 2 (FUEL)\n" +
					"                    25,000 of them were IDLE last turn: BY ORDER\n" +
					"                    50,000 of them were IDLE last turn: NO STAFF\n" +
					"            LAST TURN: 25,000 units worked, burned 125,000 FUEL, and mined 812,500 FUEL\n",
			},
		},
		{
			name:     "space and lost food",
			empireID: 1,
			edit: func(d *game.Data) {
				c := &d.Colonies[0]
				c.Operational = append(c.Operational, game.Unit{Type: "STRL", TechLevel: 10, Qty: 200000})
				c.Harvest = game.Harvest{Units: 100000, Worked: 100000, Fuel: 1000000, Food: 4000000, Lost: 1000000}
			},
			want: []string{
				"SPACE                  VU\n  Capacity     20,000,000\n  Used         15,409,970\n  Free          4,590,030\n",
				"  1,000,000 FOOD LOST: no room to store it",
			},
			lack: []string{"OVER CAPACITY"},
		},
		{
			name:     "workers",
			empireID: 1,
			edit: func(d *game.Data) {
				d.Colonies[0].Workers = game.Workers{PRO: 200000, USK: 400000, AutoUSK: 100000, IdlePRO: 100000, IdleUSK: 100000}
			},
			want: []string{
				"WORKERS LAST TURN    Pool           Used           Idle\n",
				"  PRO             200,000        100,000        100,000\n",
				"  USK             400,000        300,000        100,000\n",
				"  100,000 of the USK were automation (AUTO)",
			},
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

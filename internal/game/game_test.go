// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

// sampleData returns a tiny game with a record of every kind.
func sampleData() game.Data {
	return game.Data{
		// The top bit of Seed1 is set, so a float64 JSON reader would lose precision.
		Game:     game.Game{Code: "alpha", Seed1: 0xfedcba9876543210, Seed2: 1},
		Stellia:  []game.Stellium{{ID: 1, X: 1, Y: -2, Z: 3, HomeStellium: true}},
		Stars:    []game.Star{{ID: 1, StelliumID: 1, SeqNo: 1, HomeStar: true}},
		Orbits:   []game.Orbit{{ID: 1, StarID: 1, OrbitNo: 3}},
		Planets:  []game.Planet{{ID: 1, OrbitID: 1, Habitable: true, Viability: 25}},
		Deposits: []game.Deposit{{ID: 1, PlanetID: 1, DepositNo: 1, Type: game.FUEL, Size: 1_000_000, Yield: 5}},
		Races:    []game.Race{{ID: 1, HomeWorldID: 1}},
		Empires:  []game.Empire{{ID: 1, Name: "Empire #1", RaceID: 1, HomeWorldID: 1}},
		Colonies: []game.Colony{{
			ID: 1, EmpireID: 1, PlanetID: 1, No: 1, Type: "COPN", TechLevel: 1,
			Population: game.Population{UEM: 4, USK: 3, PRO: 2, SLD: 1},
			SOL:        game.DefaultSOL, DeathRate: game.DefaultDeathRate, BirthRate: game.DefaultBirthRate,
			Operational:   []game.Unit{{Type: "FARM", TechLevel: 1, Qty: 10}},
			Cargo:         []game.Unit{{Type: "FOOD", Qty: 20}},
			FarmingGroups: []game.Group{{No: 1, Units: []game.GroupUnits{{Qty: 10, TechLevel: 1}}}},
			MiningGroups:  []game.MiningGroup{{No: 1, DepositID: 1, Units: []game.GroupUnits{{Qty: 5, TechLevel: 1}}}},
			FactoryGroups: []game.Group{},
		}},
		OrderResults: []game.OrderResult{
			{EmpireID: 1, Line: 1, Text: "set rations for 1 to 50%", Accepted: true},
			{EmpireID: 1, Line: 3, Text: "dance", Reason: "unknown order"},
		},
	}
}

// fileNames are the files Write creates, relative to the game's folder.
var fileNames = []string{
	game.FileName, game.StelliaFileName, game.StarsFileName, game.OrbitsFileName, game.PlanetsFileName,
	game.RacesFileName,
	filepath.Join("turns", "0", game.DepositsFileName),
	filepath.Join("turns", "0", game.EmpiresFileName),
	filepath.Join("turns", "0", game.ColoniesFileName),
	filepath.Join("turns", "0", game.OrderResultsFileName),
}

func TestWriteThenLoadData(t *testing.T) {
	dir := t.TempDir()
	want := sampleData()
	if err := want.Write(dir); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, game.FileName))
	if err != nil {
		t.Fatal(err)
	}
	const wantJSON = `{"code":"alpha","seed1":"fedcba9876543210","seed2":"0000000000000001","turn":0}` + "\n"
	if string(data) != wantJSON {
		t.Errorf("game.json =\n%s\nwant\n%s", data, wantJSON)
	}

	got, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadData =\n%+v\nwant\n%+v", got, want)
	}
	if g, err := game.Load(dir); err != nil || g != want.Game {
		t.Errorf("Load = %+v, %v; want %+v", g, err, want.Game)
	}
}

func TestWriteWritesAllFilesOrNone(t *testing.T) {
	for _, existing := range fileNames {
		t.Run(existing+" exists", func(t *testing.T) {
			dir := t.TempDir()
			const contents = "keep me"
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, existing)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, existing), []byte(contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := sampleData().Write(dir); !errors.Is(err, game.ErrGameExists) {
				t.Fatalf("Write = %v, want ErrGameExists", err)
			}
			if data, err := os.ReadFile(filepath.Join(dir, existing)); err != nil || string(data) != contents {
				t.Errorf("%s changed: %q, %v", existing, data, err)
			}
			for _, name := range fileNames {
				if name == existing {
					continue
				}
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("%s was left behind: stat = %v", name, err)
				}
			}
			// Write removes the turn folders only if it made them.
			_, err := os.Stat(filepath.Join(dir, "turns"))
			if made := filepath.Dir(existing) == "."; made != errors.Is(err, fs.ErrNotExist) {
				t.Errorf("turns folder: stat = %v, want it removed: %v", err, made)
			}
		})
	}
}

func TestWriteDoesNotCreateFolders(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	if err := sampleData().Write(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Write = %v, want fs.ErrNotExist", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Write made the folder: stat = %v", err)
	}
}

func TestCode(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	alpha := filepath.Join(root, "alpha")
	if err := os.Mkdir(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		dir  string
		want string
		err  error
	}{
		{"folder", alpha, "alpha", nil},
		{"trailing separator", alpha + string(filepath.Separator), "alpha", nil},
		{"missing folder", filepath.Join(root, "missing"), "", fs.ErrNotExist},
		{"path is a file", file, "", game.ErrNotDirectory},
		{"filesystem root", string(filepath.Separator), "", game.ErrInvalidCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := game.Code(tc.dir)
			if got != tc.want || !errors.Is(err, tc.err) {
				t.Errorf("Code = %q, %v; want %q, %v", got, err, tc.want, tc.err)
			}
		})
	}
}

func TestSeedUnmarshalText(t *testing.T) {
	for _, tc := range []struct {
		text string
		want game.Seed
		ok   bool
	}{
		{"0000000000000000", 0, true},
		{"ffffffffffffffff", 0xffffffffffffffff, true},
		{"0123456789abcdef", 0x0123456789abcdef, true},
		{"", 0, false},
		{"123", 0, false},
		{"0x23456789abcdef", 0, false},
		{"0123456789abcdefa", 0, false},
		{"0123456789abcdeg", 0, false},
		{"+123456789abcdef", 0, false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			var s game.Seed
			err := s.UnmarshalText([]byte(tc.text))
			if tc.ok && (err != nil || s != tc.want) {
				t.Errorf("got %#x, %v; want %#x", uint64(s), err, uint64(tc.want))
			} else if !tc.ok && !errors.Is(err, game.ErrInvalidSeed) {
				t.Errorf("err = %v, want ErrInvalidSeed", err)
			}
		})
	}
}

func TestLoadRejectsBadJSON(t *testing.T) {
	for name, data := range map[string]string{
		"numeric seed":   `{"code":"a","seed1":18364758544493064720,"seed2":"0000000000000000"}`,
		"unknown member": `{"code":"a","seed1":"0000000000000000","seed2":"0000000000000000","x":1}`,
		"short seed":     `{"code":"a","seed1":"00","seed2":"0000000000000000"}`,
	} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, game.FileName), []byte(data), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := game.Load(dir); err == nil {
				t.Error("Load succeeded")
			}
		})
	}
}

func TestDepositTypeUnmarshal(t *testing.T) {
	for _, tc := range []struct {
		text string
		ok   bool
	}{
		{`"GOLD"`, true}, {`"FUEL"`, true}, {`"MTL"`, true}, {`"NMTL"`, true},
		{`"MTLS"`, false}, {`"gold"`, false}, {`""`, false},
	} {
		t.Run(tc.text, func(t *testing.T) {
			var got game.DepositType
			err := json.Unmarshal([]byte(tc.text), &got)
			if tc.ok && (err != nil || `"`+string(got)+`"` != tc.text) {
				t.Errorf("got %q, %v", got, err)
			} else if !tc.ok && err == nil {
				t.Errorf("accepted %s as %q", tc.text, got)
			}
		})
	}
}

func TestValidateHandle(t *testing.T) {
	for _, tc := range []struct {
		handle string
		ok     bool
	}{
		{"abcdefgh", true},
		{"abcdefghijkl", true},
		{"abcdefg", false},       // 7 letters
		{"abcdefghijklm", false}, // 13 letters
		{"Abcdefgh", false},
		{"abcdefg1", false},
		{"abcd_efgh", false},
		{"abcdéfgh", false},
		{"", false},
	} {
		t.Run(tc.handle, func(t *testing.T) {
			err := game.ValidateHandle(tc.handle)
			if tc.ok != (err == nil) {
				t.Errorf("got error %v, want ok %v", err, tc.ok)
			}
			if err != nil && !errors.Is(err, game.ErrInvalidHandle) {
				t.Errorf("got error %v, want ErrInvalidHandle", err)
			}
		})
	}
}

func TestWriteAndLoadDataUseTheGameTurn(t *testing.T) {
	dir := t.TempDir()
	d := sampleData()
	d.Game.Turn = 3
	if err := d.Write(dir); err != nil {
		t.Fatal(err)
	}
	if got, want := game.TurnDir(dir, 3), filepath.Join(dir, "turns", "3"); got != want {
		t.Errorf("TurnDir = %s, want %s", got, want)
	}
	for _, name := range []string{game.DepositsFileName, game.EmpiresFileName, game.ColoniesFileName, game.OrderResultsFileName} {
		if _, err := os.Stat(filepath.Join(dir, "turns", "3", name)); err != nil {
			t.Errorf("turn file: %v", err)
		}
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is in the game folder: stat = %v", name, err)
		}
	}
	got, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, d) {
		t.Errorf("LoadData =\n%+v\nwant\n%+v", got, d)
	}
}

func TestWriteTurn(t *testing.T) {
	dir := t.TempDir()
	d := sampleData()
	if err := d.Write(dir); err != nil {
		t.Fatal(err)
	}
	turn0 := d

	d.Game.Turn = 1
	d.Colonies = slices.Clone(d.Colonies)
	d.Colonies[0].Population.UEM = 99
	if err := d.WriteTurn(dir); err != nil {
		t.Fatal(err)
	}
	got, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, d) {
		t.Errorf("LoadData =\n%+v\nwant\n%+v", got, d)
	}
	// Turn 0 is still there, unchanged.
	colonies, err := game.LoadColonies(game.TurnDir(dir, 0))
	if err != nil || !reflect.DeepEqual(colonies, turn0.Colonies) {
		t.Errorf("turn 0 colonies = %+v, %v; want %+v", colonies, err, turn0.Colonies)
	}

	// Writing the same turn again fails and changes nothing.
	d.Colonies[0].Population.UEM = 1
	if err := d.WriteTurn(dir); !errors.Is(err, game.ErrTurnExists) {
		t.Fatalf("second WriteTurn = %v, want ErrTurnExists", err)
	}
	if colonies, err := game.LoadColonies(game.TurnDir(dir, 1)); err != nil || colonies[0].Population.UEM != 99 {
		t.Errorf("turn 1 changed: %+v, %v", colonies, err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "turns"))
	if err != nil || len(entries) != 2 {
		t.Errorf("turns folder holds %d entries, %v; want 2", len(entries), err)
	}
}

func TestTurnsAndRollback(t *testing.T) {
	dir := t.TempDir()
	d := sampleData()
	if err := d.Write(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := game.Rollback(dir); !errors.Is(err, game.ErrNoPriorTurn) {
		t.Fatalf("Rollback at turn 0 = %v, want ErrNoPriorTurn", err)
	}

	d.Game.Turn = 1
	if err := d.WriteTurn(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"reports", "orders"} {
		if err := os.Mkdir(filepath.Join(dir, "turns", "1", name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"reports/a.txt", "reports/b.txt", "orders/a.txt"} {
		if err := os.WriteFile(filepath.Join(dir, "turns", "1", filepath.FromSlash(name)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "turns", "notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	g, turns, err := game.Turns(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []game.TurnInfo{{Turn: 0}, {Turn: 1, Current: true, Reports: 2, Orders: 1}}
	if g.Turn != 1 || !reflect.DeepEqual(turns, want) {
		t.Errorf("Turns = turn %d, %+v; want turn 1, %+v", g.Turn, turns, want)
	}

	g, err = game.Rollback(dir)
	if err != nil {
		t.Fatal(err)
	}
	if g.Turn != 0 {
		t.Errorf("Rollback left the game at turn %d, want 0", g.Turn)
	}
	if _, err := os.Stat(filepath.Join(dir, "turns", "1")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("turn 1 folder: stat = %v, want it gone", err)
	}
	got, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := sampleData(); !reflect.DeepEqual(got, want) {
		t.Errorf("after Rollback, LoadData =\n%+v\nwant\n%+v", got, want)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "turns"))
	if len(entries) != 2 { // 0 and notes
		t.Errorf("turns folder holds %d entries, want 2", len(entries))
	}
}

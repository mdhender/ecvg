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
	}
}

var fileNames = []string{
	game.FileName, game.StelliaFileName, game.StarsFileName, game.OrbitsFileName, game.PlanetsFileName,
	game.DepositsFileName, game.RacesFileName, game.EmpiresFileName, game.ColoniesFileName,
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
	const wantJSON = `{"code":"alpha","seed1":"fedcba9876543210","seed2":"0000000000000001"}` + "\n"
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

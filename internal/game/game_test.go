// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

func TestCreateWritesGameJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The top bit is set, so a float64 JSON reader would lose precision.
	const seed1 = 0xfedcba9876543210

	d, err := game.Create(dir, seed1)
	if err != nil {
		t.Fatal(err)
	}
	want := game.Game{Code: "alpha", Seed1: seed1, Seed2: game.Seed(prng.SplitMix64(seed1))}
	if d.Game != want {
		t.Errorf("Create = %+v, want %+v", d.Game, want)
	}

	data, err := os.ReadFile(filepath.Join(dir, game.FileName))
	if err != nil {
		t.Fatal(err)
	}
	wantJSON := `{"code":"alpha","seed1":"fedcba9876543210","seed2":"` +
		string(must(game.Seed(prng.SplitMix64(seed1)).MarshalText())) + "\"}\n"
	if string(data) != wantJSON {
		t.Errorf("game.json =\n%s\nwant\n%s", data, wantJSON)
	}

	loaded, err := game.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded != want {
		t.Errorf("Load = %+v, want %+v", loaded, want)
	}
}

func TestCreateFails(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	existing := filepath.Join(root, "existing")
	if err := os.Mkdir(existing, 0o755); err != nil {
		t.Fatal(err)
	}
	const contents = "keep me"
	if err := os.WriteFile(filepath.Join(existing, game.FileName), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		dir  string
		want error
	}{
		{"missing folder", filepath.Join(root, "missing"), fs.ErrNotExist},
		{"missing parent", filepath.Join(root, "missing", "alpha"), fs.ErrNotExist},
		{"path is a file", file, game.ErrNotDirectory},
		{"game.json exists", existing, game.ErrGameExists},
		{"filesystem root", string(filepath.Separator), game.ErrInvalidCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := game.Create(tc.dir, 1); !errors.Is(err, tc.want) {
				t.Fatalf("Create = %v, want %v", err, tc.want)
			}
		})
	}

	if _, err := os.Stat(filepath.Join(root, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Create made a missing folder: stat = %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(existing, game.FileName)); err != nil || string(data) != contents {
		t.Errorf("existing game.json changed: %q, %v", data, err)
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

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

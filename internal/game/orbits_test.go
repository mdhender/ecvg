// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestGenerateOrbits(t *testing.T) {
	_, stars := game.GenerateStellia(goldenRoot())
	orbits := game.GenerateOrbits(stars)
	if want := len(stars) * game.OrbitsPerStar; len(orbits) != want {
		t.Fatalf("got %d orbits, want %d", len(orbits), want)
	}

	// Orbit IDs count up from 1, so they are unique, and each star's orbits are
	// numbered 1 to OrbitsPerStar in star order.
	for i, o := range orbits {
		want := game.Orbit{ID: i + 1, StarID: stars[i/game.OrbitsPerStar].ID, OrbitNo: i%game.OrbitsPerStar + 1}
		if o != want {
			t.Fatalf("orbit %d = %+v, want %+v", i, o, want)
		}
	}

	for _, tc := range []struct {
		name  string
		stars []game.Star
		want  int
	}{
		{"no stars", nil, 0},
		{"one star", []game.Star{{ID: 7, StelliumID: 3, SeqNo: 1}}, game.OrbitsPerStar},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := game.GenerateOrbits(tc.stars)
			if len(got) != tc.want {
				t.Fatalf("got %d orbits, want %d", len(got), tc.want)
			}
			for _, o := range got {
				if o.StarID != 7 {
					t.Errorf("orbit %+v has star_id %d, want 7", o, o.StarID)
				}
			}
		})
	}
}

func TestCreateWritesOrbitsJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Orbits) != 1380 {
		t.Errorf("got %d orbits, want 1380 (138 stars × 10)", len(d.Orbits))
	}
	if want := game.GenerateOrbits(d.Stars); !reflect.DeepEqual(d.Orbits, want) {
		t.Error("Create returned orbits that differ from GenerateOrbits")
	}
	loaded, err := game.LoadOrbits(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, d.Orbits) {
		t.Error("LoadOrbits differs from what Create wrote")
	}

	data, err := os.ReadFile(filepath.Join(dir, game.OrbitsFileName))
	if err != nil {
		t.Fatal(err)
	}
	const wantPrefix = "[\n  {\n    \"id\": 1,\n    \"star_id\": 1,\n    \"orbit_no\": 1\n  },"
	if len(data) < len(wantPrefix) || string(data[:len(wantPrefix)]) != wantPrefix {
		t.Errorf("orbits.json starts\n%.80s\nwant\n%s", data, wantPrefix)
	}
}

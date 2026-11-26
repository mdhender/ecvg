// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestDefaultHomeStarTemplate(t *testing.T) {
	tmpl := game.DefaultHomeStarTemplate()
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("default template is invalid: %v", err)
	}
	if got := tmpl.Planets(); got != 10 {
		t.Errorf("got %d planets, want 10", got)
	}
	home := tmpl.Orbits[game.HomeWorldOrbitNo-1].Planet
	if home == nil || !home.HomeWorld || !home.Habitable || home.Viability != game.HomeWorldViability {
		t.Errorf("home world is %+v, want marked homeworld, habitable with viability %d", home, game.HomeWorldViability)
	}
}

func TestHomeStarTemplateValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*game.HomeStarTemplate)
		want   []string // substrings of the error; none means valid
	}{
		{name: "default", change: func(*game.HomeStarTemplate) {}},
		{
			name:   "too few orbits",
			change: func(t *game.HomeStarTemplate) { t.Orbits = t.Orbits[:9] },
			want:   []string{"9 orbits: want 10"},
		},
		{
			name:   "orbits out of order",
			change: func(t *game.HomeStarTemplate) { t.Orbits[0].OrbitNo, t.Orbits[1].OrbitNo = 2, 1 },
			want:   []string{"orbit 1: orbit_no 2", "orbit 2: orbit_no 1"},
		},
		{
			name:   "home world missing",
			change: func(t *game.HomeStarTemplate) { t.Orbits[2].Planet = nil },
			want:   []string{"orbit 3: empty"},
		},
		{
			name:   "home world viability",
			change: func(t *game.HomeStarTemplate) { t.Orbits[2].Planet.Viability = 24 },
			want:   []string{"orbit 3: home world must be habitable"},
		},
		{
			name:   "home world not habitable",
			change: func(t *game.HomeStarTemplate) { t.Orbits[2].Planet.Habitable = false },
			want:   []string{"orbit 3: home world must be habitable"},
		},
		{
			name:   "home world not marked",
			change: func(t *game.HomeStarTemplate) { t.Orbits[2].Planet.HomeWorld = false },
			want:   []string{"orbit 3: home world must be marked homeworld"},
		},
		{
			name:   "second home world",
			change: func(t *game.HomeStarTemplate) { t.Orbits[3].Planet.HomeWorld = true },
			want:   []string{"orbit 4: homeworld: only orbit 3"},
		},
		{
			name:   "habitable with viability 0",
			change: func(t *game.HomeStarTemplate) { t.Orbits[3].Planet.Viability = 0 },
			want:   []string{"orbit 4: habitable with viability 0"},
		},
		{
			name: "another habitable planet is fine",
			change: func(t *game.HomeStarTemplate) {
				t.Orbits[0].Planet.Habitable, t.Orbits[0].Planet.Viability = true, 5
			},
		},
		{
			name:   "uninhabitable with viability",
			change: func(t *game.HomeStarTemplate) { t.Orbits[0].Planet.Viability = 3 },
			want:   []string{"orbit 1: not habitable with viability 3"},
		},
		{
			name:   "planet without deposits",
			change: func(t *game.HomeStarTemplate) { t.Orbits[9].Planet.Deposits = nil },
			want:   []string{"orbit 10: 0 deposits"},
		},
		{
			name:   "bad deposit",
			change: func(t *game.HomeStarTemplate) { t.Orbits[5].Planet.Deposits[1].Yield = 23 },
			want:   []string{"orbit 6: deposit 2: FUEL yield 23"},
		},
		{
			name:   "empty orbit is fine",
			change: func(t *game.HomeStarTemplate) { t.Orbits[4].Planet = nil },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := game.DefaultHomeStarTemplate() // a fresh copy each time
			tc.change(&tmpl)
			err := tmpl.Validate()
			if len(tc.want) == 0 {
				if err != nil {
					t.Fatalf("got error %v", err)
				}
				return
			}
			if !errors.Is(err, game.ErrInvalidTemplate) {
				t.Fatalf("got error %v, want ErrInvalidTemplate", err)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error %q lacks %q", err, w)
				}
			}
		})
	}
}

func TestCreateHomeStarTemplate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   func(t *testing.T, dir string)
		wantErr error
	}{
		{
			name:  "writes the default",
			setup: func(t *testing.T, dir string) { createGame(t, dir) },
		},
		{
			name:    "needs a game",
			setup:   func(*testing.T, string) {},
			wantErr: fs.ErrNotExist,
		},
		{
			name: "never overwrites",
			setup: func(t *testing.T, dir string) {
				createGame(t, dir)
				if _, err := game.CreateHomeStarTemplate(dir); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: game.ErrTemplateExists,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "test")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			tc.setup(t, dir)
			got, err := game.CreateHomeStarTemplate(dir)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if want := game.DefaultHomeStarTemplate(); !reflect.DeepEqual(got, want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
			loaded, err := game.LoadHomeStarTemplate(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if !reflect.DeepEqual(loaded, got) {
				t.Errorf("loaded %+v, want %+v", loaded, got)
			}
		})
	}
}

func TestLoadHomeStarTemplateRejectsUnknownMembers(t *testing.T) {
	dir := t.TempDir()
	data := `{"orbits":[], "star_type":"G"}`
	if err := os.WriteFile(filepath.Join(dir, game.HomeStarTemplateFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := game.LoadHomeStarTemplate(dir); err == nil || !strings.Contains(err.Error(), "star_type") {
		t.Errorf("got error %v, want one naming star_type", err)
	}
}

// createGame creates a game in dir, which must exist.
func createGame(t *testing.T, dir string) {
	t.Helper()
	if _, err := game.Create(dir, goldenSeed); err != nil {
		t.Fatal(err)
	}
}

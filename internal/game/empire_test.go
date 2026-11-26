// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

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

// homeData returns the golden game with the first stars of Stellia 12 and 11
// designated as home stars, in that order, and their home world IDs, lowest
// first.
func homeData(t *testing.T) (game.Data, []int) {
	t.Helper()
	d := goldenData(t)
	var worlds []int
	for _, stellium := range []int{12, 11} {
		c, err := d.DesignateHomeStar(starOf(t, d, stellium, 1), game.DefaultHomeStarTemplate())
		if err != nil {
			t.Fatal(err)
		}
		worlds = append(worlds, c.HomeWorldID)
	}
	if worlds[0] > worlds[1] {
		worlds[0], worlds[1] = worlds[1], worlds[0]
	}
	return d, worlds
}

func TestHomeWorlds(t *testing.T) {
	if got := goldenData(t).HomeWorlds(); len(got) != 0 {
		t.Errorf("new game has home worlds %v, want none", got)
	}
	d, worlds := homeData(t)
	if got := d.HomeWorlds(); !reflect.DeepEqual(got, worlds) {
		t.Errorf("got home worlds %v, want %v", got, worlds)
	}
}

// handle returns a valid handle for the i'th player.
func handle(i int) string {
	return fmt.Sprintf("player%c%c", 'a'+i/26, 'a'+i%26)
}

func TestAddEmpireFillsHomeWorldsInOrder(t *testing.T) {
	d, worlds := homeData(t)
	for i := range 2 * game.MaxEmpiresPerHomeWorld {
		e, newRace, err := d.AddEmpire(handle(i), "")
		if err != nil {
			t.Fatalf("empire %d: %v", i+1, err)
		}
		world := worlds[i/game.MaxEmpiresPerHomeWorld]
		want := game.Empire{ID: i + 1, Handle: handle(i), Name: fmt.Sprintf("Empire #%d", i+1), RaceID: world, HomeWorldID: world}
		if e != want {
			t.Fatalf("empire %d: got %+v, want %+v", i+1, e, want)
		}
		if wantNew := i%game.MaxEmpiresPerHomeWorld == 0; newRace != wantNew {
			t.Errorf("empire %d: got newRace %v, want %v", i+1, newRace, wantNew)
		}
	}
	wantRaces := []game.Race{{ID: worlds[0], HomeWorldID: worlds[0]}, {ID: worlds[1], HomeWorldID: worlds[1]}}
	if !reflect.DeepEqual(d.Races, wantRaces) {
		t.Errorf("got races %+v, want %+v", d.Races, wantRaces)
	}

	before := d
	if _, _, err := d.AddEmpire(handle(99), ""); !errors.Is(err, game.ErrNoHomeWorld) {
		t.Fatalf("got error %v, want ErrNoHomeWorld", err)
	}
	if !reflect.DeepEqual(d, before) {
		t.Errorf("data changed on error")
	}
}

func TestAddEmpire(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handle  string
		empName string
		setup   func(t *testing.T, d *game.Data)
		want    func(worlds []int) game.Empire
		wantErr error
	}{
		{
			name:   "default name",
			handle: "abcdefgh",
			want: func(w []int) game.Empire {
				return game.Empire{ID: 1, Handle: "abcdefgh", Name: "Empire #1", RaceID: w[0], HomeWorldID: w[0]}
			},
		},
		{
			name:    "given name",
			handle:  "abcdefgh",
			empName: "The Glorious Horde",
			want: func(w []int) game.Empire {
				return game.Empire{ID: 1, Handle: "abcdefgh", Name: "The Glorious Horde", RaceID: w[0], HomeWorldID: w[0]}
			},
		},
		{
			name:   "ids follow the largest",
			handle: "abcdefgh",
			setup: func(t *testing.T, d *game.Data) {
				d.Empires = []game.Empire{{ID: 7, Handle: "zzzzzzzz", Name: "Empire #7", RaceID: 1, HomeWorldID: 1}}
			},
			want: func(w []int) game.Empire {
				return game.Empire{ID: 8, Handle: "abcdefgh", Name: "Empire #8", RaceID: w[0], HomeWorldID: w[0]}
			},
		},
		{
			name:    "invalid handle",
			handle:  "short",
			wantErr: game.ErrInvalidHandle,
		},
		{
			name:   "duplicate handle",
			handle: "abcdefgh",
			setup: func(t *testing.T, d *game.Data) {
				if _, _, err := d.AddEmpire("abcdefgh", ""); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: game.ErrDuplicateHandle,
		},
		{
			name:    "no home worlds",
			handle:  "abcdefgh",
			setup:   func(t *testing.T, d *game.Data) { *d = goldenData(t) },
			wantErr: game.ErrNoHomeWorld,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, worlds := homeData(t)
			if tc.setup != nil {
				tc.setup(t, &d)
			}
			before := d
			e, _, err := d.AddEmpire(tc.handle, tc.empName)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				if !reflect.DeepEqual(d, before) {
					t.Errorf("data changed on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if want := tc.want(worlds); e != want {
				t.Errorf("got %+v, want %+v", e, want)
			}
		})
	}
}

func TestAddEmpireFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	createGame(t, dir)
	if _, _, err := game.AddEmpire(dir, "abcdefgh", ""); !errors.Is(err, game.ErrNoHomeWorld) {
		t.Fatalf("got error %v, want ErrNoHomeWorld", err)
	}
	for _, name := range []string{game.RacesFileName, game.EmpiresFileName} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			t.Errorf("%s written on error", name)
		}
	}

	if _, err := game.CreateHomeStarTemplate(dir); err != nil {
		t.Fatal(err)
	}
	c, err := game.DesignateHomeStellium(dir, 11)
	if err != nil {
		t.Fatal(err)
	}
	for i, h := range []string{"abcdefgh", "ijklmnop"} {
		e, newRace, err := game.AddEmpire(dir, h, "")
		if err != nil {
			t.Fatalf("empire %d: %v", i+1, err)
		}
		if e.ID != i+1 || e.HomeWorldID != c.HomeWorldID || newRace != (i == 0) {
			t.Errorf("empire %d: got %+v, newRace %v", i+1, e, newRace)
		}
	}
	d, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Empires) != 2 || len(d.Races) != 1 || d.Races[0].ID != c.HomeWorldID {
		t.Errorf("saved races %+v and empires %+v", d.Races, d.Empires)
	}
}

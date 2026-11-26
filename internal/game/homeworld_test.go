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

func TestDefaultHomeWorldTemplate(t *testing.T) {
	tmpl := game.DefaultHomeWorldTemplate()
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("default template is invalid: %v", err)
	}
	if tmpl.OrbitNo != 3 || tmpl.Viability != 20 {
		t.Errorf("got orbit %d viability %d, want 3 and 20", tmpl.OrbitNo, tmpl.Viability)
	}
	counts := map[game.DepositType]int{}
	for _, d := range tmpl.Deposits {
		counts[d.Type]++
	}
	want := map[game.DepositType]int{game.GOLD: 1, game.FUEL: 5, game.MTL: 5, game.NMTL: 5}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("got deposit counts %v, want %v", counts, want)
	}
}

func TestHomeWorldTemplateValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*game.HomeWorldTemplate)
		want   []string // substrings of the error; none means valid
	}{
		{name: "default", change: func(*game.HomeWorldTemplate) {}},
		{name: "orbit 0", change: func(t *game.HomeWorldTemplate) { t.OrbitNo = 0 }, want: []string{"orbit_no 0"}},
		{name: "orbit 11", change: func(t *game.HomeWorldTemplate) { t.OrbitNo = 11 }, want: []string{"orbit_no 11"}},
		{name: "viability 0", change: func(t *game.HomeWorldTemplate) { t.Viability = 0 }, want: []string{"viability 0"}},
		{name: "viability 26", change: func(t *game.HomeWorldTemplate) { t.Viability = 26 }, want: []string{"viability 26"}},
		{name: "no deposits", change: func(t *game.HomeWorldTemplate) { t.Deposits = nil }, want: []string{"0 deposits"}},
		{
			name: "32 deposits",
			change: func(t *game.HomeWorldTemplate) {
				for len(t.Deposits) < 32 {
					t.Deposits = append(t.Deposits, t.Deposits[1])
				}
			},
			want: []string{"32 deposits"},
		},
		{
			name:   "gold too big",
			change: func(t *game.HomeWorldTemplate) { t.Deposits[0].Size = 20_000_001 },
			want:   []string{"deposit 1: GOLD size 20000001"},
		},
		{
			name:   "fuel yield too high",
			change: func(t *game.HomeWorldTemplate) { t.Deposits[1].Yield = 23 },
			want:   []string{"deposit 2: FUEL yield 23"},
		},
		{
			name:   "unknown type",
			change: func(t *game.HomeWorldTemplate) { t.Deposits[2].Type = "IRON" },
			want:   []string{`deposit 3: invalid type "IRON"`},
		},
		{
			name: "reports every problem",
			change: func(t *game.HomeWorldTemplate) {
				t.OrbitNo, t.Viability, t.Deposits[3].Size = 0, 0, 0
			},
			want: []string{"orbit_no 0", "viability 0", "deposit 4: FUEL size 0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := game.DefaultHomeWorldTemplate()
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

func TestCreateHomeWorldTemplate(t *testing.T) {
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
				if _, err := game.CreateHomeWorldTemplate(dir); err != nil {
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
			got, err := game.CreateHomeWorldTemplate(dir)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if want := game.DefaultHomeWorldTemplate(); !reflect.DeepEqual(got, want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
			loaded, err := game.LoadHomeWorldTemplate(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if !reflect.DeepEqual(loaded, got) {
				t.Errorf("loaded %+v, want %+v", loaded, got)
			}
		})
	}
}

func TestLoadHomeWorldTemplate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		data    string
		wantErr bool
		wantIs  error // if not nil, the error must match it
	}{
		{name: "rejects invalid values", data: `{"orbit_no":0,"viability":20,"deposits":[{"type":"FUEL","size":5000000,"yield":5}]}`, wantErr: true, wantIs: game.ErrInvalidTemplate},
		{name: "rejects unknown members", data: `{"orbit_no":3,"viability":20,"habitable":true,"deposits":[{"type":"FUEL","size":5000000,"yield":5}]}`, wantErr: true},
		{name: "accepts a valid template", data: `{"orbit_no":4,"viability":25,"deposits":[{"type":"FUEL","size":5000000,"yield":5}]}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, game.HomeWorldTemplateFileName), []byte(tc.data), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := game.LoadHomeWorldTemplate(dir)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got error %v, want error %v", err, tc.wantErr)
			}
			if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
				t.Errorf("got error %v, want %v", err, tc.wantIs)
			}
		})
	}
}

// createGame creates a game in dir, which must exist.
func createGame(t *testing.T, dir string) {
	t.Helper()
	if _, err := game.Create(dir, goldenSeed); err != nil {
		t.Fatal(err)
	}
}

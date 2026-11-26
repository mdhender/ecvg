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

func TestDefaultEmpireTemplate(t *testing.T) {
	tmpl := game.DefaultEmpireTemplate()
	if err := tmpl.Validate(); err != nil {
		t.Fatalf("default template is invalid: %v", err)
	}
	if err := tmpl.CheckHomeWorld(game.DefaultHomeStarTemplate()); err != nil {
		t.Errorf("default template doesn't fit the default home world: %v", err)
	}
	if got := len(tmpl.FarmingGroups); got != 1 {
		t.Errorf("got %d farming groups, want 1", got)
	}
	if got := len(tmpl.MiningGroups); got != 1 {
		t.Errorf("got %d mining groups, want 1", got)
	}
	if got := len(tmpl.FactoryGroups); got != 4 {
		t.Errorf("got %d factory groups, want 4", got)
	}
	var cache []string
	for _, c := range tmpl.Cache {
		cache = append(cache, string(c.Type))
	}
	if want := []string{"FOOD", "FUEL", "MTL", "NMTL"}; !reflect.DeepEqual(cache, want) {
		t.Errorf("got cache %v, want %v", cache, want)
	}
}

func TestEmpireTemplateValidate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*game.EmpireTemplate)
		want   []string // substrings of the error; none means valid
	}{
		{name: "default", change: func(*game.EmpireTemplate) {}},
		{
			name:   "unknown population type",
			change: func(t *game.EmpireTemplate) { t.Population[0].Type = "ROB" },
			want:   []string{`population 1: invalid type "ROB"`},
		},
		{
			name:   "population type twice",
			change: func(t *game.EmpireTemplate) { t.Population[1].Type = t.Population[0].Type },
			want:   []string{"population 2: UEM listed twice"},
		},
		{
			name:   "no people",
			change: func(t *game.EmpireTemplate) { t.Population[2].Qty = 0 },
			want:   []string{"population 3: PRO qty 0"},
		},
		{
			name:   "two farming groups",
			change: func(t *game.EmpireTemplate) { t.FarmingGroups = append(t.FarmingGroups, t.FarmingGroups[0]) },
			want:   []string{"2 farming groups: want 1"},
		},
		{
			name:   "no mining groups",
			change: func(t *game.EmpireTemplate) { t.MiningGroups = nil },
			want:   []string{"0 mining groups: want 1"},
		},
		{
			name:   "three factory groups",
			change: func(t *game.EmpireTemplate) { t.FactoryGroups = t.FactoryGroups[:3] },
			want:   []string{"3 factory groups: want 4"},
		},
		{
			name:   "empty group",
			change: func(t *game.EmpireTemplate) { t.FarmingGroups[0].Units = 0 },
			want:   []string{"farming group 1: units 0"},
		},
		{
			name:   "tech level 0",
			change: func(t *game.EmpireTemplate) { t.FactoryGroups[2].TechLevel = 0 },
			want:   []string{"factory group 3: tech_level 0"},
		},
		{
			name:   "deposit 0",
			change: func(t *game.EmpireTemplate) { t.MiningGroups[0].DepositNo = 0 },
			want:   []string{"mining group 1: deposit_no 0"},
		},
		{
			name:   "deposit 32",
			change: func(t *game.EmpireTemplate) { t.MiningGroups[0].DepositNo = 32 },
			want:   []string{"mining group 1: deposit_no 32"},
		},
		{
			name:   "bad orders",
			change: func(t *game.EmpireTemplate) { t.FactoryGroups[0].Orders = "consumer goods" },
			want:   []string{`factory group 1: orders "consumer goods"`},
		},
		{
			name:   "orders with a tech level",
			change: func(t *game.EmpireTemplate) { t.FactoryGroups[0].Orders = "SDRV-3" },
		},
		{
			name:   "unknown cache type",
			change: func(t *game.EmpireTemplate) { t.Cache[0].Type = "GOLD" },
			want:   []string{`cache 1: invalid type "GOLD"`},
		},
		{
			name:   "cache type twice",
			change: func(t *game.EmpireTemplate) { t.Cache[3].Type = "FOOD" },
			want:   []string{"cache 4: FOOD listed twice"},
		},
		{
			name: "reports every problem",
			change: func(t *game.EmpireTemplate) {
				t.Population[0].Qty, t.MiningGroups[0].Units, t.Cache[1].Qty = -1, 0, 0
			},
			want: []string{"population 1: UEM qty -1", "mining group 1: units 0", "cache 2: FUEL qty 0"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := game.DefaultEmpireTemplate() // a fresh copy each time
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

func TestEmpireTemplateCheckHomeWorld(t *testing.T) {
	tmpl := game.DefaultEmpireTemplate()
	home := smallTemplate() // a home world with 4 deposits
	for _, tc := range []struct {
		depositNo int
		ok        bool
	}{{1, true}, {4, true}, {5, false}} {
		tmpl.MiningGroups[0].DepositNo = tc.depositNo
		err := tmpl.CheckHomeWorld(home)
		if tc.ok != (err == nil) {
			t.Errorf("deposit %d: got error %v, want ok %v", tc.depositNo, err, tc.ok)
		}
		if err != nil && !errors.Is(err, game.ErrInvalidTemplate) {
			t.Errorf("deposit %d: got error %v, want ErrInvalidTemplate", tc.depositNo, err)
		}
	}
}

func TestCreateEmpireTemplate(t *testing.T) {
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
				if _, err := game.CreateEmpireTemplate(dir); err != nil {
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
			got, err := game.CreateEmpireTemplate(dir)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v", err)
			}
			if want := game.DefaultEmpireTemplate(); !reflect.DeepEqual(got, want) {
				t.Errorf("got %+v, want %+v", got, want)
			}
			loaded, err := game.LoadEmpireTemplate(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if !reflect.DeepEqual(loaded, got) {
				t.Errorf("loaded %+v, want %+v", loaded, got)
			}
		})
	}
}

func TestLoadEmpireTemplateRejectsUnknownMembers(t *testing.T) {
	dir := t.TempDir()
	data := `{"population":[],"farming_groups":[],"mining_groups":[],"factory_groups":[],"cache":[],"ships":[]}`
	if err := os.WriteFile(filepath.Join(dir, game.EmpireTemplateFileName), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := game.LoadEmpireTemplate(dir); err == nil || !strings.Contains(err.Error(), "ships") {
		t.Errorf("got error %v, want one naming ships", err)
	}
}

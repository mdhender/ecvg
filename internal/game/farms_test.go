// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

func TestCanInstallFarm(t *testing.T) {
	home := game.Planet{Habitable: true, Viability: 25}
	rock := game.Planet{}
	for _, tc := range []struct {
		tl     int
		t      game.EntityType
		orbit  int
		planet game.Planet
		ok     bool
	}{
		// FARM-1: open air, habitable, viability above 0, orbits 1..5.
		{1, game.COPN, 3, home, true},
		{1, game.COPN, 1, home, true},
		{1, game.COPN, 5, home, true},
		{1, game.COPN, 6, home, false},
		{1, game.COPN, 3, rock, false},
		{1, game.COPN, 3, game.Planet{Habitable: true}, false},
		{1, game.CENC, 3, home, false},
		{1, game.CORB, 3, home, false},
		{1, game.CSHP, 3, home, false},
		// FARM-2..5: orbital colonies anywhere, surface colonies in 1..5.
		{2, game.CORB, 10, rock, true},
		{5, game.CORB, 1, rock, true},
		{2, game.COPN, 5, home, true},
		{5, game.CENC, 5, rock, true},
		{5, game.CENC, 6, rock, false},
		{2, game.COPN, 6, home, false},
		{2, game.CSHP, 3, home, false},
		{5, game.CSHP, 10, rock, false},
		// FARM-6..10: anything, anywhere.
		{6, game.CSHP, 10, rock, true},
		{6, game.CSHP, 0, rock, true},
		{10, game.COPN, 3, home, true},
		{10, game.CENC, 9, rock, true},
		{10, game.CORB, 7, rock, true},
		// Not farms.
		{0, game.COPN, 3, home, false},
		{11, game.COPN, 3, home, false},
	} {
		t.Run(fmt.Sprintf("FARM-%d %s orbit %d viability %d", tc.tl, tc.t, tc.orbit, tc.planet.Viability), func(t *testing.T) {
			err := game.CanInstallFarm(tc.tl, tc.t, tc.orbit, tc.planet)
			if tc.ok != (err == nil) {
				t.Fatalf("err = %v, want ok %v", err, tc.ok)
			}
			if err != nil && !errors.Is(err, game.ErrFarmSite) {
				t.Errorf("err = %v, want ErrFarmSite", err)
			}
		})
	}
}

func TestFarmCap(t *testing.T) {
	for _, tc := range []struct {
		p    game.Planet
		want int
	}{
		{game.Planet{}, 0},
		{game.Planet{Habitable: true, Viability: 1}, 100_000},
		{game.Planet{Habitable: true, Viability: 25}, 2_500_000},
	} {
		if got := tc.p.FarmCap(); got != tc.want {
			t.Errorf("%+v: FarmCap = %d, want %d", tc.p, got, tc.want)
		}
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"math"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

func TestLifeSupportRules(t *testing.T) {
	if got := game.LifeSupported(5, 1); got != 25 {
		t.Errorf("LifeSupported(5, 1) = %d, want 25 (manual G6.1)", got)
	}
	if got := game.LifeSupportFuel(5, 3); got != 15 {
		t.Errorf("LifeSupportFuel(5, 3) = %d, want 15", got)
	}
	if got := game.LifeSupportFuelled(5, 14); got != 2 {
		t.Errorf("LifeSupportFuelled(5, 14) = %d, want 2", got)
	}
	for _, tc := range []struct {
		t     game.EntityType
		orbit int
		solar bool
	}{{game.CORB, 1, true}, {game.CORB, 5, true}, {game.CORB, 6, false}, {game.CENC, 3, false}, {game.CSHP, 3, false}} {
		if got := game.LifeSupportSolar(tc.t, tc.orbit); got != tc.solar {
			t.Errorf("LifeSupportSolar(%s, %d) = %v, want %v", tc.t, tc.orbit, got, tc.solar)
		}
	}
	for _, tc := range []struct {
		units int
		rate  float64
		want  int
	}{
		{1000, game.DefaultDeathRate, 0},
		{1000, 0, 0},
		{1000, 11 * game.DefaultDeathRate, 63}, // 62.5, rounded up
		{1600, 11 * game.DefaultDeathRate, 100},
		{1, 0.5, 1},
		{1000, 1, 1000},
		{0, 1, 0},
	} {
		if got := game.ExcessDeaths(tc.units, tc.rate); got != tc.want {
			t.Errorf("ExcessDeaths(%d, %v) = %d, want %d", tc.units, tc.rate, got, tc.want)
		}
	}
}

func TestLifeSupport(t *testing.T) {
	const normal = game.DefaultDeathRate
	lfsu := func(tl, qty int) game.Unit { return game.Unit{Type: "LFSU", TechLevel: tl, Qty: qty} }
	fuel := func(qty int) game.Unit { return game.Unit{Type: "FUEL", Qty: qty} }
	for _, tc := range []struct {
		name   string
		turn   int
		colony game.Colony
		// What is wanted afterwards.
		life game.LifeSupport
		rate float64
		pop  game.Population
		fuel int
	}{
		{
			name: "an open-air colony needs none, and its LFSU don't run",
			turn: 1,
			colony: game.Colony{Type: game.COPN, DeathRate: normal, Population: game.Population{UEM: 500},
				Operational: []game.Unit{lfsu(10, 1)}, Cargo: []game.Unit{fuel(100)}},
			rate: normal, pop: game.Population{UEM: 500}, fuel: 100,
		},
		{
			name: "every assembled LFSU runs and burns FUEL, needed or not",
			turn: 1,
			colony: game.Colony{Type: game.CSHP, DeathRate: normal, Population: game.Population{PRO: 100},
				Operational: []game.Unit{lfsu(5, 4), lfsu(1, 3)}, Cargo: []game.Unit{fuel(100)}},
			life: game.LifeSupport{Units: 7, Worked: 7, Fuel: 23, Needed: 100, Supported: 103},
			rate: normal, pop: game.Population{PRO: 100}, fuel: 77,
		},
		{
			name: "LFSU in storage support nobody",
			turn: 1,
			colony: game.Colony{Type: game.CENC, DeathRate: normal, Population: game.Population{PRO: 100},
				Cargo: []game.Unit{lfsu(10, 5), fuel(100)}},
			// Everybody is without: the rate is multiplied by 11.
			life: game.LifeSupport{Needed: 100},
			rate: 11 * normal, pop: game.Population{PRO: 100}, fuel: 100,
		},
		{
			name: "without FUEL the LFSU don't run, in the order listed",
			turn: 1,
			colony: game.Colony{Type: game.CSHP, DeathRate: normal, Population: game.Population{USK: 100},
				Operational: []game.Unit{lfsu(5, 4), lfsu(10, 1)}, Cargo: []game.Unit{fuel(12)}},
			// 2 LFSU-5 run on 10 FUEL and support 50 of the 100: P is 0.5.
			life: game.LifeSupport{Units: 5, Worked: 2, Unfuelled: 3, Fuel: 10, Needed: 100, Supported: 50},
			rate: 6 * normal, pop: game.Population{USK: 100}, fuel: 2,
		},
		{
			name: "an orbital colony in orbit 3 is solar powered",
			turn: 1,
			colony: game.Colony{Type: game.CORB, PlanetID: 1, DeathRate: normal, Population: game.Population{USK: 100},
				Operational: []game.Unit{lfsu(10, 1)}},
			life: game.LifeSupport{Units: 1, Worked: 1, Needed: 100, Supported: 100},
			rate: normal, pop: game.Population{USK: 100},
		},
		{
			name: "an orbital colony in orbit 6 is not",
			turn: 1,
			colony: game.Colony{Type: game.CORB, PlanetID: 2, DeathRate: normal, Population: game.Population{USK: 100},
				Operational: []game.Unit{lfsu(10, 1)}},
			life: game.LifeSupport{Units: 1, Unfuelled: 1, Needed: 100},
			rate: 11 * normal, pop: game.Population{USK: 100},
		},
		{
			name: "a raised death rate kills, and the shortage raises it again",
			turn: 2,
			colony: game.Colony{Type: game.CSHP, DeathRate: 11 * normal, Population: game.Population{UEM: 400, USK: 400, PRO: 800},
				Cadres: game.Cadres{CWRK: 400}},
			// 6.25% of 1,600 is 100: 25, 25, and 50. The 1,500 left have none.
			life: game.LifeSupport{Needed: 1500, Died: game.Population{UEM: 25, USK: 25, PRO: 50}},
			rate: 121 * normal, pop: game.Population{UEM: 375, USK: 375, PRO: 750},
		},
		{
			name:   "the death rate never passes 100%, and then everybody dies",
			turn:   3,
			colony: game.Colony{Type: game.CSHP, DeathRate: 0.5, Population: game.Population{UEM: 3, SLD: 2}},
			// Half of 5, less the normal rate, rounded up: 3 die.
			life: game.LifeSupport{Needed: 2, Died: game.Population{UEM: 2, SLD: 1}},
			rate: 1, pop: game.Population{UEM: 1, SLD: 1},
		},
		{
			name: "with life support again the rate is halved",
			turn: 2,
			colony: game.Colony{Type: game.CSHP, DeathRate: 4 * normal, Population: game.Population{USK: 100},
				Operational: []game.Unit{lfsu(10, 1)}, Cargo: []game.Unit{fuel(10)}},
			// 1.875% of 100, rounded up, die first.
			life: game.LifeSupport{Units: 1, Worked: 1, Fuel: 10, Needed: 98, Supported: 100, Died: game.Population{USK: 2}},
			rate: 2 * normal, pop: game.Population{USK: 98},
		},
		{
			name:   "but never below normal",
			turn:   2,
			colony: game.Colony{Type: game.COPN, DeathRate: 1.5 * normal, Population: game.Population{USK: 100}},
			life:   game.LifeSupport{Died: game.Population{USK: 1}},
			rate:   normal, pop: game.Population{USK: 99},
		},
		{
			name: "nothing happens in the setup turn",
			turn: 0,
			colony: game.Colony{Type: game.CSHP, DeathRate: 0.5, Population: game.Population{USK: 100},
				LifeSupport: game.LifeSupport{Units: 9}},
			rate: 0.5, pop: game.Population{USK: 100},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := game.Data{
				Orbits:   []game.Orbit{{ID: 1, OrbitNo: 3}, {ID: 2, OrbitNo: 6}},
				Planets:  []game.Planet{{ID: 1, OrbitID: 1}, {ID: 2, OrbitID: 2}},
				Colonies: []game.Colony{tc.colony},
			}
			if d.Colonies[0].PlanetID == 0 {
				d.Colonies[0].PlanetID = 1
				if tc.colony.Type == game.CORB {
					t.Fatal("an orbital colony must say which planet it is at")
				}
			}
			engine.LifeSupport(&d, tc.turn)
			c := d.Colonies[0]
			if c.LifeSupport != tc.life {
				t.Errorf("life support %+v, want %+v", c.LifeSupport, tc.life)
			}
			if math.Abs(c.DeathRate-tc.rate) > 1e-12 {
				t.Errorf("death rate %v, want %v", c.DeathRate, tc.rate)
			}
			if c.Population != tc.pop {
				t.Errorf("population %+v, want %+v", c.Population, tc.pop)
			}
			if got := c.Stock("FUEL"); got != tc.fuel {
				t.Errorf("FUEL %d, want %d", got, tc.fuel)
			}
			if c.Cadres != c.Cadres.Fit(c.Population) {
				t.Errorf("cadres %+v don't fit the population %+v", c.Cadres, c.Population)
			}
		})
	}
}

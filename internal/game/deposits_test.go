// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"encoding/json/v2"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

func goldenDeposits(root *prng.Stream) ([]game.Planet, []game.Deposit) {
	_, _, planets := goldenPlanets(root)
	return planets, game.GenerateDeposits(root, planets)
}

func TestGenerateDepositsRules(t *testing.T) {
	planets, deposits := goldenDeposits(goldenRoot())
	if len(deposits) != game.NumDeposits {
		t.Fatalf("got %d deposits, want %d", len(deposits), game.NumDeposits)
	}

	isPlanet := map[int]bool{}
	for _, p := range planets {
		isPlanet[p.ID] = true
	}
	perPlanet := map[int]int{}
	types := map[game.DepositType]int{}
	for i, d := range deposits {
		// Deposit IDs count up from 1, so they are unique.
		if d.ID != i+1 {
			t.Errorf("deposit %d has id %d, want %d", i, d.ID, i+1)
		}
		if !isPlanet[d.PlanetID] {
			t.Fatalf("deposit %d has planet_id %d, which is not a planet", d.ID, d.PlanetID)
		}
		if i > 0 && d.PlanetID < deposits[i-1].PlanetID {
			t.Errorf("deposit %d is out of planet order", d.ID)
		}
		perPlanet[d.PlanetID]++
		if d.DepositNo != perPlanet[d.PlanetID] {
			t.Errorf("deposit %d has deposit_no %d, want %d", d.ID, d.DepositNo, perPlanet[d.PlanetID])
		}
		types[d.Type]++
	}

	if want := map[game.DepositType]int{game.FUEL: 3378, game.MTL: 3180, game.NMTL: 3180, game.GOLD: 198}; !reflect.DeepEqual(types, want) {
		t.Errorf("deposits by type = %v, want %v", types, want)
	}

	// Planets by number of deposits follow 3d11 + 1 scaled to the planets,
	// each count within rounding of the exact value and symmetric about 16.
	var hist [32]int
	for _, p := range planets {
		n := perPlanet[p.ID]
		if n < 1 || n > 31 {
			t.Fatalf("planet %d has %d deposits, want 1..31", p.ID, n)
		}
		hist[n]++
	}
	var ways [32]int // ways[n] is the number of 3d11 rolls giving n-1
	for a := range 11 {
		for b := range 11 {
			for c := range 11 {
				ways[a+b+c+1]++
			}
		}
	}
	for n := 1; n <= 31; n++ {
		exact := float64(len(planets)) * float64(ways[n]) / (11 * 11 * 11)
		if math.Abs(float64(hist[n])-exact) >= 1 {
			t.Errorf("%d planets have %d deposits, want about %.2f", hist[n], n, exact)
		}
		if hist[n] != hist[32-n] {
			t.Errorf("%d planets have %d deposits but %d have %d", hist[n], n, hist[32-n], 32-n)
		}
	}
	if hist[1] != 0 || hist[31] != 0 {
		t.Errorf("%d planets have 1 deposit and %d have 31, want none", hist[1], hist[31])
	}
}

func TestGenerateDepositsMatchesGolden(t *testing.T) {
	_, deposits := goldenDeposits(goldenRoot())
	for i, want := range map[int]game.Deposit{
		0:    {ID: 1, PlanetID: 1, DepositNo: 1, Type: game.FUEL},
		1:    {ID: 2, PlanetID: 1, DepositNo: 2, Type: game.NMTL},
		9935: {ID: 9936, PlanetID: 621, DepositNo: 21, Type: game.MTL},
	} {
		if deposits[i] != want {
			t.Errorf("deposit %d = %+v, want %+v", i, deposits[i], want)
		}
	}
}

func TestGenerateDepositsDependsOnSeeds(t *testing.T) {
	_, a := goldenDeposits(goldenRoot())
	root := goldenRoot()
	root.Uint64() // drawing from the root does not move the deposits
	if _, b := goldenDeposits(root); !reflect.DeepEqual(a, b) {
		t.Error("same seeds gave different deposits")
	}
	if _, c := goldenDeposits(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same deposits")
	}
}

func TestGenerateDepositsPanicsOnWrongPlanetCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("did not panic")
		}
	}()
	game.GenerateDeposits(goldenRoot(), []game.Planet{{ID: 1, OrbitID: 1}})
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

func TestCreateWritesDepositsJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, want := goldenDeposits(goldenRoot()); !reflect.DeepEqual(d.Deposits, want) {
		t.Error("Create returned deposits that differ from GenerateDeposits")
	}
	loaded, err := game.LoadDeposits(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, d.Deposits) {
		t.Error("LoadDeposits differs from what Create wrote")
	}
}

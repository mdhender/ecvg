// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

const goldenSeed = 0x0123456789abcdef

func goldenRoot() *prng.Stream {
	return prng.New(goldenSeed, prng.SplitMix64(goldenSeed))
}

func TestGenerateStelliaRules(t *testing.T) {
	stellia := game.GenerateStellia(goldenRoot())
	if len(stellia) != game.NumStellia {
		t.Fatalf("got %d stellia, want %d", len(stellia), game.NumStellia)
	}

	seen := map[[3]int]bool{}
	counts := map[int]int{}
	for i, s := range stellia {
		p := [3]int{s.X, s.Y, s.Z}
		if p == [3]int{} {
			t.Errorf("stellium %d is at the origin", i)
		}
		if d2 := s.X*s.X + s.Y*s.Y + s.Z*s.Z; d2 > game.StelliaRadius*game.StelliaRadius {
			t.Errorf("stellium %d at %v is beyond radius %d", i, p, game.StelliaRadius)
		}
		if seen[p] {
			t.Errorf("stellium %d at %v duplicates another", i, p)
		}
		seen[p] = true

		for j, star := range s.Stars {
			if star.SeqNo != j+1 {
				t.Errorf("stellium %d star %d has seq_no %d, want %d", i, j, star.SeqNo, j+1)
			}
		}
		counts[len(s.Stars)]++
	}

	// 1 with 5 stars, 1 with 4, 8 with 3, 15 with 2, and the remaining 75 with 1,
	// in that order.
	if want := map[int]int{5: 1, 4: 1, 3: 8, 2: 15, 1: 75}; !reflect.DeepEqual(counts, want) {
		t.Errorf("stellia by star count = %v, want %v", counts, want)
	}
	for i := 1; i < len(stellia); i++ {
		if len(stellia[i].Stars) > len(stellia[i-1].Stars) {
			t.Fatalf("stellium %d has more stars than stellium %d", i, i-1)
		}
	}
}

func TestGenerateStelliaMatchesGolden(t *testing.T) {
	// Also guards against a Go release changing math/rand/v2's Shuffle.
	stellia := game.GenerateStellia(goldenRoot())
	for i, want := range map[int][4]int{
		0:  {-7, 4, 5, 5},
		1:  {6, 5, -6, 4},
		2:  {-4, -4, -5, 3},
		3:  {1, 8, 9, 3},
		99: {-4, 2, -2, 1},
	} {
		s := stellia[i]
		if got := [4]int{s.X, s.Y, s.Z, len(s.Stars)}; got != want {
			t.Errorf("stellium %d = %v, want %v", i, got, want)
		}
	}
}

func TestGenerateStelliaDependsOnSeeds(t *testing.T) {
	a := game.GenerateStellia(goldenRoot())
	root := goldenRoot()
	root.Uint64() // drawing from the root does not move the stellia
	if b := game.GenerateStellia(root); !reflect.DeepEqual(a, b) {
		t.Error("same seeds gave different stellia")
	}
	if c := game.GenerateStellia(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same stellia")
	}
}

func TestCreateWritesStelliaJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, created, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	if want := game.GenerateStellia(goldenRoot()); !reflect.DeepEqual(created, want) {
		t.Error("Create returned stellia that differ from GenerateStellia")
	}
	loaded, err := game.LoadStellia(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, created) {
		t.Error("LoadStellia differs from what Create wrote")
	}
}

func TestCreateWritesBothFilesOrNeither(t *testing.T) {
	for _, tc := range []struct {
		existing, absent string
	}{
		{game.FileName, game.StelliaFileName},
		{game.StelliaFileName, game.FileName},
	} {
		t.Run(tc.existing+" exists", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "alpha")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			const contents = "keep me"
			if err := os.WriteFile(filepath.Join(dir, tc.existing), []byte(contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, _, err := game.Create(dir, 1); !errors.Is(err, game.ErrGameExists) {
				t.Fatalf("Create = %v, want ErrGameExists", err)
			}
			if data, err := os.ReadFile(filepath.Join(dir, tc.existing)); err != nil || string(data) != contents {
				t.Errorf("%s changed: %q, %v", tc.existing, data, err)
			}
			if _, err := os.Stat(filepath.Join(dir, tc.absent)); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s was left behind: stat = %v", tc.absent, err)
			}
		})
	}
}

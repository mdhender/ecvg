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
	stellia, stars := game.GenerateStellia(goldenRoot())
	if len(stellia) != game.NumStellia {
		t.Fatalf("got %d stellia, want %d", len(stellia), game.NumStellia)
	}

	seen := map[[3]int]bool{}
	for i, s := range stellia {
		if s.ID != i+1 {
			t.Errorf("stellium %d has id %d, want %d", i, s.ID, i+1)
		}
		p := [3]int{s.X, s.Y, s.Z}
		if p == [3]int{} {
			t.Errorf("stellium %d is at the origin", s.ID)
		}
		if d2 := s.X*s.X + s.Y*s.Y + s.Z*s.Z; d2 > game.StelliaRadius*game.StelliaRadius {
			t.Errorf("stellium %d at %v is beyond radius %d", s.ID, p, game.StelliaRadius)
		}
		if seen[p] {
			t.Errorf("stellium %d at %v duplicates another", s.ID, p)
		}
		seen[p] = true
	}

	if len(stars) != 138 {
		t.Errorf("got %d stars, want 138", len(stars))
	}
	starsIn := make([]int, len(stellia)+1) // indexed by stellium ID
	for i, star := range stars {
		// Star IDs are unique because they count up.
		if star.ID != i+1 {
			t.Errorf("star %d has id %d, want %d", i, star.ID, i+1)
		}
		if star.StelliumID < 1 || star.StelliumID > len(stellia) {
			t.Fatalf("star %d has stellium_id %d, which is not a stellium", star.ID, star.StelliumID)
		}
		if i > 0 && star.StelliumID < stars[i-1].StelliumID {
			t.Errorf("star %d is out of stellium order", star.ID)
		}
		starsIn[star.StelliumID]++
		if star.SeqNo != starsIn[star.StelliumID] {
			t.Errorf("star %d has seq_no %d, want %d", star.ID, star.SeqNo, starsIn[star.StelliumID])
		}
	}

	// 1 with 5 stars, 1 with 4, 8 with 3, 15 with 2, and the remaining 75 with 1,
	// in that order.
	for id := 1; id <= len(stellia); id++ {
		want := 1
		switch {
		case id == 1:
			want = 5
		case id == 2:
			want = 4
		case id <= 10:
			want = 3
		case id <= 25:
			want = 2
		}
		if starsIn[id] != want {
			t.Errorf("stellium %d has %d stars, want %d", id, starsIn[id], want)
		}
	}
}

func TestGenerateStelliaMatchesGolden(t *testing.T) {
	stellia, stars := game.GenerateStellia(goldenRoot())
	for i, want := range map[int][4]int{
		0:  {1, 0, -4, -9},
		1:  {2, -7, 4, 5},
		2:  {3, -4, -4, -5},
		3:  {4, 1, 8, 9},
		99: {100, -4, 2, -2},
	} {
		s := stellia[i]
		if got := [4]int{s.ID, s.X, s.Y, s.Z}; got != want {
			t.Errorf("stellium %d = %v, want %v", i, got, want)
		}
	}
	for i, want := range map[int]game.Star{
		0:   {ID: 1, StelliumID: 1, SeqNo: 1},
		4:   {ID: 5, StelliumID: 1, SeqNo: 5},
		5:   {ID: 6, StelliumID: 2, SeqNo: 1},
		137: {ID: 138, StelliumID: 100, SeqNo: 1},
	} {
		if stars[i] != want {
			t.Errorf("star %d = %+v, want %+v", i, stars[i], want)
		}
	}
}

func TestGenerateStelliaDependsOnSeeds(t *testing.T) {
	a, _ := game.GenerateStellia(goldenRoot())
	root := goldenRoot()
	root.Uint64() // drawing from the root does not move the stellia
	if b, _ := game.GenerateStellia(root); !reflect.DeepEqual(a, b) {
		t.Error("same seeds gave different stellia")
	}
	if c, _ := game.GenerateStellia(prng.New(goldenSeed+1, prng.SplitMix64(goldenSeed+1))); reflect.DeepEqual(a, c) {
		t.Error("different seeds gave the same stellia")
	}
}

func TestCreateWritesStelliaAndStarsJSON(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	stellia, stars := d.Stellia, d.Stars
	wantStellia, wantStars := game.GenerateStellia(goldenRoot())
	if !reflect.DeepEqual(stellia, wantStellia) || !reflect.DeepEqual(stars, wantStars) {
		t.Error("Create returned stellia or stars that differ from GenerateStellia")
	}

	loadedStellia, err := game.LoadStellia(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadedStellia, stellia) {
		t.Error("LoadStellia differs from what Create wrote")
	}
	loadedStars, err := game.LoadStars(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loadedStars, stars) {
		t.Error("LoadStars differs from what Create wrote")
	}

	data, err := os.ReadFile(filepath.Join(dir, game.StarsFileName))
	if err != nil {
		t.Fatal(err)
	}
	const wantPrefix = "[\n  {\n    \"id\": 1,\n    \"stellium_id\": 1,\n    \"seq_no\": 1\n  },"
	if len(data) < len(wantPrefix) || string(data[:len(wantPrefix)]) != wantPrefix {
		t.Errorf("stars.json starts\n%.80s\nwant\n%s", data, wantPrefix)
	}
}

func TestCreateWritesAllFilesOrNone(t *testing.T) {
	files := []string{game.FileName, game.StelliaFileName, game.StarsFileName, game.OrbitsFileName}
	for _, existing := range files {
		t.Run(existing+" exists", func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "alpha")
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			const contents = "keep me"
			if err := os.WriteFile(filepath.Join(dir, existing), []byte(contents), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := game.Create(dir, 1); !errors.Is(err, game.ErrGameExists) {
				t.Fatalf("Create = %v, want ErrGameExists", err)
			}
			if data, err := os.ReadFile(filepath.Join(dir, existing)); err != nil || string(data) != contents {
				t.Errorf("%s changed: %q, %v", existing, data, err)
			}
			for _, name := range files {
				if name == existing {
					continue
				}
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("%s was left behind: stat = %v", name, err)
				}
			}
		})
	}
}

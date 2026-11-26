// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game_test

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

// goldenData returns the data Create makes from goldenSeed, without writing it.
func goldenData(t *testing.T) game.Data {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "golden")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	d, err := game.Create(dir, goldenSeed)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// firstStarOf returns the ID of the star with seqNo in the Stellium with
// stelliumID.
func starOf(t *testing.T, d game.Data, stelliumID, seqNo int) int {
	t.Helper()
	for _, s := range d.Stars {
		if s.StelliumID == stelliumID && s.SeqNo == seqNo {
			return s.ID
		}
	}
	t.Fatalf("stellium %d has no star %d", stelliumID, seqNo)
	return 0
}

func TestDesignateHomeStar(t *testing.T) {
	d := goldenData(t)
	before := d
	tmpl := game.DefaultHomeStarTemplate()
	const stelliumID = 11 // the first two-star Stellium
	starID := starOf(t, d, stelliumID, 1)

	c, err := d.DesignateHomeStar(starID, tmpl)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if c.StarID != starID || c.StelliumID != stelliumID {
		t.Errorf("got star %d stellium %d, want %d and %d", c.StarID, c.StelliumID, starID, stelliumID)
	}
	if c.Removed.Orbits != game.OrbitsPerStar || c.Added.Orbits != game.OrbitsPerStar {
		t.Errorf("got %d orbits removed and %d added, want %d each", c.Removed.Orbits, c.Added.Orbits, game.OrbitsPerStar)
	}
	if c.Added.Planets != tmpl.Planets() {
		t.Errorf("got %d planets added, want %d", c.Added.Planets, tmpl.Planets())
	}
	if got, want := len(d.Planets), len(before.Planets)-c.Removed.Planets+c.Added.Planets; got != want {
		t.Errorf("got %d planets, want %d", got, want)
	}
	if got, want := len(d.Deposits), len(before.Deposits)-c.Removed.Deposits+c.Added.Deposits; got != want {
		t.Errorf("got %d deposits, want %d", got, want)
	}

	// Only the chosen Stellium and star are flagged.
	for _, s := range d.Stellia {
		if s.HomeStellium != (s.ID == stelliumID) {
			t.Errorf("stellium %d: homestellium %v", s.ID, s.HomeStellium)
		}
	}
	for _, s := range d.Stars {
		if s.HomeStar != (s.ID == starID) {
			t.Errorf("star %d: homestar %v", s.ID, s.HomeStar)
		}
	}
	// The caller's original slices are untouched.
	if before.Stellia[stelliumID-1].HomeStellium || before.Stars[starID-1].HomeStar {
		t.Errorf("DesignateHomeStar changed the original slices")
	}

	// New records follow the old IDs, and every slice stays in ID order.
	checkIDs := func(name string, ids []int, oldMax int, added int) {
		t.Helper()
		for i := 1; i < len(ids); i++ {
			if ids[i] <= ids[i-1] {
				t.Fatalf("%s: id %d follows %d", name, ids[i], ids[i-1])
			}
		}
		for i, id := range ids[len(ids)-added:] {
			if id != oldMax+1+i {
				t.Errorf("%s: new id %d, want %d", name, id, oldMax+1+i)
			}
		}
	}
	checkIDs("orbits", ids(d.Orbits, func(o game.Orbit) int { return o.ID }), len(before.Orbits), c.Added.Orbits)
	checkIDs("planets", ids(d.Planets, func(p game.Planet) int { return p.ID }), len(before.Planets), c.Added.Planets)
	checkIDs("deposits", ids(d.Deposits, func(x game.Deposit) int { return x.ID }), len(before.Deposits), c.Added.Deposits)

	// The star's orbits and planets match the template.
	orbitNo := map[int]int{} // orbit ID to orbit number, for the star
	for _, o := range d.Orbits {
		if o.StarID == starID {
			orbitNo[o.ID] = o.OrbitNo
		}
	}
	if len(orbitNo) != game.OrbitsPerStar {
		t.Fatalf("star has %d orbits, want %d", len(orbitNo), game.OrbitsPerStar)
	}
	planetsSeen := 0
	for _, p := range d.Planets {
		no, ok := orbitNo[p.OrbitID]
		if !ok {
			continue
		}
		planetsSeen++
		tp := tmpl.Orbits[no-1].Planet
		if tp == nil {
			t.Errorf("orbit %d: got planet %d, want none", no, p.ID)
			continue
		}
		if p.Habitable != tp.Habitable || p.Viability != tp.Viability {
			t.Errorf("orbit %d: got habitable %v viability %d, want %v and %d", no, p.Habitable, p.Viability, tp.Habitable, tp.Viability)
		}
		if (p.ID == c.HomeWorldID) != (no == game.HomeWorldOrbitNo) {
			t.Errorf("orbit %d: planet %d, home world is %d", no, p.ID, c.HomeWorldID)
		}
		var got []game.TemplateDeposit
		for _, x := range d.Deposits {
			if x.PlanetID == p.ID {
				if x.DepositNo != len(got)+1 {
					t.Errorf("planet %d: deposit %d has deposit_no %d", p.ID, x.ID, x.DepositNo)
				}
				got = append(got, game.TemplateDeposit{Type: x.Type, Size: x.Size, Yield: x.Yield})
			}
		}
		if !reflect.DeepEqual(got, tp.Deposits) {
			t.Errorf("orbit %d: got deposits %v, want %v", no, got, tp.Deposits)
		}
	}
	if planetsSeen != tmpl.Planets() {
		t.Errorf("star has %d planets, want %d", planetsSeen, tmpl.Planets())
	}

	// Nothing refers to a removed record.
	orbitIDs := map[int]bool{}
	for _, o := range d.Orbits {
		orbitIDs[o.ID] = true
	}
	planetIDs := map[int]bool{}
	for _, p := range d.Planets {
		planetIDs[p.ID] = true
		if !orbitIDs[p.OrbitID] {
			t.Errorf("planet %d is in removed orbit %d", p.ID, p.OrbitID)
		}
	}
	for _, x := range d.Deposits {
		if !planetIDs[x.PlanetID] {
			t.Errorf("deposit %d is on removed planet %d", x.ID, x.PlanetID)
		}
	}

	// A second home star keeps counting up from the first one's IDs.
	second := starOf(t, d, stelliumID+1, 1)
	lastOrbit := d.Orbits[len(d.Orbits)-1].ID
	if _, err := d.DesignateHomeStar(second, tmpl); err != nil {
		t.Fatalf("second home star: %v", err)
	}
	if got := d.Orbits[len(d.Orbits)-game.OrbitsPerStar].ID; got != lastOrbit+1 {
		t.Errorf("second home star's first orbit is %d, want %d", got, lastOrbit+1)
	}
}

func ids[T any](s []T, id func(T) int) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[i] = id(v)
	}
	return out
}

func TestDesignateHomeStarRejects(t *testing.T) {
	for _, tc := range []struct {
		name    string
		star    func(t *testing.T, d *game.Data) int
		wantErr error
	}{
		{
			name:    "unknown star",
			star:    func(*testing.T, *game.Data) int { return 9999 },
			wantErr: game.ErrNoSuchStar,
		},
		{
			name:    "second star of a two-star stellium",
			star:    func(t *testing.T, d *game.Data) int { return starOf(t, *d, 11, 2) },
			wantErr: game.ErrNotHomeStar,
		},
		{
			name:    "three-star stellium",
			star:    func(t *testing.T, d *game.Data) int { return starOf(t, *d, 3, 1) },
			wantErr: game.ErrNotHomeStar,
		},
		{
			name:    "one-star stellium",
			star:    func(t *testing.T, d *game.Data) int { return starOf(t, *d, 26, 1) },
			wantErr: game.ErrNotHomeStar,
		},
		{
			name: "already home",
			star: func(t *testing.T, d *game.Data) int {
				id := starOf(t, *d, 11, 1)
				if _, err := d.DesignateHomeStar(id, game.DefaultHomeStarTemplate()); err != nil {
					t.Fatal(err)
				}
				return id
			},
			wantErr: game.ErrAlreadyHome,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := goldenData(t)
			id := tc.star(t, &d)
			before := d
			_, err := d.DesignateHomeStar(id, game.DefaultHomeStarTemplate())
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error %v, want %v", err, tc.wantErr)
			}
			if !reflect.DeepEqual(d, before) {
				t.Errorf("data changed on error")
			}
		})
	}
}

func TestDesignateHomeStarFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "game")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	createGame(t, dir)

	// Without a template there is nothing to apply, and nothing changes.
	if _, err := game.DesignateHomeStar(dir, 0); err == nil {
		t.Fatalf("got nil error without a template")
	}

	if _, err := game.CreateHomeStarTemplate(dir); err != nil {
		t.Fatal(err)
	}
	d, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	starID := starOf(t, d, 11, 1)
	want := d
	wantChanges, err := want.DesignateHomeStar(starID, game.DefaultHomeStarTemplate())
	if err != nil {
		t.Fatal(err)
	}

	got, err := game.DesignateHomeStar(dir, starID)
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if !reflect.DeepEqual(got, wantChanges) {
		t.Errorf("got changes %+v, want %+v", got, wantChanges)
	}
	saved, err := game.LoadData(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !reflect.DeepEqual(saved, want) {
		t.Errorf("saved data differs from the designated data")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name()[0] == '.' {
			t.Errorf("left temporary file %s", e.Name())
		}
	}
}

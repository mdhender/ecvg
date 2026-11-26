// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/game"
)

// defaults returns the built-in templates.
func defaults(t *testing.T) (HomeStarTemplate, EmpireTemplate) {
	t.Helper()
	h, err := parseHomeStarTemplate(defaultHomeStarTemplate)
	if err != nil {
		t.Fatalf("default home star template: %v", err)
	}
	e, err := parseEmpireTemplate(defaultEmpireTemplate)
	if err != nil {
		t.Fatalf("default empire template: %v", err)
	}
	return h, e
}

func TestApportion(t *testing.T) {
	for _, tc := range []struct {
		name    string
		total   int
		weights []int
		want    []int
	}{
		{"exact", 10, []int{1, 1}, []int{5, 5}},
		{"tie goes to the earlier weight", 3, []int{1, 1}, []int{2, 1}},
		{"largest fractions first", 138, []int{25, 50, 45, 20, 15}, []int{22, 45, 40, 18, 13}},
		{"nothing", 0, []int{3, 4}, []int{0, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := apportion(tc.total, tc.weights); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("apportion(%d, %v) = %v, want %v", tc.total, tc.weights, got, tc.want)
			}
		})
	}
}

func TestGenerateRules(t *testing.T) {
	h, e := defaults(t)
	d := generate("alpha", 0x1234, h, e)

	// Stellia: 1 with 5 stars, 3 with 4, 10 with 3, 30 with 2, 56 with 1.
	if len(d.Stellia) != 100 {
		t.Fatalf("%d stellia, want 100", len(d.Stellia))
	}
	starsIn := map[int]int{}
	for _, s := range d.Stars {
		starsIn[s.StelliumID]++
	}
	sizes := map[int]int{}
	seen := map[[3]int]bool{}
	for _, s := range d.Stellia {
		sizes[starsIn[s.ID]]++
		if s.HomeStellium != (starsIn[s.ID] == 3) {
			t.Errorf("stellium %d with %d stars: homestellium %v", s.ID, starsIn[s.ID], s.HomeStellium)
		}
		p := [3]int{s.X, s.Y, s.Z}
		if d2 := s.X*s.X + s.Y*s.Y + s.Z*s.Z; d2 == 0 || d2 > 225 || seen[p] {
			t.Errorf("stellium %d at %v: at the origin, out of range, or a duplicate", s.ID, p)
		}
		seen[p] = true
	}
	if want := map[int]int{5: 1, 4: 3, 3: 10, 2: 30, 1: 56}; !reflect.DeepEqual(sizes, want) {
		t.Errorf("stellia by star count = %v, want %v", sizes, want)
	}
	homeStars := map[int]bool{}
	for _, s := range d.Stars {
		if s.HomeStar != (starsIn[s.StelliumID] == 3 && s.SeqNo == 1) {
			t.Errorf("star %d (seq %d of %d): homestar %v", s.ID, s.SeqNo, starsIn[s.StelliumID], s.HomeStar)
		}
		homeStars[s.ID] = s.HomeStar
	}

	// Orbits and planets.
	if len(d.Orbits) != 10*len(d.Stars) {
		t.Errorf("%d orbits, want %d", len(d.Orbits), 10*len(d.Stars))
	}
	orbits := map[int]game.Orbit{}
	for _, o := range d.Orbits {
		orbits[o.ID] = o
	}
	planetsOf := map[int]int{} // by star
	homeWorlds := map[int]bool{}
	random, habitable := 0, 0
	usedOrbits := map[int]bool{}
	for i, p := range d.Planets {
		o := orbits[p.OrbitID]
		if p.ID != i+1 || usedOrbits[p.OrbitID] || (i > 0 && p.OrbitID < d.Planets[i-1].OrbitID) {
			t.Fatalf("planet %d in orbit %d: bad ID, shared orbit, or out of order", p.ID, p.OrbitID)
		}
		usedOrbits[p.OrbitID] = true
		planetsOf[o.StarID]++
		if p.Habitable != (p.Viability != 0) || p.Viability < 0 || p.Viability > 25 {
			t.Errorf("planet %d: habitable %v with viability %d", p.ID, p.Habitable, p.Viability)
		}
		if homeStars[o.StarID] {
			tp := h.Orbits[o.OrbitNo-1].Planet
			if tp == nil || p.Habitable != tp.Habitable || p.Viability != tp.Viability {
				t.Errorf("planet %d in orbit %d of home star %d doesn't match the template", p.ID, o.OrbitNo, o.StarID)
			}
			if o.OrbitNo == 3 {
				homeWorlds[p.ID] = true
			}
			continue
		}
		random++
		if p.Habitable {
			habitable++
			if o.OrbitNo < 2 || o.OrbitNo > 6 {
				t.Errorf("planet %d is habitable in orbit %d", p.ID, o.OrbitNo)
			}
		}
	}
	for _, s := range d.Stars {
		if want := 10; s.HomeStar && planetsOf[s.ID] != want {
			t.Errorf("home star %d has %d planets, want %d", s.ID, planetsOf[s.ID], want)
		} else if planetsOf[s.ID] == 0 {
			t.Errorf("star %d has no planets", s.ID)
		}
	}
	if random != 688 || habitable != 138 { // 153 stars * 4.5; 688 * 125/621
		t.Errorf("%d random planets, %d habitable; want 688, 138", random, habitable)
	}
	if len(homeWorlds) != 10 {
		t.Errorf("%d home worlds, want 10", len(homeWorlds))
	}

	// Deposits.
	depositsOn := map[int][]game.Deposit{}
	for i, x := range d.Deposits {
		if x.ID != i+1 {
			t.Fatalf("deposit %d has ID %d", i+1, x.ID)
		}
		depositsOn[x.PlanetID] = append(depositsOn[x.PlanetID], x)
		if x.DepositNo != len(depositsOn[x.PlanetID]) {
			t.Errorf("deposit %d: deposit_no %d, want %d", x.ID, x.DepositNo, len(depositsOn[x.PlanetID]))
		}
		a, ok := depositAmounts[x.Type]
		if !ok || x.Size < a.minSize || x.Size > a.maxSize || x.Yield < a.minYield || x.Yield > a.maxYield {
			t.Errorf("deposit %d: %s size %d yield %d out of range", x.ID, x.Type, x.Size, x.Yield)
		}
	}
	for _, p := range d.Planets {
		if n := len(depositsOn[p.ID]); n < 1 || n > 31 {
			t.Errorf("planet %d has %d deposits", p.ID, n)
		}
	}
	for id := range homeWorlds {
		var got []TemplateDeposit
		for _, x := range depositsOn[id] {
			got = append(got, TemplateDeposit{Type: x.Type, Size: x.Size, Yield: x.Yield})
		}
		if !reflect.DeepEqual(got, h.Orbits[2].Planet.Deposits) {
			t.Errorf("home world %d: deposits don't match the template", id)
		}
	}
	// Random planets average 16 deposits, give or take the rounding when the
	// distribution is scaled; each home star has the template's 84.
	if got, want := len(d.Deposits)-84*10, 688*16; got < want-68 || got > want+68 {
		t.Errorf("%d deposits on random planets, want about %d", got, want)
	}

	// Races, empires, and colonies.
	if len(d.Races) != 10 || len(d.Empires) != 150 || len(d.Colonies) != 150 {
		t.Fatalf("%d races, %d empires, %d colonies; want 10, 150, 150", len(d.Races), len(d.Empires), len(d.Colonies))
	}
	for _, r := range d.Races {
		if !homeWorlds[r.ID] || r.HomeWorldID != r.ID {
			t.Errorf("race %+v is not on a home world", r)
		}
	}
	empiresOn := map[int]int{}
	for i, emp := range d.Empires {
		if emp.ID != i+1 || emp.RaceID != emp.HomeWorldID || !homeWorlds[emp.HomeWorldID] || emp.Handle != "" {
			t.Errorf("empire %+v", emp)
		}
		empiresOn[emp.HomeWorldID]++
		c := d.Colonies[i]
		tc := e.Colonies[0]
		if c.ID != i+1 || c.EmpireID != emp.ID || c.PlanetID != emp.HomeWorldID || c.Type != tc.Type ||
			c.Population != tc.Population || !reflect.DeepEqual(c.Cargo, tc.Cargo) ||
			!reflect.DeepEqual(c.FarmingGroups, tc.FarmingGroups) ||
			len(c.MiningGroups) != 0 || len(c.FactoryGroups) != len(tc.FactoryGroups) {
			t.Errorf("colony %d doesn't match empire %d and the template", c.ID, emp.ID)
		}
	}
	for id, n := range empiresOn {
		if n != 15 {
			t.Errorf("home world %d has %d empires, want 15", id, n)
		}
	}
}

func TestGenerateDependsOnlyOnSeed(t *testing.T) {
	h, e := defaults(t)
	a := generate("alpha", 1, h, e)
	b := generate("alpha", 1, h, e)
	c := generate("alpha", 2, h, e)
	if !reflect.DeepEqual(a, b) {
		t.Error("the same seed gave different games")
	}
	if reflect.DeepEqual(a.Stellia, c.Stellia) || reflect.DeepEqual(a.Deposits, c.Deposits) {
		t.Error("different seeds gave the same Stellia or deposits")
	}
}

func TestTemplatesRejectInvalid(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{"unknown member", `{"colonies":[],"ships":[]}`},
		{"no colonies", `{"colonies":[]}`},
		{"mining groups", `{"colonies":[{"no":1,"type":"COPN","tech_level":1,"mining_groups":[]}]}`},
		{"colony numbering", `{"colonies":[{"no":2,"type":"COPN","tech_level":1}]}`},
		{"missing type", `{"colonies":[{"no":1,"tech_level":1}]}`},
		{"unknown type", `{"colonies":[{"no":1,"type":"BASE","tech_level":1}]}`},
		{"zero cargo", `{"colonies":[{"no":1,"type":"COPN","tech_level":1,"cargo":[{"type":"FOOD","qty":0}]}]}`},
		{"empty group", `{"colonies":[{"no":1,"type":"COPN","tech_level":1,"farming_groups":[{"no":1,"units":[]}]}]}`},
	} {
		t.Run("empire/"+tc.name, func(t *testing.T) {
			if _, err := parseEmpireTemplate([]byte(tc.data)); err == nil {
				t.Error("accepted")
			}
		})
	}
	for _, tc := range []struct {
		name     string
		old, new string
	}{
		{"unknown member", `"orbits": [`, `"x": 1, "orbits": [`},
		{"home world not marked", `"homeworld": true, `, ``},
		{"home world viability", `"viability": 25`, `"viability": 24`},
		{"second home world", `"habitable": true, "viability": 8`, `"homeworld": true, "habitable": true, "viability": 8`},
		{"deposit size", `"size": 8000000`, `"size": 80000000`},
		{"deposit type", `"type": "GOLD", "size": 8000000`, `"type": "IRON", "size": 8000000`},
	} {
		t.Run("homestar/"+tc.name, func(t *testing.T) {
			data := strings.Replace(string(defaultHomeStarTemplate), tc.old, tc.new, 1)
			if data == string(defaultHomeStarTemplate) {
				t.Fatalf("%q is not in the default template", tc.old)
			}
			if _, err := parseHomeStarTemplate([]byte(data)); err == nil {
				t.Error("accepted")
			}
		})
	}
}

func TestCreate(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "alpha")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	// A home star with only the home world, and empires with twice the soldiers.
	homeStar := `{"orbits":[` +
		`{"orbit_no":1,"planet":null},{"orbit_no":2,"planet":null},` +
		`{"orbit_no":3,"planet":{"homeworld":true,"habitable":true,"viability":25,"deposits":[{"type":"FUEL","size":1000000,"yield":1}]}},` +
		`{"orbit_no":4,"planet":null},{"orbit_no":5,"planet":null},{"orbit_no":6,"planet":null},{"orbit_no":7,"planet":null},` +
		`{"orbit_no":8,"planet":null},{"orbit_no":9,"planet":null},{"orbit_no":10,"planet":null}]}`
	empire := strings.Replace(string(defaultEmpireTemplate), `"sld": 100000`, `"sld": 200000`, 1)
	homeStarPath, empirePath := filepath.Join(root, "h.json"), filepath.Join(root, "e.json")
	for path, data := range map[string]string{homeStarPath: homeStar, empirePath: empire} {
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	d, err := create(dir, 42, homeStarPath, empirePath)
	if err != nil {
		t.Fatal(err)
	}
	if d.Game.Code != "alpha" || d.Game.Seed1 != 42 {
		t.Errorf("game = %+v", d.Game)
	}
	if want := 688 + 10; len(d.Planets) != want {
		t.Errorf("%d planets, want %d", len(d.Planets), want)
	}
	if got := d.Colonies[0].Population.SLD; got != 200000 {
		t.Errorf("colony 1 has %d SLD, want 200000", got)
	}
	loaded, err := game.LoadData(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, d) {
		t.Error("the game loaded from the folder differs from the one created")
	}

	if _, err := create(dir, 42, "", ""); !errors.Is(err, game.ErrGameExists) {
		t.Errorf("second create: err = %v, want ErrGameExists", err)
	}
	other := filepath.Join(root, "beta")
	if err := os.Mkdir(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := create(other, 42, filepath.Join(root, "missing.json"), ""); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing template: err = %v, want os.ErrNotExist", err)
	}
	if entries, _ := os.ReadDir(other); len(entries) != 0 {
		t.Errorf("a failed create left %d files behind", len(entries))
	}
}

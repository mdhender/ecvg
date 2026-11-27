// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package units_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/units"
)

// tables reads the Markdown tables of unit-tables.md, keyed by their header
// row's cells joined with "|". Each row is its trimmed cells.
func tables(t *testing.T) map[string][][]string {
	t.Helper()
	data, err := os.ReadFile("../../unit-tables.md")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string][][]string{}
	header := ""
	for line := range strings.SplitSeq(string(data), "\n") {
		if !strings.HasPrefix(line, "|") {
			header = ""
			continue
		}
		var cells []string
		for cell := range strings.SplitSeq(strings.Trim(line, "|"), "|") {
			cells = append(cells, strings.TrimSpace(cell))
		}
		switch {
		case header == "":
			header = strings.Join(cells, "|")
		case strings.HasPrefix(cells[0], "---"):
		default:
			got[header] = append(got[header], cells)
		}
	}
	return got
}

// rows returns the table with the given header, failing if it isn't there.
func rows(t *testing.T, header string) [][]string {
	t.Helper()
	r := tables(t)[header]
	if len(r) == 0 {
		t.Fatalf("unit-tables.md has no table %q", header)
	}
	return r
}

// TestClasses checks every unit's name, class, and whether it may be kept in
// a depot against the document, and
// that the catalogue holds exactly the document's units.
func TestClasses(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range rows(t, "Item Code|Item Name|Class|Depot?") {
		code, name, class := r[0], r[1], r[2]
		seen[code] = true
		d, ok := units.Lookup(code)
		if !ok {
			t.Errorf("%s: not in the catalogue", code)
			continue
		}
		if d.Code != code || d.Name != name || string(d.Class) != class {
			t.Errorf("%s: got %q, %q, %q; want name %q, class %q", code, d.Code, d.Name, d.Class, name, class)
		}
		if depot := r[3] == "yes"; d.DepotAllowed() != depot || (r[3] != "yes" && r[3] != "no") {
			t.Errorf("%s: DepotAllowed = %v, but the document says %q", code, d.DepotAllowed(), r[3])
		}
	}
	for _, d := range units.All() {
		if !seen[d.Code] {
			t.Errorf("%s: in the catalogue but not in unit-tables.md", d.Code)
		}
	}
}

// TestFlags checks which units carry a tech level and need assembly.
func TestFlags(t *testing.T) {
	listed := map[string]bool{}
	for _, r := range rows(t, "unit_code|tech_level?|requires_assembly?") {
		listed[r[0]] = true
		d, ok := units.Lookup(r[0])
		if !ok {
			t.Errorf("%s: not in the catalogue", r[0])
			continue
		}
		if tl, asm := r[1] == "yes", r[2] == "yes"; d.HasTechLevel != tl || d.RequiresAssembly != asm {
			t.Errorf("%s: tech level %v, assembly %v; want %v, %v", r[0], d.HasTechLevel, d.RequiresAssembly, tl, asm)
		}
	}
	// The units the document leaves out are people and research points.
	for _, d := range units.All() {
		if listed[d.Code] {
			continue
		}
		if d.HasTechLevel || d.RequiresAssembly || d.Manufactured || !(d.Population() || d.Code == units.RSCH) {
			t.Errorf("%s: not in the document's catalogue, but %+v", d.Code, d)
		}
	}
}

// TestCatalogue checks every row of the document's generated catalogue: lot
// size, inputs, mass, volume, and stowed volume, each for a whole lot.
func TestCatalogue(t *testing.T) {
	// The document's numbers may be halves, so compare two lots' worth.
	twice := func(t *testing.T, s string) int {
		t.Helper()
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatal(err)
		}
		return int(2 * f)
	}
	count := map[string]int{}
	for _, r := range rows(t, "unit_code|tech_level|lot_size|mtl_input|nmtl_input|mass_units|volume_units|stowed_volume_units") {
		t.Run(r[0]+"-"+r[1], func(t *testing.T) {
			d, ok := units.Lookup(r[0])
			if !ok {
				t.Fatal("not in the catalogue")
			}
			count[d.Code]++
			tl, _ := strconv.Atoi(r[1])
			if !d.ValidTechLevel(tl) {
				t.Errorf("tech level %d is not valid", tl)
			}
			lot, _ := strconv.Atoi(r[2])
			if got := d.LotSize(tl); got != lot {
				t.Fatalf("lot size %d, want %d", got, lot)
			}
			mtl, nmtl, ok := d.Inputs(tl, 2)
			switch {
			case r[3] == "TBD" || r[3] == "N/A":
				if ok {
					t.Errorf("inputs %d, %d; want none", mtl, nmtl)
				}
			case !ok || mtl != twice(t, r[3]) || nmtl != twice(t, r[4]):
				t.Errorf("inputs for 2 lots %d, %d, %v; want twice %s, %s", mtl, nmtl, ok, r[3], r[4])
			}
			if got, want := d.Mass(tl, 2*lot), twice(t, r[5]); got != want {
				t.Errorf("mass of 2 lots %d, want %d", got, want)
			}
			if got, want := d.Volume(tl, 2*lot), twice(t, r[6]); got != want {
				t.Errorf("volume of 2 lots %d, want %d", got, want)
			}
			if got, want := d.StowedVolume(tl, 2*lot), twice(t, r[7]); got != want {
				t.Errorf("stowed volume of 2 lots %d, want %d", got, want)
			}
		})
	}
	// One row per valid tech level.
	for _, d := range units.All() {
		want := 0
		switch {
		case d.HasTechLevel:
			want = units.MaxTechLevel - units.MinTechLevel + 1
		case !d.Population() && d.Code != units.RSCH:
			want = 1
		}
		if count[d.Code] != want {
			t.Errorf("%s: %d rows in the document, want %d", d.Code, count[d.Code], want)
		}
	}
}

// TestRounding checks the document's worked examples: totals round up once.
func TestRounding(t *testing.T) {
	def := func(code string) units.Def {
		d, ok := units.Lookup(code)
		if !ok {
			t.Fatalf("%s: not in the catalogue", code)
		}
		return d
	}
	for _, tc := range []struct {
		name      string
		got, want int
	}{
		{"40 PROB-3 mass", def(units.PROB).Mass(3, 40), 223},
		{"10 stowed ACFT-1", def(units.ACFT).StowedVolume(1, 10), 25},
		{"11 stowed ACFT-1", def(units.ACFT).StowedVolume(1, 11), 28},
		{"1 CNGD mass", def(units.CNGD).Mass(0, 1), 1},
		{"515,279 CNGD mass", def(units.CNGD).Mass(0, 515279), 309168},
		{"3 FUEL volume", def(units.FUEL).Volume(0, 3), 2},
		{"20,000 STRL-10 mass", def(units.STRL).Mass(10, 20000), 20000},
		{"650,000 stowed FACT-10", def(units.FACT).StowedVolume(10, 650000), 10400000},
		{"no units", def(units.FACT).Mass(10, 0), 0},
		{"a population unit", def(units.USK).Mass(0, 7), 7},
		{"research points", def(units.RSCH).Mass(0, 7), 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %d, want %d", tc.got, tc.want)
			}
		})
	}
}

func TestLookup(t *testing.T) {
	for _, code := range []string{"", "fact", "ORE", "FACTS", "ZZZZ"} {
		if d, ok := units.Lookup(code); ok {
			t.Errorf("Lookup(%q) = %+v, want none", code, d)
		}
	}
	d, _ := units.Lookup(units.FOOD)
	for tl, want := range map[int]bool{-1: false, 0: true, 1: false} {
		if got := d.ValidTechLevel(tl); got != want {
			t.Errorf("FOOD tech level %d: valid = %v, want %v", tl, got, want)
		}
	}
	d, _ = units.Lookup(units.FARM)
	for tl, want := range map[int]bool{0: false, 1: true, 10: true, 11: false} {
		if got := d.ValidTechLevel(tl); got != want {
			t.Errorf("FARM tech level %d: valid = %v, want %v", tl, got, want)
		}
	}
	if _, _, ok := d.Inputs(1, 1); !ok {
		t.Error("FARM-1 has no inputs")
	}
	d, _ = units.Lookup(units.FOOD)
	if _, _, ok := d.Inputs(0, 1); ok {
		t.Error("FOOD has inputs")
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package report writes the turn reports sent to players. A report is plain
// text describing one empire as of the start of a turn.
package report

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mdhender/ecvg/internal/game"
)

// FileName returns the name of the report file for the empire with ID
// empireID.
func FileName(empireID int) string {
	return fmt.Sprintf("empire-%03d.txt", empireID)
}

// WriteAll writes a report for every empire in d into the reports folder of
// the game's current turn, dir/turns/{turn}/reports, creating the folder if
// need be and replacing reports already there. It returns the folder and the
// number of reports written. Every report is made before any is written.
func WriteAll(dir string, d game.Data) (string, int, error) {
	reports := make([][]byte, len(d.Empires))
	for i, e := range d.Empires {
		var buf bytes.Buffer
		if err := Empire(&buf, d, e.ID); err != nil {
			return "", 0, err
		}
		reports[i] = buf.Bytes()
	}
	out := filepath.Join(game.TurnDir(dir, d.Game.Turn), game.ReportsDirName)
	if err := os.Mkdir(out, 0o755); err != nil && !os.IsExist(err) {
		return "", 0, err
	}
	for i, e := range d.Empires {
		if err := os.WriteFile(filepath.Join(out, FileName(e.ID)), reports[i], 0o644); err != nil {
			return "", 0, err
		}
	}
	return out, len(reports), nil
}

// entityNames describes each entity type.
var entityNames = map[game.EntityType]string{
	game.CSHP: "ship",
	game.COPN: "open-air surface colony",
	game.CENC: "enclosed surface colony",
	game.CORB: "enclosed orbital colony",
}

// Empire writes the report for the empire with ID empireID as of the game's
// current turn: the empire, each of its colonies (population, units, cargo,
// and groups), and the deposits of its home world.
func Empire(w io.Writer, d game.Data, empireID int) error {
	var empire *game.Empire
	for i := range d.Empires {
		if d.Empires[i].ID == empireID {
			empire = &d.Empires[i]
		}
	}
	if empire == nil {
		return fmt.Errorf("empire %d: no such empire", empireID)
	}
	home, err := locate(d, empire.HomeWorldID)
	if err != nil {
		return fmt.Errorf("empire %d: home world: %w", empireID, err)
	}

	var b strings.Builder
	p := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	rule := strings.Repeat("=", 72)

	p("%s", rule)
	p("EMPYREAN CHALLENGE   Game %s   Turn %d", d.Game.Code, d.Game.Turn)
	p("%s", rule)
	p("Empire %d: %s", empire.ID, empire.Name)
	player := empire.Handle
	if player == "" {
		player = "(none)"
	}
	p("Player: %s", player)
	p("Race:   %d", empire.RaceID)
	p("Home world: planet %d at %s", empire.HomeWorldID, home)

	colonies := 0
	for _, c := range d.Colonies {
		if c.EmpireID != empireID {
			continue
		}
		colonies++
		loc, err := locate(d, c.PlanetID)
		if err != nil {
			return fmt.Errorf("empire %d: colony %d: %w", empireID, c.ID, err)
		}
		p("")
		p("%s", strings.Repeat("-", 72))
		p("COLONY %d   %s (%s)   Tech level %d", c.No, c.Type, entityNames[c.Type], c.TechLevel)
		p("%s", strings.Repeat("-", 72))
		p("Entity ID: %d", c.ID)
		p("Location: planet %d at %s", c.PlanetID, loc)
		p("Rations %d%%   Standard of living %.2f   Birth rate %s   Death rate %s", c.Rations, c.SOL, percent(c.BirthRate), percent(c.DeathRate))

		pop := c.Population
		p("")
		p("POPULATION %14s %15s", "Units", "People")
		for _, row := range []struct {
			name  string
			units int
		}{{"UEM", pop.UEM}, {"USK", pop.USK}, {"PRO", pop.PRO}, {"SLD", pop.SLD}, {"Total", pop.Units()}} {
			p("  %-8s %14s %15s", row.name, commas(row.units), commas(row.units*game.PeoplePerPopulationUnit))
		}
		p("  Food needed each turn: %s FOOD", commas(pop.Food()))
		if c.Type.Enclosed() {
			p("  Space taken: %s VU   Life support needed: %s LU", commas(pop.Volume(c.Type)), commas(pop.LifeSupport(c.Type)))
		}

		units := func(title string, us []game.Unit) {
			p("")
			p("%s", title)
			p("  %-8s %4s %14s", "Unit", "TL", "Qty")
			if len(us) == 0 {
				p("  None")
			}
			for _, u := range us {
				tl := ""
				if u.TechLevel != 0 {
					tl = strconv.Itoa(u.TechLevel)
				}
				p("  %-8s %4s %14s", u.Type, tl, commas(u.Qty))
			}
		}
		units("OPERATIONAL", c.Operational)
		units("CARGO", c.Cargo)

		groups := func(title string, gs []game.Group) {
			p("")
			p("%s", title)
			if len(gs) == 0 {
				p("  None assembled")
			}
			for _, g := range gs {
				for _, u := range g.Units {
					p("  Group %-3d %14s units at TL %d", g.No, commas(u.Qty), u.TechLevel)
				}
			}
		}
		groups("FARMING GROUPS", c.FarmingGroups)
		p("")
		p("MINING GROUPS")
		if len(c.MiningGroups) == 0 {
			p("  None assembled")
		}
		for _, g := range c.MiningGroups {
			for _, u := range g.Units {
				p("  Group %-3d %14s units at TL %d working deposit %d", g.No, commas(u.Qty), u.TechLevel, g.DepositID)
			}
		}
		groups("FACTORY GROUPS", c.FactoryGroups)
	}
	if colonies == 0 {
		p("")
		p("No colonies.")
	}

	p("")
	p("%s", strings.Repeat("-", 72))
	p("HOME WORLD DEPOSITS   planet %d", empire.HomeWorldID)
	p("%s", strings.Repeat("-", 72))
	p("  %3s  %-5s %14s %6s", "No", "Type", "Size", "Yield")
	for _, x := range d.Deposits {
		if x.PlanetID == empire.HomeWorldID {
			p("  %3d  %-5s %14s %5d%%", x.DepositNo, x.Type, commas(x.Size), x.Yield)
		}
	}
	_, err = io.WriteString(w, b.String())
	return err
}

// locate describes where the planet with ID planetID is: its Stellium's
// coordinates, its star, and its orbit.
func locate(d game.Data, planetID int) (string, error) {
	for _, planet := range d.Planets {
		if planet.ID != planetID {
			continue
		}
		for _, o := range d.Orbits {
			if o.ID != planet.OrbitID {
				continue
			}
			for _, star := range d.Stars {
				if star.ID != o.StarID {
					continue
				}
				for _, s := range d.Stellia {
					if s.ID == star.StelliumID {
						return fmt.Sprintf("(%d, %d, %d) star %d orbit %d", s.X, s.Y, s.Z, star.SeqNo, o.OrbitNo), nil
					}
				}
			}
		}
		return "", fmt.Errorf("planet %d: can't find its orbit, star, or stellium", planetID)
	}
	return "", fmt.Errorf("planet %d: no such planet", planetID)
}

// commas formats n with a comma between each group of three digits.
func commas(n int) string {
	s := strconv.Itoa(n)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	if neg {
		s = "-" + s
	}
	return s
}

// percent formats the fraction r as a percentage, without trailing zeros.
func percent(r float64) string {
	return strconv.FormatFloat(r*100, 'f', -1, 64) + "%"
}

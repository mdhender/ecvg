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
// current turn: the empire, what became of the orders it gave in the prior
// turn, each of its colonies (population, units, cargo,
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

	if d.Game.Turn > 0 {
		p("")
		p("%s", strings.Repeat("-", 72))
		p("ORDERS RECEIVED FOR TURN %d", d.Game.Turn-1)
		p("%s", strings.Repeat("-", 72))
		received := 0
		for _, r := range d.OrderResults {
			if r.EmpireID != empireID {
				continue
			}
			received++
			p("  Line %d: %s", r.Line, r.Text)
			if r.Accepted {
				p("    Accepted")
			} else {
				p("    REJECTED: %s", r.Reason)
			}
		}
		if received == 0 {
			p("  None")
		}
	}

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
		p("  Food eaten each turn at %d%% rations: %s FOOD", c.Rations, commas(pop.FoodWanted(c.Rations)))
		if s := c.Starved; s.Units() > 0 {
			p("  STARVED LAST TURN: %s units (UEM %s, USK %s, PRO %s, SLD %s)",
				commas(s.Units()), commas(s.UEM), commas(s.USK), commas(s.PRO), commas(s.SLD))
		}
		if c.Type.Enclosed() {
			p("  Space taken: %s VU   Life support needed: %s LU", commas(pop.Volume()), commas(pop.LifeSupport(c.Type)))
			if have, need := c.LifeSupportCapacity(), pop.LifeSupport(c.Type); have < need {
				p("  LIFE SUPPORT SHORT: the assembled LFSU support %s units, so %s have none and the death rate will rise", commas(have), commas(need-have))
			} else {
				p("  The assembled LFSU support %s units", commas(have))
			}
		} else {
			p("  Space wanted: %s VU", commas(pop.Volume()))
		}
		if l := c.LifeSupport; l.Units > 0 || l.Short() > 0 {
			p("  LIFE SUPPORT LAST TURN: %s of %s LFSU ran, burned %s FUEL, and supported %s of the %s units that needed it",
				commas(l.Worked), commas(l.Units), commas(l.Fuel), commas(min(l.Supported, l.Needed)), commas(l.Needed))
			if l.Unfuelled > 0 {
				p("  %s LFSU idle: no FUEL", commas(l.Unfuelled))
			}
			if l.Short() > 0 {
				p("  %s units had NO LIFE SUPPORT: the death rate rose", commas(l.Short()))
			}
		}
		if s := c.LifeSupport.Died; s.Units() > 0 {
			p("  DIED LAST TURN, of the raised death rate: %s units (UEM %s, USK %s, PRO %s, SLD %s)",
				commas(s.Units()), commas(s.UEM), commas(s.USK), commas(s.PRO), commas(s.SLD))
		}

		if k := c.Cadres; k != (game.Cadres{}) || len(c.Drafted)+len(c.Disbanded) > 0 || c.Trained.Trainees > 0 {
			// Cadres aren't population units: they are made up of them.
			num := func(n int) string {
				if n == 0 {
					return ""
				}
				return commas(n)
			}
			p("")
			p("CADRES %18s %14s %14s %14s", "Size", "PRO held", "USK held", "SLD held")
			if k == (game.Cadres{}) {
				p("  None")
			}
			for _, row := range []struct {
				name                string
				size, pro, usk, sld int
			}{
				{"CWRK", k.CWRK, k.CWRK, k.CWRK, 0},
				{"TRNE", k.TRNE, k.PRO() - k.CWRK - k.SPY, k.TRNE, 0},
				{"SPY", k.SPY, k.SPY, 0, k.SPY},
			} {
				if row.size > 0 {
					p("%s", strings.TrimRight(fmt.Sprintf("  %-8s %14s %14s %14s %14s", row.name, commas(row.size), num(row.pro), num(row.usk), num(row.sld)), " "))
				}
			}
			if t := c.Trained; t.Trainees > 0 {
				p("  TRAINED LAST TURN: %s of %s TRNE became PRO, and the cadre was disbanded", commas(t.Graduated), commas(t.Trainees))
			}
			for _, dr := range c.Disbanded {
				if dr.Drafted < dr.Asked {
					p("  DISBANDED LAST TURN: %s of the %s %s ordered: there were no more", commas(dr.Drafted), commas(dr.Asked), dr.Unit)
				} else {
					p("  DISBANDED LAST TURN: %s %s", commas(dr.Drafted), dr.Unit)
				}
			}
			for _, dr := range c.Drafted {
				if dr.Drafted < dr.Asked {
					p("  DRAFTED LAST TURN: %s of the %s %s ordered: no more could be drafted", commas(dr.Drafted), commas(dr.Asked), dr.Unit)
				} else {
					p("  DRAFTED LAST TURN: %s %s", commas(dr.Drafted), dr.Unit)
				}
			}
		}
		if len(c.Setups) > 0 {
			p("")
			p("SET UPS")
			for _, s := range c.Setups {
				p("  Entity %d (%s): %s of %s EMU of assembly work done; %s CWRK shuttled to it last turn",
					s.Entity.ID, s.Entity.Type, commas(s.EMU), commas(s.Need), commas(s.Crew))
				switch s.Waiting {
				case "structure":
					p("    WAITING: no assembly work is done until all of the structure is supplied")
				case "population":
					p("    WAITING: none of the people reserved for it are left")
				}
				list := func(us []game.Unit) string {
					var names []string
					for _, u := range us {
						names = append(names, commas(u.Qty)+" "+unitName(u.Type, u.TechLevel))
					}
					return strings.Join(names, ", ")
				}
				p("    RESERVED: %s", list(s.BOM))
				if short := s.Short(); len(short) > 0 {
					p("    SHORT: %s", list(short))
				}
				if s.Hauled > 0 {
					p("    LATE SUPPLIES: the TRNS carried %s MU for it last turn", commas(s.Hauled))
				}
			}
		}
		if len(c.Assembly) > 0 || c.Construction.Used > 0 {
			p("")
			p("ASSEMBLY")
			if k := c.Construction; k.Pool > 0 || k.Used > 0 {
				p("  LAST TURN: the CWRK did %s EMU of assembly work of the %s EMU they could", commas(k.Used), commas(k.Pool))
				if k.Setup > 0 {
					p("  SET UPS: %s EMU of that was the CWRK reserved for set ups or shuttled to them", commas(k.Setup))
				}
			} else {
				p("  LAST TURN: no CWRK, so no assembly work was done")
			}
			for _, a := range c.Assembly {
				if a.Action == game.WorkTechLevel {
					if a.Unit.Qty == 0 {
						p("  Tech level raised to TL %d last turn, and the work is done", a.Unit.TechLevel)
						continue
					}
					limit := ""
					if a.MaxCWRK != 0 {
						limit = fmt.Sprintf(" (using at most %s CWRK)", commas(a.MaxCWRK))
					}
					p("  Tech level: raising to TL %d, with %s EMU done of the %s EMU it takes%s",
						a.Unit.TechLevel, commas(a.EMU), commas(a.Need), limit)
					continue
				}
				name := unitName(a.Unit.Type, a.Unit.TechLevel)
				done := map[game.Work]string{game.WorkAssemble: "assembled", game.WorkDisassemble: "disassembled", game.WorkScrap: "scrapped"}[a.Action]
				what := ""
				if a.GroupNo != 0 {
					what = fmt.Sprintf("%s group %d: ", title(a.Kind), a.GroupNo)
				}
				limit := ""
				if a.MaxCWRK != 0 {
					limit = fmt.Sprintf(" (using at most %s CWRK)", commas(a.MaxCWRK))
				}
				if a.Unit.Qty == 0 {
					p("  %s%s %s %s last turn, and the work is done", what, commas(a.Assembled), name, done)
				} else {
					p("  %s%s %s %s last turn; %s still to %s, with %s EMU done toward them%s",
						what, commas(a.Assembled), name, done, commas(a.Unit.Qty), a.Action.Verb(), commas(a.EMU), limit)
				}
				if a.Destroyed > 0 {
					p("      LOST: %s of them, in taking them down", commas(a.Destroyed))
				}
				if a.Action == game.WorkScrap && a.Assembled > 0 {
					p("      RECOVERED: %s MTL and %s NMTL", commas(a.MTL), commas(a.NMTL))
					if a.Lost > 0 {
						p("      LOST: %s of that, which the entity had no room for", commas(a.Lost))
					}
				}
				switch {
				case a.Waiting == "room":
					p("      WAITING: no room in the entity for more of them %s", done)
				case a.Waiting == "units" && a.Action == game.WorkDisassemble:
					p("      WAITING: the units aren't assembled")
				case a.Waiting == "units":
					p("      WAITING: the units aren't in storage")
				}
			}
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
		p("")
		p("SPACE %19s", "VU")
		// The engine keeps space exactly; the report rounds each line up, so
		// the lines needn't add up.
		p("  %-8s %14s", "Capacity", commas(c.Capacity().Ceil()))
		p("  %-8s %14s", "Used", commas(c.SpaceUsed().Ceil()))
		if free := c.SpaceFree(); free.Sign() >= 0 {
			p("  %-8s %14s", "Free", commas(free.Ceil()))
		} else {
			why := "OVER CAPACITY: nothing more can be stored in cargo"
			if !c.Type.Enclosed() {
				why = "an open-air colony may hold more than it encloses"
			}
			p("  %-8s %14s   %s", "Over by", commas(c.SpaceUsed().Sub(c.Capacity()).Ceil()), why)
		}
		units("OPERATIONAL", c.Operational)
		units("CARGO", c.Cargo)
		switch {
		case c.HasDepot():
			units("DEPOT, OPERATIONAL", c.DepotOperational)
			units("DEPOT, STORED", c.Depot)
		case len(c.Depot) > 0:
			units("EXTERNAL DEPOT, STORED", c.Depot)
		}

		idleRows := func(idle game.Idle) {
			for _, row := range []struct {
				n   int
				why string
			}{{idle.Ordered, "BY ORDER"}, {idle.Unstaffed, "NO STAFF"}, {idle.Unfuelled, "NO FUEL"}} {
				if row.n > 0 {
					p("            %14s of them were IDLE last turn: %s", commas(row.n), row.why)
				}
			}
		}
		groups := func(title string, gs []game.Group) {
			p("")
			p("%s", title)
			if len(gs) == 0 {
				p("  None assembled")
			}
			for _, g := range gs {
				if g.ShutDown > 0 {
					p("  Group %-3d SHUT DOWN BY ORDER: %s of its units", g.No, commas(g.ShutDown))
				}
				if u := g.Producing; u.Type != "" {
					p("  Group %-3d producing %s", g.No, unitName(u.Type, u.TechLevel))
				}
				if g.Retooled {
					p("  Group %-3d RETOOLED: waiting to be told what to produce", g.No)
				}
				for _, u := range g.Units {
					p("  Group %-3d %14s units at TL %d", g.No, commas(u.Qty), u.TechLevel)
					idleRows(u.Idle)
				}
			}
		}
		if w := c.Workers; w.PRO > 0 || w.USK > 0 {
			p("")
			p("WORKERS LAST TURN %7s %14s %14s", "Pool", "Used", "Idle")
			p("  %-8s %14s %14s %14s", "PRO", commas(w.PRO), commas(w.PRO-w.IdlePRO), commas(w.IdlePRO))
			p("  %-8s %14s %14s %14s", "USK", commas(w.USK), commas(w.USK-w.IdleUSK), commas(w.IdleUSK))
			if w.AutoUSK > 0 {
				p("  %s of the USK were automation (AUTO)", commas(w.AutoUSK))
			}
		}
		groups("FARMING GROUPS", c.FarmingGroups)
		if h := c.Harvest; h.Units > 0 {
			p("  LAST TURN: %s of %s units worked, burned %s FUEL, and made %s FOOD",
				commas(h.Worked), commas(h.Units), commas(h.Fuel), commas(h.Food))
			if h.Ordered > 0 {
				p("  %s units idle: by order", commas(h.Ordered))
			}
			if h.Unstaffed > 0 {
				p("  %s units idle: no staff (each needs %d PRO and %d USK)", commas(h.Unstaffed), game.FarmPRO, game.FarmUSK)
			}
			if h.Unfuelled > 0 {
				p("  %s units idle: no FUEL", commas(h.Unfuelled))
			}
			if h.Lost > 0 {
				p("  %s FOOD LOST: no room to store it", commas(h.Lost))
			}
		}
		p("")
		p("MINING GROUPS")
		if len(c.MiningGroups) == 0 {
			p("  None assembled")
		}
		for _, g := range c.MiningGroups {
			// Players know a deposit by its number on its planet.
			resource, depositNo := "product", g.DepositID
			for _, dep := range d.Deposits {
				if dep.ID == g.DepositID {
					resource, depositNo = string(dep.Type), dep.DepositNo
				}
			}
			if g.ShutDown > 0 {
				p("  Group %-3d SHUT DOWN BY ORDER: %s of its units", g.No, commas(g.ShutDown))
			}
			for _, u := range g.Units {
				p("  Group %-3d %14s units at TL %d working deposit %d (%s)", g.No, commas(u.Qty), u.TechLevel, depositNo, resource)
				idleRows(u.Idle)
			}
			if m := g.Mined; m.Worked > 0 || m.Fuel > 0 || m.Product > 0 || m.Returned > 0 {
				p("            LAST TURN: %s units worked, burned %s FUEL, and mined %s %s",
					commas(m.Worked), commas(m.Fuel), commas(m.Product), resource)
				if m.Returned > 0 {
					p("            %s %s RETURNED TO THE DEPOSIT: no room to store it", commas(m.Returned), resource)
				}
			}
		}
		groups("FACTORY GROUPS", c.FactoryGroups)
		if t := c.Transport; len(t.Transfers) > 0 {
			p("")
			p("TRANSFERS")
			p("  LAST TURN: the TRNS carried %s MU of the %s MU they could, with %s PRO to operate them, and burned %s FUEL",
				commas(t.Used), commas(t.Capacity), commas(t.Crew), commas(t.Fuel))
			for _, x := range t.Transfers {
				short := ""
				if x.Unit.Qty < x.Ordered {
					short = fmt.Sprintf(" (of the %s ordered)", commas(x.Ordered))
				}
				p("  %s %s to entity %d%s", commas(x.Unit.Qty), unitName(x.Unit.Type, x.Unit.TechLevel), x.To, short)
			}
		}
		p("")
		p("LABORATORY GROUPS")
		if len(c.LaboratoryGroups) == 0 {
			p("  None assembled")
		}
		for _, g := range c.LaboratoryGroups {
			if g.ShutDown > 0 {
				p("  Group %-3d SHUT DOWN BY ORDER: %s of its units", g.No, commas(g.ShutDown))
			}
			for _, u := range g.Units {
				p("  Group %-3d %14s units at TL %d", g.No, commas(u.Qty), u.TechLevel)
				idleRows(u.Idle)
			}
		}
		if r := c.Research; r.Units > 0 {
			p("  LAST TURN: %s of %s units worked, burned %s FUEL, and made %s RSCH",
				commas(r.Worked), commas(r.Units), commas(r.Fuel), commas(r.RSCH))
			if r.Ordered > 0 {
				p("  %s units idle: by order", commas(r.Ordered))
			}
			if r.Unstaffed > 0 {
				p("  %s units idle: no staff (each needs %d PRO and %d USK)", commas(r.Unstaffed), game.LabPRO, game.LabUSK)
			}
			if r.Unfuelled > 0 {
				p("  %s units idle: no FUEL", commas(r.Unfuelled))
			}
		}
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

// plural returns one if n is 1 and many otherwise.
func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// unitName formats a unit as orders write it: FACT-10, or CNGD.
func unitName(code string, tl int) string {
	if tl == 0 {
		return code
	}
	return fmt.Sprintf("%s-%d", code, tl)
}

// title returns the kind of group with a capital letter.
func title(k game.GroupKind) string {
	s := string(k)
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

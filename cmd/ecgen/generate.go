// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package main

import (
	"fmt"
	"slices"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/prng"
)

const (
	// homeStelliumStars is the number of stars in a home Stellium: the first
	// star of every Stellium with this many stars is a home star.
	homeStelliumStars = 3
	// empiresPerHomeWorld is the number of empires created on each home world.
	empiresPerHomeWorld = 15
)

// starsInStellium returns the number of stars in the i'th Stellium generated:
// 1 Stellium has 5 stars, 3 have 4, 10 have 3, 30 have 2, and the rest have 1.
func starsInStellium(i int) int {
	switch {
	case i < 1:
		return 5
	case i < 4: // the next 3
		return 4
	case i < 14: // the next 10
		return 3
	case i < 44: // the next 30
		return 2
	default:
		return 1
	}
}

// habitableShares gives the orbits whose planets can be habitable and the
// share of the habitable planets each orbit gets.
var habitableShares = []struct{ orbitNo, share int }{
	{2, 25},
	{3, 50},
	{4, 45},
	{5, 20},
	{6, 15},
}

// depositShares gives the share of the randomly generated deposits that each
// type gets, in the order they are dealt before shuffling.
var depositShares = []struct {
	typ   game.DepositType
	share int
}{
	{game.GOLD, 2},
	{game.FUEL, 34},
	{game.MTL, 32},
	{game.NMTL, 32},
}

// depositAmounts gives the range of the size and yield of each type of
// deposit.
var depositAmounts = map[game.DepositType]struct{ minSize, maxSize, minYield, maxYield int }{
	game.GOLD: {1_000_000, 20_000_000, 1, 12},
	game.FUEL: {1_000_000, 99_000_000, 1, 22},
	game.MTL:  {1_000_000, 99_000_000, 1, 28},
	game.NMTL: {1_000_000, 99_000_000, 1, 28},
}

// apportion splits total among the weights in proportion, rounding down, then
// gives what is left over one each to the weights with the largest fractions
// dropped (the earlier weight first on a tie). The results add up to total.
func apportion(total int, weights []int) []int {
	sum := 0
	for _, w := range weights {
		sum += w
	}
	out := make([]int, len(weights))
	remainders := make([]int, len(weights)) // in units of 1/sum
	left := total
	for i, w := range weights {
		out[i] = total * w / sum
		remainders[i] = total * w % sum
		left -= out[i]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range remainders {
			if remainders[i] > remainders[best] {
				best = i
			}
		}
		out[best]++
		remainders[best] = -1
	}
	return out
}

// bellCurve returns a value in [lo, hi] on a bell curve: lo plus the sum of
// three dice, numbered from 0, whose highest faces add up to hi-lo. When hi-lo
// does not divide by 3, the first dice are one larger.
func bellCurve(s *prng.Stream, lo, hi int) int {
	span, v := hi-lo, lo
	for die := range 3 {
		faces := span/3 + 1
		if die < span%3 {
			faces++
		}
		v += s.IntN(faces)
	}
	return v
}

// generate returns a new game with the given code and first seed; the second
// seed is prng.SplitMix64(seed1).
//
// The first star of every Stellium with homeStelliumStars stars is a home
// star, whose orbits, planets, and deposits come from h. Everything around the
// other stars is random. Each home world (the planet in orbit
// game.HomeWorldOrbitNo of a home star) gets a race and empiresPerHomeWorld
// empires of that race, each with the colonies in e.
func generate(code string, seed1 uint64, h HomeStarTemplate, e EmpireTemplate) game.Data {
	var d game.Data
	d.Game = game.Game{Code: code, Seed1: game.Seed(seed1), Seed2: game.Seed(prng.SplitMix64(seed1))}
	root := prng.New(uint64(d.Game.Seed1), uint64(d.Game.Seed2))

	d.Stellia, d.Stars = generateStellia(root)
	for _, star := range d.Stars {
		for orbitNo := 1; orbitNo <= game.OrbitsPerStar; orbitNo++ {
			d.Orbits = append(d.Orbits, game.Orbit{ID: len(d.Orbits) + 1, StarID: star.ID, OrbitNo: orbitNo})
		}
	}
	templated := generatePlanets(root, &d, h)
	markHabitable(root, &d, templated)
	generateDeposits(root, &d, templated)
	generateEmpires(&d, templated, e)
	d.OrderResults = []game.OrderResult{} // no orders have been given yet
	return d
}

// generateStellia places the Stellia and their stars using the stream
// root.Derive(TagStellia). Every point within game.StelliaRadius of the
// origin, other than the origin itself, is a candidate; the candidates are
// shuffled and the first game.NumStellia are taken. They get the number of
// stars starsInStellium gives, in that order. Stellium IDs run from 1 in that
// order, and star IDs run from 1 across all the Stellia. Stellia with
// homeStelliumStars stars are flagged as home Stellia, and their first stars
// as home stars.
func generateStellia(root *prng.Stream) ([]game.Stellium, []game.Star) {
	const r = game.StelliaRadius
	var points [][3]int
	for x := -r; x <= r; x++ {
		for y := -r; y <= r; y++ {
			for z := -r; z <= r; z++ {
				if d2 := x*x + y*y + z*z; d2 != 0 && d2 <= r*r {
					points = append(points, [3]int{x, y, z})
				}
			}
		}
	}
	root.Derive(prng.TagStellia).Shuffle(len(points), func(i, j int) { points[i], points[j] = points[j], points[i] })

	stellia := make([]game.Stellium, game.NumStellia)
	var stars []game.Star
	for i := range stellia {
		p, n := points[i], starsInStellium(i)
		home := n == homeStelliumStars
		stellia[i] = game.Stellium{ID: i + 1, X: p[0], Y: p[1], Z: p[2], HomeStellium: home}
		for seqNo := 1; seqNo <= n; seqNo++ {
			stars = append(stars, game.Star{ID: len(stars) + 1, StelliumID: stellia[i].ID, SeqNo: seqNo, HomeStar: home && seqNo == 1})
		}
	}
	return stellia, stars
}

// generatePlanets sets d.Planets, at most one planet per orbit, with IDs from
// 1 in orbit ID order. It returns the template planet of every planet that
// came from h, by planet ID.
//
// A home star gets the planets in h, with their habitability and viability.
// The other stars share an average of 4.5 planets each (rounded down), placed
// using the stream root.Derive(TagPlanets): for each star in order, its orbits
// are shuffled and a planet goes in the first, so every star has a planet; the
// orbits left over from every star are then shuffled together, and the rest of
// the planets go in the first of them.
func generatePlanets(root *prng.Stream, d *game.Data, h HomeStarTemplate) map[int]*TemplatePlanet {
	orbitsOf := map[int][]game.Orbit{} // star ID to its orbits, in order
	for _, o := range d.Orbits {
		orbitsOf[o.StarID] = append(orbitsOf[o.StarID], o)
	}

	s := root.Derive(prng.TagPlanets)
	fromTemplate := map[int]*TemplatePlanet{} // by orbit ID
	var chosen, rest []int                    // orbit IDs
	random := 0                               // stars that aren't home stars
	for _, star := range d.Stars {
		if star.HomeStar {
			for _, o := range orbitsOf[star.ID] {
				if p := h.Orbits[o.OrbitNo-1].Planet; p != nil {
					fromTemplate[o.ID] = p
					chosen = append(chosen, o.ID)
				}
			}
			continue
		}
		random++
		var ids []int
		for _, o := range orbitsOf[star.ID] {
			ids = append(ids, o.ID)
		}
		s.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		chosen = append(chosen, ids[0])
		rest = append(rest, ids[1:]...)
	}
	more := random*9/2 - random
	s.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
	chosen = append(chosen, rest[:more]...)
	slices.Sort(chosen)

	templated := map[int]*TemplatePlanet{}
	for i, orbitID := range chosen {
		p := game.Planet{ID: i + 1, OrbitID: orbitID}
		if t := fromTemplate[orbitID]; t != nil {
			p.Habitable, p.Viability = t.Habitable, t.Viability
			templated[p.ID] = t
		}
		d.Planets = append(d.Planets, p)
	}
	return templated
}

// markHabitable makes some of the random planets (those not in templated)
// habitable and gives them a viability.
//
// The number of habitable planets is 125 for every 621 random planets, rounded
// to the nearest planet, split among the orbit numbers in habitableShares in
// proportion to their shares (see apportion). For each of those orbit numbers,
// in order, the planets in orbits with that number are shuffled, using the
// stream root.Derive(TagHabitable), and the first of them are made habitable.
// If an orbit number has too few planets, all of them are made habitable, and
// the shortfall is made up from the other planets in those orbit numbers,
// shuffled together.
//
// A habitable planet's viability is game.MinViability..game.MaxViability on a
// bell curve (see bellCurve) drawn from the stream root.Derive(TagViability,
// id), so it depends only on the planet's ID.
func markHabitable(root *prng.Stream, d *game.Data, templated map[int]*TemplatePlanet) {
	orbitNo := map[int]int{} // orbit ID to orbit number
	for _, o := range d.Orbits {
		orbitNo[o.ID] = o.OrbitNo
	}
	inOrbit := map[int][]int{} // orbit number to planet indexes, in planet order
	random := 0
	for i, p := range d.Planets {
		if templated[p.ID] == nil {
			random++
			inOrbit[orbitNo[p.OrbitID]] = append(inOrbit[orbitNo[p.OrbitID]], i)
		}
	}

	weights := make([]int, len(habitableShares))
	for i, share := range habitableShares {
		weights[i] = share.share
	}
	quotas := apportion((random*125+310)/621, weights)

	s := root.Derive(prng.TagHabitable)
	shuffle := func(xs []int) {
		s.Shuffle(len(xs), func(i, j int) { xs[i], xs[j] = xs[j], xs[i] })
	}
	habitable := func(indexes []int) {
		for _, i := range indexes {
			p := &d.Planets[i]
			p.Habitable = true
			p.Viability = bellCurve(root.Derive(prng.TagViability, prng.Key(p.ID)), game.MinViability, game.MaxViability)
		}
	}
	shortfall := 0
	var spare []int
	for i, share := range habitableShares {
		candidates := inOrbit[share.orbitNo]
		shuffle(candidates)
		n := min(quotas[i], len(candidates))
		habitable(candidates[:n])
		shortfall += quotas[i] - n
		spare = append(spare, candidates[n:]...)
	}
	shuffle(spare)
	habitable(spare[:min(shortfall, len(spare))])
}

// generateDeposits sets d.Deposits, with IDs from 1 in planet order.
//
// A planet in templated gets the deposits its template lists. The number of
// deposits on each of the other planets is 1 plus the sum of three 11-sided
// dice numbered 0 to 10, about 16 on average: the number of planets with each
// count is that distribution scaled to the number of random planets (see
// apportion), and the counts are shuffled and dealt to the planets in order.
// The types of their deposits are a list in the proportions of depositShares,
// shuffled and dealt to the deposits in order. Both shuffles use the stream
// root.Derive(TagDeposits).
//
// A random deposit's size and yield are drawn on a bell curve (see bellCurve)
// within the ranges for its type, size first, from the stream
// root.Derive(TagDepositAmounts, id), so they depend only on the deposit's ID
// and type.
func generateDeposits(root *prng.Stream, d *game.Data, templated map[int]*TemplatePlanet) {
	random := len(d.Planets) - len(templated)

	// ways[n-1] is the number of ways three dice can give a planet n deposits.
	ways := make([]int, game.MaxDepositsPerPlanet)
	for a := range 11 {
		for b := range 11 {
			for c := range 11 {
				ways[a+b+c]++
			}
		}
	}
	var counts []int
	total := 0
	for i, planets := range apportion(random, ways) {
		for range planets {
			counts = append(counts, i+1)
			total += i + 1
		}
	}
	s := root.Derive(prng.TagDeposits)
	s.Shuffle(len(counts), func(i, j int) { counts[i], counts[j] = counts[j], counts[i] })

	weights := make([]int, len(depositShares))
	for i, share := range depositShares {
		weights[i] = share.share
	}
	types := make([]game.DepositType, 0, total)
	for i, n := range apportion(total, weights) {
		for range n {
			types = append(types, depositShares[i].typ)
		}
	}
	s.Shuffle(len(types), func(i, j int) { types[i], types[j] = types[j], types[i] })

	for _, planet := range d.Planets {
		if t := templated[planet.ID]; t != nil {
			for i, td := range t.Deposits {
				d.Deposits = append(d.Deposits, game.Deposit{
					ID: len(d.Deposits) + 1, PlanetID: planet.ID, DepositNo: i + 1,
					Type: td.Type, Size: td.Size, Yield: td.Yield,
				})
			}
			continue
		}
		for depositNo := 1; depositNo <= counts[0]; depositNo++ {
			x := game.Deposit{ID: len(d.Deposits) + 1, PlanetID: planet.ID, DepositNo: depositNo, Type: types[0]}
			types = types[1:]
			amounts, ds := depositAmounts[x.Type], root.Derive(prng.TagDepositAmounts, prng.Key(x.ID))
			x.Size = bellCurve(ds, amounts.minSize, amounts.maxSize)
			x.Yield = bellCurve(ds, amounts.minYield, amounts.maxYield)
			d.Deposits = append(d.Deposits, x)
		}
		counts = counts[1:]
	}
}

// generateEmpires sets d.Races, d.Empires, and d.Colonies. For each home
// world, in planet ID order, it creates a race, whose ID is the home world's
// ID, and empiresPerHomeWorld empires of that race, named "Empire #ID", with
// no handle. Each empire gets the colonies in e on its home world, with no
// mining groups: an empire assembles those, claiming deposits for itself. Every
// colony starts with the default standard of living, death rate, and birth
// rate.
func generateEmpires(d *game.Data, templated map[int]*TemplatePlanet, e EmpireTemplate) {
	for _, planet := range d.Planets {
		if t := templated[planet.ID]; t == nil || !t.HomeWorld {
			continue
		}
		d.Races = append(d.Races, game.Race{ID: planet.ID, HomeWorldID: planet.ID})
		for range empiresPerHomeWorld {
			empire := game.Empire{ID: len(d.Empires) + 1, RaceID: planet.ID, HomeWorldID: planet.ID}
			empire.Name = fmt.Sprintf("Empire #%d", empire.ID)
			d.Empires = append(d.Empires, empire)
			for _, tc := range e.Colonies {
				colony := game.Colony{
					ID:            len(d.Colonies) + 1,
					EmpireID:      empire.ID,
					PlanetID:      planet.ID,
					No:            tc.No,
					Type:          tc.Type,
					TechLevel:     tc.TechLevel,
					Population:    tc.Population,
					Rations:       game.DefaultRations,
					SOL:           game.DefaultSOL,
					DeathRate:     game.DefaultDeathRate,
					BirthRate:     game.DefaultBirthRate,
					Operational:   slices.Clone(tc.Operational),
					Cargo:         slices.Clone(tc.Cargo),
					Depot:         slices.Clone(tc.Depot),
					FarmingGroups: cloneGroups(tc.FarmingGroups),
					FactoryGroups: cloneGroups(tc.FactoryGroups),
					MiningGroups:  []game.MiningGroup{},
				}
				d.Colonies = append(d.Colonies, colony)
			}
		}
	}
}

// cloneGroups returns a copy of groups that shares nothing with it.
func cloneGroups(groups []game.Group) []game.Group {
	out := make([]game.Group, len(groups))
	for i, g := range groups {
		out[i] = game.Group{No: g.No, Units: slices.Clone(g.Units)}
	}
	return out
}

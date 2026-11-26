// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/mdhender/ecvg/internal/cerrs"
)

const (
	ErrNoSuchStar  = cerrs.Error("no such star")
	ErrNotHomeStar = cerrs.Error("star can't be a home star")
	ErrAlreadyHome = cerrs.Error("already a home stellium")
)

// HomeStarChanges reports what DesignateHomeStar did.
type HomeStarChanges struct {
	StarID     int
	StelliumID int
	// HomeWorldID is the ID of the new home world planet.
	HomeWorldID int
	// Removed and Added count the orbits, planets, and deposits removed from
	// the star and added to it.
	Removed, Added struct{ Orbits, Planets, Deposits int }
}

// DesignateHomeStar makes the star with ID starID a home star, and its
// Stellium a home Stellium, then replaces everything around the star with the
// contents of t.
//
// The star must be the first star (SeqNo 1) of a Stellium with exactly
// HomeStarsPerStellium stars, and that Stellium must not already be a home
// Stellium.
//
// The star's orbits, the planets in them, and those planets' deposits are
// removed. New orbits, planets, and deposits are made from t, with IDs that
// continue past the largest ID of each kind before the removal, so IDs are
// never reused. (A later call can't fall back below an ID removed earlier:
// the records added by each call have the largest IDs at the time, and they
// belong to a home star, which is never replaced again.) They are appended, so every slice stays in ID order. A planet
// takes its habitability and viability from t; deposits are numbered from 1
// on each planet in the order t lists them.
//
// On error, d is unchanged.
func (d *Data) DesignateHomeStar(starID int, t HomeStarTemplate) (HomeStarChanges, error) {
	var c HomeStarChanges
	if err := t.Validate(); err != nil {
		return c, err
	}
	si := slices.IndexFunc(d.Stars, func(s Star) bool { return s.ID == starID })
	if si < 0 {
		return c, fmt.Errorf("star %d: %w", starID, ErrNoSuchStar)
	}
	star := d.Stars[si]
	li := slices.IndexFunc(d.Stellia, func(s Stellium) bool { return s.ID == star.StelliumID })
	if li < 0 {
		return c, fmt.Errorf("star %d: stellium %d: %w", starID, star.StelliumID, ErrNoSuchStar)
	}
	stellium := d.Stellia[li]
	if stellium.HomeStellium || star.HomeStar {
		return c, fmt.Errorf("star %d: stellium %d: %w", starID, stellium.ID, ErrAlreadyHome)
	}
	inStellium := 0
	for _, s := range d.Stars {
		if s.StelliumID == stellium.ID {
			inStellium++
		}
	}
	if inStellium != HomeStarsPerStellium {
		return c, fmt.Errorf("star %d: stellium %d has %d stars, want %d: %w",
			starID, stellium.ID, inStellium, HomeStarsPerStellium, ErrNotHomeStar)
	}
	if star.SeqNo != 1 {
		return c, fmt.Errorf("star %d is star %d of stellium %d, want its first star: %w",
			starID, star.SeqNo, stellium.ID, ErrNotHomeStar)
	}

	// The next ID of each kind comes from the largest before anything is removed.
	nextOrbit := maxID(d.Orbits, func(o Orbit) int { return o.ID }) + 1
	nextPlanet := maxID(d.Planets, func(p Planet) int { return p.ID }) + 1
	nextDeposit := maxID(d.Deposits, func(x Deposit) int { return x.ID }) + 1

	oldOrbits := map[int]bool{}
	for _, o := range d.Orbits {
		if o.StarID == starID {
			oldOrbits[o.ID] = true
		}
	}
	oldPlanets := map[int]bool{}
	for _, p := range d.Planets {
		if oldOrbits[p.OrbitID] {
			oldPlanets[p.ID] = true
		}
	}

	// Build the new slices without touching d, so an error leaves it as it was.
	orbits := slices.DeleteFunc(slices.Clone(d.Orbits), func(o Orbit) bool { return oldOrbits[o.ID] })
	planets := slices.DeleteFunc(slices.Clone(d.Planets), func(p Planet) bool { return oldPlanets[p.ID] })
	deposits := slices.DeleteFunc(slices.Clone(d.Deposits), func(x Deposit) bool { return oldPlanets[x.PlanetID] })
	c.Removed.Orbits = len(d.Orbits) - len(orbits)
	c.Removed.Planets = len(d.Planets) - len(planets)
	c.Removed.Deposits = len(d.Deposits) - len(deposits)

	for _, to := range t.Orbits {
		orbit := Orbit{ID: nextOrbit, StarID: starID, OrbitNo: to.OrbitNo}
		nextOrbit++
		orbits = append(orbits, orbit)
		c.Added.Orbits++
		if to.Planet == nil {
			continue
		}
		planet := Planet{ID: nextPlanet, OrbitID: orbit.ID, Habitable: to.Planet.Habitable, Viability: to.Planet.Viability}
		nextPlanet++
		planets = append(planets, planet)
		c.Added.Planets++
		if to.Planet.HomeWorld {
			c.HomeWorldID = planet.ID
		}
		for i, td := range to.Planet.Deposits {
			deposits = append(deposits, Deposit{
				ID:        nextDeposit,
				PlanetID:  planet.ID,
				DepositNo: i + 1,
				Type:      td.Type,
				Size:      td.Size,
				Yield:     td.Yield,
			})
			nextDeposit++
			c.Added.Deposits++
		}
	}

	d.Stellia = slices.Clone(d.Stellia)
	d.Stellia[li].HomeStellium = true
	d.Stars = slices.Clone(d.Stars)
	d.Stars[si].HomeStar = true
	d.Orbits, d.Planets, d.Deposits = orbits, planets, deposits
	c.StarID, c.StelliumID = starID, stellium.ID
	return c, nil
}

// maxID returns the largest ID in s, or 0 if s is empty.
func maxID[T any](s []T, id func(T) int) int {
	m := 0
	for _, v := range s {
		m = max(m, id(v))
	}
	return m
}

// LoadData reads a game's master data, Stellia, stars, orbits, planets, and
// deposits from dir.
func LoadData(dir string) (Data, error) {
	var d Data
	var err error
	if d.Game, err = Load(dir); err != nil {
		return Data{}, err
	}
	if d.Stellia, err = LoadStellia(dir); err != nil {
		return Data{}, err
	}
	if d.Stars, err = LoadStars(dir); err != nil {
		return Data{}, err
	}
	if d.Orbits, err = LoadOrbits(dir); err != nil {
		return Data{}, err
	}
	if d.Planets, err = LoadPlanets(dir); err != nil {
		return Data{}, err
	}
	if d.Deposits, err = LoadDeposits(dir); err != nil {
		return Data{}, err
	}
	return d, nil
}

// DesignateHomeStar loads the game in dir and its homestar-template.json,
// applies Data.DesignateHomeStar to the star with ID starID, and saves the
// Stellia, stars, orbits, planets, and deposits. It writes every file to a
// temporary file before renaming any of them into place, so an error before
// the renames leaves the game unchanged.
func DesignateHomeStar(dir string, starID int) (HomeStarChanges, error) {
	d, err := LoadData(dir)
	if err != nil {
		return HomeStarChanges{}, err
	}
	t, err := LoadHomeStarTemplate(dir)
	if err != nil {
		return HomeStarChanges{}, err
	}
	c, err := d.DesignateHomeStar(starID, t)
	if err != nil {
		return HomeStarChanges{}, err
	}
	err = replaceFiles(dir, []struct {
		name string
		v    any
	}{
		{StelliaFileName, d.Stellia},
		{StarsFileName, d.Stars},
		{OrbitsFileName, d.Orbits},
		{PlanetsFileName, d.Planets},
		{DepositsFileName, d.Deposits},
	})
	if err != nil {
		return HomeStarChanges{}, err
	}
	return c, nil
}

// replaceFiles writes each value as compact JSON to a temporary file in dir,
// then renames the temporary files over the named files. If anything fails
// before the renames, it removes the temporary files and changes nothing.
func replaceFiles(dir string, files []struct {
	name string
	v    any
}) error {
	var temps []string
	cleanup := func() {
		for _, path := range temps {
			_ = os.Remove(path)
		}
	}
	for _, file := range files {
		data, err := json.Marshal(file.v)
		if err != nil {
			cleanup()
			return fmt.Errorf("%s: %w", file.name, err)
		}
		fd, err := os.CreateTemp(dir, "."+file.name+".*")
		if err != nil {
			cleanup()
			return err
		}
		temps = append(temps, fd.Name())
		_, err = fd.Write(append(data, '\n'))
		if err == nil {
			err = fd.Sync()
		}
		if cerr := fd.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Chmod(fd.Name(), 0o644)
		}
		if err != nil {
			cleanup()
			return err
		}
	}
	for i, file := range files {
		if err := os.Rename(temps[i], filepath.Join(dir, file.name)); err != nil {
			cleanup()
			return err
		}
	}
	return nil
}

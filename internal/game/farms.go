// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// ErrFarmSite is matched by the errors of CanInstallFarm.
const ErrFarmSite = cerrs.Error("farm can't be installed here")

const (
	// FarmsPerViability is the number of FARM-1 units a planet can hold for
	// each point of its viability.
	FarmsPerViability = 100_000
	// MaxSurfaceFarmOrbit is the outermost orbit in which a surface colony
	// can farm with FARM-1 to FARM-5 units.
	MaxSurfaceFarmOrbit = 5
)

// FarmCap returns the most FARM-1 units the planet can hold. It is the total
// for all the empires on the planet, not a limit per empire.
func (p Planet) FarmCap() int {
	if !p.Habitable {
		return 0
	}
	return p.Viability * FarmsPerViability
}

// CanInstallFarm reports whether an entity of type t, at the planet p in
// orbit orbitNo, may add FARM units of tech level tl to a farming group. The
// error, if any, matches ErrFarmSite.
//
//   - FARM-1 needs an open-air surface colony on a habitable planet with a
//     viability above 0, in orbits 1..MaxSurfaceFarmOrbit. The planet's
//     FarmCap is a separate check.
//   - FARM-2 to FARM-5 need an orbital colony, in any orbit, or a surface
//     colony in orbits 1..MaxSurfaceFarmOrbit.
//   - FARM-6 to FARM-10 can be installed by a ship or any colony, anywhere.
func CanInstallFarm(tl int, t EntityType, orbitNo int, p Planet) error {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: FARM-%d "+format, append([]any{ErrFarmSite, tl}, args...)...)
	}
	inner := 1 <= orbitNo && orbitNo <= MaxSurfaceFarmOrbit
	switch {
	case tl < 1 || tl > 10:
		return bad("is not a farm")
	case tl >= 6:
		return nil
	case tl >= 2:
		switch {
		case t == CSHP:
			return bad("can't be installed in a ship")
		case t != CORB && !inner:
			return bad("needs a surface colony in orbits 1 to %d, not orbit %d", MaxSurfaceFarmOrbit, orbitNo)
		}
		return nil
	case t != COPN:
		return bad("needs an open-air surface colony")
	case !p.Habitable || p.Viability <= 0:
		return bad("needs a habitable planet with a viability above 0")
	case !inner:
		return bad("needs a planet in orbits 1 to %d, not orbit %d", MaxSurfaceFarmOrbit, orbitNo)
	}
	return nil
}

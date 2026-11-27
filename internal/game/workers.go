// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import "github.com/mdhender/ecvg/internal/units"

// AutoUSK returns the USK that qty assembled AUTO units of tech level tl
// stand in for: TL^2 each. Automation never replaces PRO.
func AutoUSK(tl, qty int) int {
	return tl * tl * qty
}

// Workers is a colony's worker (WRKR) cadre for a turn. The engine makes it,
// when resources are allocated, from the PRO, USK, and AUTO units that the
// empire's orders didn't allocate to something else, and then drains it as it
// staffs the farms, mines, factories, and laboratories. It lasts only for the turn; the
// population itself doesn't change.
type Workers struct {
	// PRO and USK are the cadre as it was made. USK includes AutoUSK, the
	// USK that the colony's AUTO units joined as.
	PRO     int `json:"pro"`
	USK     int `json:"usk"`
	AutoUSK int `json:"auto_usk,omitzero"`
	// IdlePRO and IdleUSK are what is left of the cadre: nothing has taken
	// them yet, or, once the turn is over, nothing did.
	IdlePRO int `json:"idle_pro"`
	IdleUSK int `json:"idle_usk"`
}

// NewWorkers returns the worker cadre of the colony c, with none of it taken:
// every PRO and USK that no drafted cadre holds, less the PRO that operated
// its TRNS this turn (Transport.Crew), plus the USK its operational
// AUTO units, in the colony or its depot, stand in for.
func NewWorkers(c Colony) Workers {
	auto := 0
	for _, list := range [][]Unit{c.Operational, c.DepotOperational} {
		for _, u := range list {
			if u.Type == units.AUTO {
				auto += AutoUSK(u.TechLevel, u.Qty)
			}
		}
	}
	pro, usk := max(0, c.FreePRO()-c.Transport.Crew), c.FreeUSK()+auto
	return Workers{PRO: pro, USK: usk, AutoUSK: auto, IdlePRO: pro, IdleUSK: usk}
}

// Take staffs up to want units that each need pro PRO and usk USK, draining
// the cadre, and returns the number of units it staffed.
func (w *Workers) Take(want, pro, usk int) int {
	n := max(0, want)
	if pro > 0 {
		n = min(n, w.IdlePRO/pro)
	}
	if usk > 0 {
		n = min(n, w.IdleUSK/usk)
	}
	w.IdlePRO -= n * pro
	w.IdleUSK -= n * usk
	return n
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// AssembleTechLevel carries out an assemble tech-level order, given by the
// empire with ID empireID: it pays for the tech levels and starts the work
// that raises the entity to them (see game.Colony.StartTechLevel); the
// Assembly phase does the work, and the entity's tech level changes when the
// work is done. Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - game.ErrTechLevelInProgress: the entity is already raising its tech
//     level.
//   - game.ErrTechLevelNotHigher, game.ErrTechLevelTooHigh: the order would
//     not raise the entity's tech level, or would raise it past the highest.
//   - game.ErrNotEnoughResearch: the entity's RSCH, or the PRTO units the
//     order names, don't pay for every level. It is all or nothing.
func AssembleTechLevel(d *game.Data, empireID int, o orders.AssembleTechLevel) error {
	for i := range d.Colonies {
		if c := &d.Colonies[i]; c.ID == o.EntityID && c.EmpireID == empireID {
			return c.StartTechLevel(o.To, o.PRTO.TechLevel, o.PRTO.Qty, o.MaxCWRK)
		}
	}
	return fmt.Errorf("entity %d: %w", o.EntityID, ErrNotControlled)
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
)

// ErrNotDraftable is the error for an order to draft a unit that can't be
// drafted.
const ErrNotDraftable = cerrs.Error("that unit can't be drafted")

// DraftOrder is a draft order that was accepted and waits for the draft
// phase.
type DraftOrder struct {
	EntityID int
	Unit     string
	Qty      int
}

// checkDraft accepts or rejects an order of the empire with ID empireID to
// draft qty of unit from the entity with ID entityID. It fails with
// ErrNotDraftable if the unit can't be drafted and ErrNotControlled if the
// empire has no such entity. It changes nothing: the draft phase does that.
func checkDraft(d *game.Data, empireID, entityID int, unit string) error {
	if !game.Draftable(unit) {
		return fmt.Errorf("%s: %w", unit, ErrNotDraftable)
	}
	for i := range d.Colonies {
		if c := &d.Colonies[i]; c.ID == entityID && c.EmpireID == empireID {
			return nil
		}
	}
	return fmt.Errorf("entity %d: %w", entityID, ErrNotControlled)
}

// Draft is the draft phase: it carries out the accepted draft orders, in
// order. Each drafts as many as its entity can supply (see
// game.Colony.Draft), and the entity's Drafted records what was asked and
// what was drafted. The phase comes after production, so the people drafted
// worked this turn and a cadre is first of use in the next one. Unlike the
// other phases it runs in the setup turn too.
func Draft(d *game.Data, drafts []DraftOrder) {
	for i := range d.Colonies {
		d.Colonies[i].Drafted = nil
	}
	for _, o := range drafts {
		for i := range d.Colonies {
			if c := &d.Colonies[i]; c.ID == o.EntityID {
				n := c.Draft(o.Unit, o.Qty)
				c.Drafted = append(c.Drafted, game.Drafted{Unit: o.Unit, Asked: o.Qty, Drafted: n})
				break
			}
		}
	}
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
	"github.com/mdhender/ecvg/internal/units"
)

const (
	// ErrSameEntity is the error for a transfer from an entity to itself.
	ErrSameEntity = cerrs.Error("an entity can't transfer to itself")
	// ErrNoSuchEntity is the error for a transfer to an entity that is not
	// in the game.
	ErrNoSuchEntity = cerrs.Error("there is no entity with that ID")
	// ErrForeignTransfer is the error for a transfer to another empire's
	// entity, which needs orders that aren't built yet.
	ErrForeignTransfer = cerrs.Error("transfers to another empire's entity are not accepted yet")
	// ErrNotSameOrbit is the error for a transfer between entities that are
	// not at the same planet.
	ErrNotSameOrbit = cerrs.Error("the entities are not at the same orbit")
	// ErrResearchNeedsPrototype is the error for a transfer of RSCH.
	ErrResearchNeedsPrototype = cerrs.Error("RSCH is transferred as prototypes: assemble PRTO-n on EntityID from laboratory-group GroupNo")
)

// AssemblePrototype carries out an assemble PRTO order, given by the empire
// with ID empireID: a laboratory group of the entity puts the entity's RSCH
// into one prototype (see game.Colony.MakePrototype). It is done at once.
// Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - game.ErrNoLabGroup: the entity has no laboratory group with that
//     number, or the group has no assembled units.
//   - game.ErrPrototypeTechLevel: the group's units are of another tech level
//     than the prototype.
//   - game.ErrNotEnoughResearch: the entity has less RSCH than the prototype
//     holds.
func AssemblePrototype(d *game.Data, empireID int, o orders.AssemblePrototype) error {
	for i := range d.Colonies {
		if c := &d.Colonies[i]; c.ID == o.EntityID && c.EmpireID == empireID {
			return c.MakePrototype(o.GroupNo, o.TechLevel)
		}
	}
	return fmt.Errorf("entity %d: %w", o.EntityID, ErrNotControlled)
}

// Transfer carries out a transfer order, given by the empire with ID
// empireID: the TRNS of the entity From move units from its storage into the
// storage of the entity To (see game.Colony.Transfer). The Transfer stage
// (B3.6, Transfers) calls it for each order, in the order they are given, and
// it takes what there is.
// Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no entity From.
//   - ErrNoSuchEntity, ErrSameEntity, ErrForeignTransfer, ErrNotSameOrbit:
//     the entity To doesn't exist, is From, belongs to another empire, or is
//     at another planet.
//   - ErrResearchNeedsPrototype: the unit is RSCH.
//   - game.ErrNothingToTransfer, game.ErrNoTransport, game.ErrNoRoom: not a
//     single unit could be moved.
func Transfer(d *game.Data, empireID int, o orders.Transfer) error {
	var from, to *game.Colony
	for i := range d.Colonies {
		switch c := &d.Colonies[i]; c.ID {
		case o.From:
			if c.EmpireID == empireID {
				from = c
			}
		case o.To:
			to = c
		}
	}
	switch {
	case from == nil:
		return fmt.Errorf("entity %d: %w", o.From, ErrNotControlled)
	case o.From == o.To:
		return fmt.Errorf("entity %d: %w", o.To, ErrSameEntity)
	case to == nil:
		return fmt.Errorf("entity %d: %w", o.To, ErrNoSuchEntity)
	case to.EmpireID != empireID:
		return fmt.Errorf("entity %d: %w", o.To, ErrForeignTransfer)
	case to.PlanetID != from.PlanetID:
		return fmt.Errorf("entities %d and %d: %w", o.From, o.To, ErrNotSameOrbit)
	case o.Unit.Type == units.RSCH:
		return ErrResearchNeedsPrototype
	}
	_, err := from.Transfer(to, o.Unit.Type, o.Unit.TechLevel, o.Unit.Qty)
	return err
}

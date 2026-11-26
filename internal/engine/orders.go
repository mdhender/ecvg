// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine

import (
	"fmt"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

const (
	ErrNoSuchEmpire  = cerrs.Error("no such empire")
	ErrNotControlled = cerrs.Error("the empire does not control an entity with that ID")
)

// Rejection is an order that was not carried out.
type Rejection struct {
	EmpireID int
	// File is the name of the orders file, and Line and Text the number and
	// text of the line in it.
	File string
	Line int
	Text string
	// Err says why the order was rejected.
	Err error
}

// String formats the rejection as file:line: text: reason.
func (r Rejection) String() string {
	return fmt.Sprintf("%s:%d: %s: %v", r.File, r.Line, r.Text, r.Err)
}

// OrdersResult is the outcome of the orders phase.
type OrdersResult struct {
	// Accepted is the number of orders carried out.
	Accepted int
	// Rejected lists the orders that were not, in file and line order.
	Rejected []Rejection
}

// ApplyOrders is the orders phase: it carries out each empire's orders on d,
// file by file and line by line. An order that is not valid, or that its
// empire may not give, is rejected and the rest are still carried out. It
// fails with ErrNoSuchEmpire, before changing d, if a file is for an empire
// that is not in the game.
func ApplyOrders(d *game.Data, files []orders.File) (OrdersResult, error) {
	empires := make(map[int]bool, len(d.Empires))
	for _, e := range d.Empires {
		empires[e.ID] = true
	}
	for _, f := range files {
		if !empires[f.EmpireID] {
			return OrdersResult{}, fmt.Errorf("%s: empire %d: %w", f.Name, f.EmpireID, ErrNoSuchEmpire)
		}
	}
	var result OrdersResult
	for _, f := range files {
		for _, line := range f.Lines {
			err := line.Err
			if err == nil {
				err = applyOrder(d, f.EmpireID, line.Order)
			}
			if err != nil {
				result.Rejected = append(result.Rejected, Rejection{
					EmpireID: f.EmpireID, File: f.Name, Line: line.No, Text: line.Text, Err: err,
				})
				continue
			}
			result.Accepted++
		}
	}
	return result, nil
}

// applyOrder carries out one order given by the empire with ID empireID.
func applyOrder(d *game.Data, empireID int, order orders.Order) error {
	switch o := order.(type) {
	case orders.SetRations:
		return SetRations(d, empireID, o.EntityID, o.Percent)
	}
	return fmt.Errorf("%w: %T", orders.ErrUnknownOrder, order)
}

// SetRations sets the rations of the entity with ID entityID to percent
// percent, on the orders of the empire with ID empireID. It fails with
// ErrNotControlled if the empire has no such entity and with
// orders.ErrInvalidPercentage if percent is not in
// game.MinRations..game.MaxRations.
func SetRations(d *game.Data, empireID, entityID, percent int) error {
	if percent < game.MinRations || percent > game.MaxRations {
		return fmt.Errorf("%w: %d%%", orders.ErrInvalidPercentage, percent)
	}
	for i := range d.Colonies {
		if c := &d.Colonies[i]; c.ID == entityID && c.EmpireID == empireID {
			c.Rations = percent
			return nil
		}
	}
	return fmt.Errorf("entity %d: %w", entityID, ErrNotControlled)
}

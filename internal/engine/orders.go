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
	// Drafts lists the accepted draft orders, which the draft phase carries
	// out later in the turn.
	Drafts []DraftOrder
}

// ApplyOrders is the orders phase: it carries out each empire's orders on d,
// file by file and line by line. An order that is not valid, or that its
// empire may not give, is rejected and the rest are still carried out. Draft
// orders are checked here and carried out by the Draft phase. Transfers are
// carried out here, by the TRNS that each entity has when the phase starts. It
// replaces d.OrderResults with a record of every order, accepted or not. It
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
	d.OrderResults = nil
	for i := range d.Colonies {
		d.Colonies[i].StartTransport()
	}
	for _, f := range files {
		for _, line := range f.Lines {
			err := line.Err
			if err == nil {
				err = applyOrder(d, &result, f.EmpireID, line.Order)
			}
			record := game.OrderResult{EmpireID: f.EmpireID, Line: line.No, Text: line.Text, Accepted: err == nil}
			if err != nil {
				record.Reason = err.Error()
			}
			d.OrderResults = append(d.OrderResults, record)
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

// applyOrder carries out one order given by the empire with ID empireID. A
// draft or disband order is only checked and added to result.Drafts.
func applyOrder(d *game.Data, result *OrdersResult, empireID int, order orders.Order) error {
	switch o := order.(type) {
	case orders.SetRations:
		return SetRations(d, empireID, o.EntityID, o.Percent)
	case orders.Draft:
		if err := checkDraft(d, empireID, o.EntityID, o.Unit); err != nil {
			return err
		}
		result.Drafts = append(result.Drafts, DraftOrder{EntityID: o.EntityID, Unit: o.Unit, Qty: o.Qty})
		return nil
	case orders.Disband:
		if err := checkDraft(d, empireID, o.EntityID, o.Unit); err != nil {
			return err
		}
		result.Drafts = append(result.Drafts, DraftOrder{Disband: true, EntityID: o.EntityID, Unit: o.Unit, Qty: o.Qty})
		return nil
	case orders.AssembleGroup:
		return AssembleGroup(d, empireID, o)
	case orders.AssemblePrototype:
		return AssemblePrototype(d, empireID, o)
	case orders.Transfer:
		return Transfer(d, empireID, o)
	case orders.Setup:
		return Setup(d, empireID, o)
	case orders.AssembleTechLevel:
		return AssembleTechLevel(d, empireID, o)
	case orders.UnitWork:
		return UnitWork(d, empireID, o)
	case orders.UpdateGroup:
		return UpdateGroup(d, empireID, o)
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

const (
	// ErrNoSuchGroup is the error for an order that names a group its
	// entity doesn't have.
	ErrNoSuchGroup = cerrs.Error("the entity has no such group")
	// ErrNotRetooled is the error for an order that has a factory group
	// change what it makes before it has retooled.
	ErrNotRetooled   = cerrs.Error("the group must retool before it changes what it produces")
	ErrNoneRunning   = cerrs.Error("the group has no units running")
	ErrNoneShutDown  = cerrs.Error("the group has no units shut down")
	ErrOnlyFactories = cerrs.Error("only a factory group can retool or be told what to produce")
)

// UpdateGroup carries out an update order for a group, given by the empire
// with ID empireID. It is the manual's build change stage (B3.7).
//
// Retool has a factory group retool: it stops making what it made, and waits
// to be told what to produce. Produce tells it. A group must have retooled to
// change what it produces, except to change from one structural unit (STRN,
// STRL, STRS) to another (manual H3.6). The retooling itself,
// which waits for the group's work in process and costs MTL and NMTL, comes
// with manufacturing.
//
// Shutdown shuts down units of the group that are running, and restart
// restarts units that are shut down: the order's Qty, or its Percent of all
// the units of the group. A fraction of a unit is rounded up for a shutdown
// and down for a restart. The order takes what there is: the units running,
// or shut down, if they are fewer than it asks for. Units shut down ask for
// no staff or fuel when resources are allocated, and stay shut down until
// they are restarted.
//
// Nothing changes if it fails:
//
//   - ErrNotControlled: the empire has no such entity.
//   - ErrNoSuchGroup: the entity has no such group.
//   - ErrOnlyFactories: the order is retool or produce, and the group is
//     not a factory group.
//   - ErrNotRetooled: a produce order changes what the group makes, and the
//     group has not retooled.
//   - ErrNoneRunning, ErrNoneShutDown: the group has no units for a shutdown
//     or a restart to work on.
//   - orders.ErrInvalidOrder: the order has no action, or its quantity or
//     percentage is not valid.
func UpdateGroup(d *game.Data, empireID int, o orders.UpdateGroup) error {
	var c *game.Colony
	for i := range d.Colonies {
		if d.Colonies[i].ID == o.EntityID && d.Colonies[i].EmpireID == empireID {
			c = &d.Colonies[i]
		}
	}
	if c == nil {
		return fmt.Errorf("entity %d: %w", o.EntityID, ErrNotControlled)
	}
	var (
		factory  *game.Group
		shutDown *int
		groupUs  []game.GroupUnits
	)
	switch o.Kind {
	case game.KindFarming:
		for j := range c.FarmingGroups {
			if g := &c.FarmingGroups[j]; g.No == o.GroupNo {
				shutDown, groupUs = &g.ShutDown, g.Units
			}
		}
	case game.KindMining:
		for j := range c.MiningGroups {
			if g := &c.MiningGroups[j]; g.No == o.GroupNo {
				shutDown, groupUs = &g.ShutDown, g.Units
			}
		}
	case game.KindFactory:
		for j := range c.FactoryGroups {
			if g := &c.FactoryGroups[j]; g.No == o.GroupNo {
				factory, shutDown, groupUs = g, &g.ShutDown, g.Units
			}
		}
	case game.KindLaboratory:
		for j := range c.LaboratoryGroups {
			if g := &c.LaboratoryGroups[j]; g.No == o.GroupNo {
				shutDown, groupUs = &g.ShutDown, g.Units
			}
		}
	}
	if shutDown == nil {
		return fmt.Errorf("entity %d: %s group %d: %w", o.EntityID, o.Kind, o.GroupNo, ErrNoSuchGroup)
	}

	switch o.Action {
	case game.ActionRetool:
		if factory == nil {
			return ErrOnlyFactories
		}
		factory.Producing, factory.Retooled = game.Unit{}, true
		return nil
	case game.ActionProduce:
		if factory == nil {
			return ErrOnlyFactories
		}
		want := game.Unit{Type: o.Producing.Type, TechLevel: o.Producing.TechLevel}
		if def, _ := units.Lookup(want.Type); !def.FactoryMade() || !def.ValidTechLevel(want.TechLevel) {
			return fmt.Errorf("%w: %s: %s", orders.ErrInvalidUnit, want.Type, orders.NotFactoryMade(want.Type))
		}
		if want == factory.Producing {
			return nil
		}
		if !factory.Retooled && !(structural(want.Type) && structural(factory.Producing.Type)) {
			return fmt.Errorf("factory group %d: %w", o.GroupNo, ErrNotRetooled)
		}
		factory.Producing, factory.Retooled = want, false
		return nil
	case game.ActionShutDown, game.ActionRestart:
	default:
		return fmt.Errorf("%w: %q: want retool, produce, shutdown, or restart", orders.ErrInvalidOrder, o.Action)
	}

	// The number of units the order asks for. A fraction of a unit is
	// rounded against the empire: up for a shutdown, down for a restart.
	total := game.GroupQty(groupUs)
	down := min(max(0, *shutDown), total)
	n := o.Qty
	switch {
	case o.Qty < 0 || o.Percent < 0 || o.Percent > 100 || (o.Qty == 0) == (o.Percent == 0):
		return fmt.Errorf("%w: want a quantity, 1 or more, or a percentage from 1%% to 100%%", orders.ErrInvalidOrder)
	case o.Percent != 0 && o.Action == game.ActionShutDown:
		n = (total*o.Percent + 99) / 100
	case o.Percent != 0:
		n = total * o.Percent / 100
	}
	if o.Action == game.ActionShutDown {
		if total == down {
			return fmt.Errorf("%s group %d: %w", o.Kind, o.GroupNo, ErrNoneRunning)
		}
		*shutDown = down + min(n, total-down)
		return nil
	}
	if down == 0 {
		return fmt.Errorf("%s group %d: %w", o.Kind, o.GroupNo, ErrNoneShutDown)
	}
	*shutDown = down - min(n, down)
	return nil
}

// structural reports whether code is a structural unit, which a factory
// group changes between without retooling (manual H3.6).
func structural(code string) bool {
	return code == units.STRN || code == units.STRL || code == units.STRS
}

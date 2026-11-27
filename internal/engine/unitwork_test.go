// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/orders"
)

// shipyard returns the builders game with units for the unit orders: colony 1
// has 20 STRN in its depot, and the orbital colony 2 has 10 SDRV-1
// assembled and 4 in its cargo.
func shipyard() game.Data {
	d := builders()
	d.Colonies[0].Depot = append(d.Colonies[0].Depot, game.Unit{Type: "STRN", Qty: 20})
	d.Colonies[1].Operational = []game.Unit{{Type: "SDRV", TechLevel: 1, Qty: 10}}
	d.Colonies[1].Cargo = append(d.Colonies[1].Cargo, game.Unit{Type: "SDRV", TechLevel: 1, Qty: 4})
	return d
}

func TestUnitWorkOrders(t *testing.T) {
	type unit = game.Unit
	sdrv := func(qty int) unit { return unit{Type: "SDRV", TechLevel: 1, Qty: qty} }
	for _, tc := range []struct {
		name string
		text string // the orders; all but the last must be accepted
		err  error  // of the last
		// work is the work the orders start in colony 1 and colony 2.
		work1, work2 []game.Assembly
	}{
		{
			name:  "assemble with a cap",
			text:  "assemble 5 STRN on 1 using 20 CWKR",
			work1: []game.Assembly{{Unit: unit{Type: "STRN", Qty: 5}, MaxCWRK: 20}},
		},
		{name: "assembling a unit that is never assembled does nothing", text: "assemble 10 CNGD on 1"},
		{name: "disassembling a unit that is never assembled does nothing", text: "disassemble 10 TRNS-1 on 1 using 5 CWRK"},
		{name: "a no-op still needs the entity", text: "assemble 10 CNGD on 3", err: engine.ErrNotControlled},
		{name: "group units are assembled by the group orders", text: "assemble 10 FARM-10 on 1", err: engine.ErrGroupUnit},
		{name: "groups can't be disassembled", text: "disassemble 10 FARM-10 on 1", err: engine.ErrGroupUnit},
		{name: "the units must be in storage", text: "assemble 1 STRL on 1", err: engine.ErrNotEnoughUnits},
		{
			name:  "an order for more units than there are takes them all",
			text:  "assemble 21 STRN on 1",
			work1: []game.Assembly{{Unit: unit{Type: "STRN", Qty: 20}}},
		},
		{
			name: "units are claimed once",
			text: "assemble 15 STRN on 1\nscrap 5 STRN on 1\nassemble 1 STRN on 1",
			err:  engine.ErrNotEnoughUnits,
			// The scrap order is started in the orders phase, and the
			// assemble orders in the Assembly stage.
			work1: []game.Assembly{{Action: game.WorkScrap, Unit: unit{Type: "STRN", Qty: 5}}, {Unit: unit{Type: "STRN", Qty: 15}}},
		},
		{
			name:  "scrap claims units from the group orders",
			text:  "scrap 100 MINE-10 on 1\nassemble mining-group on 1 with 50 MINE-10 working 1",
			err:   engine.ErrNotEnoughUnits,
			work1: []game.Assembly{{Action: game.WorkScrap, Unit: unit{Type: "MINE", TechLevel: 10, Qty: 100}}},
		},
		{
			name:  "disassemble",
			text:  "disassemble 6 SDRV-1 on 2\ndisassemble 4 SDRV-1 on 2",
			work2: []game.Assembly{{Action: game.WorkDisassemble, Unit: sdrv(6)}, {Action: game.WorkDisassemble, Unit: sdrv(4)}},
		},
		{
			name:  "units are disassembled once",
			text:  "disassemble 10 SDRV-1 on 2\ndisassemble 1 SDRV-1 on 2",
			err:   engine.ErrNotEnoughAssembled,
			work2: []game.Assembly{{Action: game.WorkDisassemble, Unit: sdrv(10)}},
		},
		{
			name:  "an order to disassemble more units than are assembled takes them all",
			text:  "disassemble 6 SDRV-1 on 2\ndisassemble 20,000 SDRV-1 on 2",
			work2: []game.Assembly{{Action: game.WorkDisassemble, Unit: sdrv(6)}, {Action: game.WorkDisassemble, Unit: sdrv(4)}},
		},
		{name: "units in storage can't be disassembled", text: "disassemble 1 STRN on 1", err: engine.ErrNotEnoughAssembled},
		{
			name:  "scrap",
			text:  "scrap 4 SDRV-1 on 2 using 200 CWRK",
			work2: []game.Assembly{{Action: game.WorkScrap, Unit: sdrv(4), MaxCWRK: 200}},
		},
		{
			name:  "scrap never takes the units in use",
			text:  "scrap 5 SDRV-1 on 2",
			work2: []game.Assembly{{Action: game.WorkScrap, Unit: sdrv(4)}},
		},
		{name: "resources can't be scrapped", text: "scrap 10 FUEL on 1", err: engine.ErrNotScrappable},
		{name: "someone else's entity", text: "scrap 1 SDRV-1 on 3", err: engine.ErrNotControlled},
		{
			name:  "a group order's cap",
			text:  "assemble factory-group on 1 with 10 FACT-1 producing CNGD using 3 CWRK",
			work1: []game.Assembly{{Kind: game.KindFactory, GroupNo: 5, Unit: unit{Type: "FACT", TechLevel: 1, Qty: 10}, MaxCWRK: 3}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := shipyard()
			result, err := applyAll(&d, []orders.File{file(t, 1, tc.text)})
			if err != nil {
				t.Fatal(err)
			}
			if tc.err == nil && len(result.Rejected) != 0 {
				t.Fatalf("rejected %+v", result.Rejected)
			}
			if tc.err != nil && (len(result.Rejected) != 1 || !errors.Is(result.Rejected[0].Err, tc.err) || result.Rejected[0].Line != result.Accepted+1) {
				t.Fatalf("rejected %+v, want the last order rejected with %v", result.Rejected, tc.err)
			}
			if !reflect.DeepEqual(d.Colonies[0].Assembly, tc.work1) || !reflect.DeepEqual(d.Colonies[1].Assembly, tc.work2) {
				t.Errorf("work %+v and %+v, want %+v and %+v", d.Colonies[0].Assembly, d.Colonies[1].Assembly, tc.work1, tc.work2)
			}
		})
	}
}

// Scrap is done by the CWRK in the Assembly phase, like assembly.
func TestScrapInATurn(t *testing.T) {
	d := shipyard()
	// 1 of the 2 CWRK scraps 300 MU: 10 MINE-10 at 30 MU each. A MINE-10 is
	// built from 15 MTL and 15 NMTL, and 70% of 150 is 105.
	if _, err := engine.RunTurn(&d, []orders.File{file(t, 1, "scrap 60 MINE-10 on 1 using 1 CWRK")}); err != nil {
		t.Fatal(err)
	}
	c := d.Colonies[0]
	want := game.Assembly{Action: game.WorkScrap, Unit: game.Unit{Type: "MINE", TechLevel: 10, Qty: 50}, MaxCWRK: 1, Assembled: 10, MTL: 105, NMTL: 105}
	if c.Assembly[0] != want || c.Construction != (game.Construction{Pool: 1000, Used: 500}) {
		t.Errorf("work %+v, construction %+v", c.Assembly, c.Construction)
	}
	// An open-air colony keeps its resources in its depot.
	if c.Stored("MINE", 10) != 90 || count(c.Depot, "MTL") != 105 || count(c.Depot, "NMTL") != 105 {
		t.Errorf("cargo %+v, depot %+v", c.Cargo, c.Depot)
	}
}

func count(list []game.Unit, code string) int {
	n := 0
	for _, u := range list {
		if u.Type == code {
			n += u.Qty
		}
	}
	return n
}

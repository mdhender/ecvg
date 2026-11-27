# Turn Processing Review Fixes Plan

This plan documents the findings from reviewing the game engine turn-processing code (`cmd/ec`, `internal/engine`, and `internal/game`) and specifies concrete steps and tests to address each item.

## Implementation Order

Address findings in priority order so population/resource invariants are fixed before assembly/setup edge cases:

1. **Population & cadre invariants** (Findings 3, 2, 5)
2. **Resource & food reservation integrity** (Findings 1, 8)
3. **Planet limits & turn-stage consistency** (Findings 4, 6)
4. **CWRK pool efficiency & minor hardening** (Finding 7, Minor Notes)

---

## Findings & Action Steps

### 1. [High] `Rations` ignores `FOOD` reserved by setups and add-ons

- **Locations:**
  - `internal/engine/food.go`: `Rations` (lines 67–91)
  - `internal/game/setup.go`: `Colony.Supply` (lines 230–260), `Colony.completeSetup` (lines 464–468), `Colony.RunAddOn` (lines 577–580)
  - `internal/game/assembly.go`: `Colony.takeStored` (lines 362–365)
- **Problem:**
  Setups and add-ons reserve `FOOD` in `BOM` without removing it from `Cargo`. `Colony.Stock` and `Colony.Spend` subtract `c.Reserved(units.FOOD, 0)`, but `Rations` sums and deducts `FOOD` directly from `c.Cargo`. In enclosed entities (`CENC`, `CORB`, `CSHP`), or in `COPN` when `FOOD` is in `Cargo`, the population can eat reserved `FOOD`. Later, `completeSetup` and `RunAddOn` call `c.takeStored`, which ignores shortfalls from `takeUnits`, while `s.Entity.Cargo` or `s.Delivered` still gets the full reserved `FOOD`—duplicating `FOOD`.
- **Steps to Fix:**
  1. In `Rations` (`internal/engine/food.go`), compute `cargoFood` as before, then cap the `FOOD` available for consumption to unreserved stock:
     `food := min(cargoFood, c.Stock(FoodUnitType))` (equivalently `max(0, cargoFood - max(0, c.Reserved(FoodUnitType, 0) - depotFood))`).
  2. Verify `Rations` still deducts eaten `FOOD` only from `c.Cargo` (never from `c.Depot`), matching the rule that rations come from cargo only.
  3. Add regression tests in `internal/engine/food_test.go` (or `internal/engine/setup_test.go`):
     - Enclosed colony (`CENC`/`CSHP`) with `FOOD` in `Cargo` reserved by a `Setup` or `AddOn`: confirm `Rations` cannot eat reserved `FOOD`, starvation triggers if unreserved `FOOD` is insufficient, and `completeSetup` / `RunAddOn` delivers the exact reserved `FOOD` without duplication.
     - `COPN` colony with `FOOD` in both `Cargo` and `Depot` where part is reserved: confirm `Rations` only eats unreserved `FOOD` from `Cargo`.

---

### 2. [High] `RunAddOn` can drive `Population` and `Cadres` negative and duplicate people

- **Locations:**
  - `internal/game/setup.go`: `Colony.RunAddOn` (lines 557–586)
  - `internal/engine/food.go`: `PopulationChanges` (lines 97–111)
  - `internal/engine/lifesupport.go`: `LifeSupport` (lines 27–48)
- **Problem:**
  An `AddOn` reserves people (`UEM`, `USK`, `PRO`, `SLD`, `CWRK`, `SPY`) in `a.BOM` and delivers starting on the set up's second turn (`s.Turns >= 1`). If the helper colony loses population to starvation (`PopulationChanges`) or life support (`LifeSupport`) before delivery, `a.BOM` is not reduced. `RunAddOn` only clamps `n` via `c.loadLimit` and unconditionally subtracts `moved` from `c.Population` and `n` from `c.Cadres.CWRK` / `c.Cadres.SPY`, driving them negative while adding phantom people and cadres to `builder`.
- **Steps to Fix:**
  1. Extract a shared helper or add explicit availability clamping in `Colony.RunAddOn` (`internal/game/setup.go`) for `peopleOf(code)` entries before calling `c.loadLimit`:
     - Clamp `avail` to `c.Population.UEM` / `USK` / `PRO` / `SLD`, or for `CWRK` to `min(max(0, c.Cadres.CWRK), c.Population.PRO, c.Population.USK)` and for `SPY` to `min(max(0, c.Cadres.SPY), c.Population.PRO, c.Population.SLD)`.
     - Alternatively (or additionally), trim `a.BOM` down to surviving `Population`/`Cadres` and call `c.ReserveAddOn` / keep `a.Want` accurate so the un-delivered shortfall stays in `a.Want` (or is released for the helper/builder to re-supply). Be careful with `s.Pledged`: if `a.BOM` shrinks because people died, `a.Want` still reflects the pledge so either the helper re-reserves future population or `a.Waiting = "units"` until cancelled/re-supplied.
  2. Call `c.Cadres = c.Cadres.Fit(c.Population)` after removing population in `RunAddOn` if any cadre invariant could be affected.
  3. Add regression tests in `internal/engine/setup_test.go`:
     - Helper reserves `UEM`/`PRO`/`CWRK`/`SPY` in `AddOn`, loses part or all of them to starvation/life-support on turn 1, then runs `Setups` on turn 2: confirm `helper.Population` and `helper.Cadres` never go negative, `builder` receives at most the surviving people, and `AddOn` waits (`"units"`) rather than finishing early with phantom people.

---

### 3. [High] `PopulationChanges` does not refit `Cadres` before `Demographics` runs `Train()`

- **Locations:**
  - `internal/engine/food.go`: `PopulationChanges` (lines 97–111)
  - `internal/engine/demographics.go`: `Demographics` (lines 23–35)
  - `internal/game/cadres.go`: `Colony.Train` (lines 64–72), `Cadres.Fit` (lines 103–112)
- **Problem:**
  `PopulationChanges` reduces `c.Population` without calling `c.Cadres = c.Cadres.Fit(c.Population)`. In Stage 20, `Demographics` calls `c.Train()` before `c.Cadres.Fit(c.Population)`. `Train()` calculates `Graduated` using pre-starvation `c.Cadres.TRNE`, allowing dead trainees to graduate out of surviving `USK` and potentially taking `USK` away from surviving `CWRK` (violating `Fit`'s `SPY > CWRK > TRNE` priority).
- **Steps to Fix:**
  1. In `PopulationChanges` (`internal/engine/food.go`), call `c.Cadres = c.Cadres.Fit(c.Population)` immediately after subtracting `s` from `c.Population`.
  2. Add regression tests in `internal/engine/food_test.go` and `internal/engine/demographics_test.go`:
     - Colony with `CWRK` and `TRNE` suffers starvation in `PopulationChanges`: verify `c.Cadres` is refitted immediately (`SPY > CWRK > TRNE`), and subsequent `Demographics` only trains surviving `TRNE` without eroding `CWRK`.

---

### 4. [High] `AssembleGroup` ignores in-progress `Setups` when enforcing `FARM-1` cap

- **Locations:**
  - `internal/engine/assembly.go`: `AssembleGroup` (lines 177–193)
  - `internal/engine/setup.go`: `Setup` (lines 182–200), `allEntities` (lines 443–452)
- **Problem:**
  `Setup` checks `planet.FarmCap()` across `allEntities(d)` (including `c.Setups`), and `AssembleGroup` uses `allEntities(d)` for `depositClaimed`, but `AssembleGroup`'s `FARM-1` cap check only loops over `d.Colonies`. An empire can exceed `planet.FarmCap()` by combining an in-progress `Setup` with `assemble farming-group`.
- **Steps to Fix:**
  1. In `AssembleGroup` (`internal/engine/assembly.go`), replace `for _, other := range d.Colonies` (line 178) with `for _, other := range allEntities(d)`.
  2. Add a regression test in `internal/engine/setup_test.go` or `internal/engine/assembly_test.go`:
     - Start a `COPN` setup with `FARM-1` units filling `planet.FarmCap()`, then order `assemble farming-group ... FARM-1` on an existing colony on the same planet: verify rejection with `engine.ErrFarmCap`.

---

### 5. [Medium] `completeSetup` can leave `c.Cadres` invalid mid-turn, and `RunSetups` can double-count `CWRK`

- **Locations:**
  - `internal/game/setup.go`: `Colony.RunSetups` (lines 373–402), `Colony.completeSetup` (lines 432–474)
  - `internal/game/assembly.go`: `Colony.Disassemble` (lines 209–210)
- **Problem:**
  1. In `completeSetup`, if `s.BOM` lists `PRO` (or `USK`/`SLD`) before `CWRK` (or `SPY`) and population died since reservation, taking `PRO` first can leave insufficient `PRO` for `CWRK` (`k.CWRK += 0`). `completeSetup` updates `c.Population = left` without refitting `c.Cadres`, leaving `c.Cadres` larger than `c.Population` for Stages 4–17 (inflating `Disassemble` EMU, distorting `Draft`/`Disband`, and distorting `NewWorkers`).
  2. In `RunSetups`, `own` checks total `c.Cadres.CWRK` per setup without tracking how many `CWRK` earlier setups already used; if `c.Cadres.CWRK` shrunk below `c.Reserved(units.CWRK, 0)`, multiple setups can reuse the same `CWRK`, and line 402 sets `c.Construction.Setup` using `c.Reserved(units.CWRK, 0)` (not capped by `c.Cadres.CWRK`), causing `Disassemble` (`Pool: max(pool, setup)`) to grant phantom EMU.
- **Steps to Fix:**
  1. In `completeSetup` (`internal/game/setup.go`), call `c.Cadres = c.Cadres.Fit(c.Population)` after `c.Population = left` and decrementing `k.CWRK` / `k.SPY` (or handle `CWRK`/`SPY` entries in `s.BOM` before `PRO`/`USK`/`SLD` entries so reserved cadres transfer intact before un-cadred population). Note: if calling `Fit`, check whether `CWRK`/`SPY` priority over un-cadred `PRO` is preferred; processing cadres first in `completeSetup` + calling `Fit` as a safety guard ensures both correctness and invariant preservation.
  2. In `RunSetups` (`internal/game/setup.go`):
     - Track remaining `availOwn := max(0, c.Cadres.CWRK)` across the `jobs` loop, clamping `own` to `availOwn` and decrementing `availOwn -= own` (or clamping `reservedCWRK` across setups).
     - Cap the reserved-CWRK contribution to `c.Construction.Setup` at `min(c.Reserved(units.CWRK, 0), max(0, c.Cadres.CWRK)-shuttled-prior)` (or `usedOwn + shuttled + prior`, ensuring `Setup <= max(0, c.Cadres.CWRK)*EMUPerCWRK`).
  3. Add unit tests in `internal/engine/setup_test.go`:
     - Setup listing `PRO` before `CWRK` where some `PRO` die before completion: verify `c.Cadres` fits `c.Population` immediately after Stage 3 (`Setups`).
     - Multiple setups on one colony when `c.Cadres.CWRK` shrunk below `c.Reserved(units.CWRK, 0)`: verify setups do not exceed `c.Cadres.CWRK` total work and `Construction.Pool` does not exceed `c.Cadres.CWRK * EMUPerCWRK`.

---

### 6. [Medium] Newly completed entities from Stage 3 (`Setups`) lack `StartTransport()` for Stage 6 (`Transfers`)

- **Locations:**
  - `internal/engine/orders.go`: `ApplyOrders` (lines 183–197)
  - `internal/engine/setup.go`: `Setups` (lines 436–441)
  - `internal/engine/turn.go`: `RunTurn` (lines 43–66)
- **Problem:**
  `ApplyOrders` calls `StartTransport()` on `d.Colonies` before Stage 3 (`Setups`) appends completed entities to `d.Colonies`. Because `Transfer`, `UpdateGroup`, `AssembleGroup`/`UnitWork`, and `SetRations` are deferred to Stages 6, 7, 8, and 13, orders targeting the newly created entity ID succeed in those stages (unlike `Draft`, which is checked and rejected in `ApplyOrders`), yet `transfer ... from <newEntity>` fails with `ErrNoTransport` even if the entity has `TRNS`, because `StartTransport()` was never called on it.
- **Steps to Fix:**
  1. Decide and align the rule with the 1978 manual:
     - **Option A (allow orders on newly completed entities from Stage 3 onward):** Call `c.StartTransport()` on each newly completed entity in `Setups` (or when appending `done` to `d.Colonies`). Check whether `Draft` should remain rejected in `ApplyOrders` or match entity availability.
     - **Option B (entities completed in Stage 3 cannot receive orders until the next turn):** Check entity existence in `ApplyOrders` (or track pre-turn entity IDs) so staged orders targeting an entity that did not exist at turn start are rejected uniformly alongside `Draft`.
  2. Add a test in `internal/engine/setup_test.go` or `internal/engine/transfer_test.go` covering orders referencing a setup's entity ID in the turn it completes.

---

### 7. [Medium] `Colony.construct` spends `CWRK` EMU on `Assembly` jobs blocked by `"room"` or `"units"`

- **Locations:**
  - `internal/game/assembly.go`: `Colony.construct` (lines 256–273)
- **Problem:**
  When `have < n` (`Waiting = "units"`) or `fit < n` (`Waiting = "room"`), `construct` still pulls up to `workEMU(..., a.Unit.Qty) - a.EMU` from `left` on every turn until `a.EMU` covers the entire remaining `a.Unit.Qty`. This drains the `CWRK` pool away from subsequent jobs in `c.Assembly` that could make progress.
- **Steps to Fix:**
  1. Before adding `take` to `a.EMU` (or by bounding `maxUnits` before computing `take`), determine the maximum number of units `limit` that currently have both storage/operational availability (`have`) and room (`fit` in enclosed colonies).
  2. Cap `take` so `a.EMU` does not advance past `workEMU(a.Action, d, tl, limit)` (and when `limit == 0`, set `a.Waiting` to `"units"` or `"room"` without consuming `left`).
  3. Check existing tests in `internal/game/assembly_test.go` (`"an enclosed colony needs room for the assembled units"` and `"the units must be in storage"`, which currently assert partial leftover `a.EMU` when `left` exceeded what fit/existed on the first turn) to confirm whether partial-unit EMU toward a blocked next unit is desired or should also stop at `workEMU(..., limit)`, and add a test with a second queued job in `c.Assembly`.

---

### 8. [Medium] `CancelSetup` stores `Delivered` `MTL`/`NMTL` in `Cargo` instead of `Depot` for `CENC`

- **Locations:**
  - `internal/game/setup.go`: `Colony.CancelSetup` (lines 415–421)
  - `internal/game/capacity.go`: `Colony.StoreResource` (lines 108–114)
- **Problem:**
  When a setup is cancelled, `CancelSetup` routes `Delivered` units to `c.Depot` only if `c.HasDepot() && d.DepotAllowed()`. For an enclosed surface colony (`CENC`), `c.HasDepot()` is `false`, so `MTL` and `NMTL` delivered by an add-on are placed into enclosed `c.Cargo` rather than the external `c.Depot`.
- **Steps to Fix:**
  1. In `Colony.CancelSetup` (`internal/game/setup.go`), mirror the depot rule used in `StoreResource` and `Transfer`:
     `if (c.HasDepot() && d.DepotAllowed()) || (c.Type == CENC && (u.Type == units.MTL || u.Type == units.NMTL))` -> store in `c.Depot`; otherwise `c.Cargo`.
  2. Add a test case in `internal/engine/setup_test.go` where a `CENC` builder cancels a setup after an add-on delivered `MTL`/`NMTL`, confirming they land in `c.Depot`.

---

### Minor Hardening Notes

- **`ApplyOrders` input ordering (`internal/engine/orders.go` lines 171–216):** `ApplyOrders` relies on callers passing `files` sorted by `EmpireID` without duplicates (as `orders.LoadTurn` does). Sorting/deduplicating or validating order in `ApplyOrders` would guard against direct callers passing unsorted/duplicate slices.

---

## Verification

After implementing each fix and adding the corresponding table-driven tests:

```sh
go test ./...
go vet ./...
```

# AGENTS.md

This file provides guidance to coding agents when working in this repository.

## Project

`ecvg` (module `github.com/mdhender/ecvg`) is another attempt at implementing Empyrean Challenge from the 1978 rulebook.
It is at the very beginning: the root `ecvg` package (`version.go`, a `semver.Version`), the `ecgen` command in `cmd/ecgen` (the game generator), the `ec` command in `cmd/ec` (the engine), the `ecview` command in `cmd/ecview`, and internal packages.
The goal is to ship the MVP: the generator follows the original 1978 setup and does not try to build a fair, balanced world.

The commands use `peterbourgon/ff/v4`. Each loads env files for the environment named by `EC_ENV` (default `development`), then reads flags from the command line and from `EC_`-prefixed environment variables (`--seed` is `EC_SEED`).
Games live in `games/{code}` (git-ignored). `internal/game` holds the record types and reads and writes the files; all generation logic is in `cmd/ecgen`.

`ecgen create [--seed N] [--homestar-template FILE] [--empire-template FILE] games/{code}` generates a whole game in one step, into an existing folder that holds none of the game's files (all are written, or none).
Files that don't change from turn to turn go in the game's folder; the state as of a turn goes in `turns/{turn}` (`deposits.json`, `empires.json`, and `colonies.json`). A new game is at turn 0, so the generator makes `turns/0`, and `game.LoadData` reads the state from the folder of the turn in `game.json`.


- `game.json` holds the master data: the code, the two seeds as 16-digit hex strings, and `turn`, the number of the current turn.
- `stellia.json` holds 100 Stellia, placed by shuffling every point with x²+y²+z² ≤ 225 except the origin and taking the first 100; they get 5, 3×4, 10×3, 30×2, and 56×1 stars, in that order.
- `stars.json` holds the 163 stars, each with the `stellium_id` of its Stellium.
  The Stellia with exactly 3 stars are home Stellia (`homestellium`), and the first star (`seq_no` 1) of each is a home star (`homestar`), so a game has 10 home stars.
- `orbits.json` holds 10 orbits per star, each with the `star_id` of its star.
- `planets.json` holds the planets, each with the `orbit_id` of its orbit, at most one per orbit.
  A home star's planets come from the home star template; its home world is the planet in orbit 3.
  The other 153 stars share 688 planets (4.5 per star, rounded down): every star gets one planet in a random orbit, then the rest go into random empty orbits.
  138 of those planets (125 per 621) are `habitable`, all in orbits 2..6, split 25:50:45:20:15; within each orbit number they are picked at random.
  A planet's `viability` is 0 if it is not habitable, otherwise 1..25 on a three-dice bell curve.
- `turns/0/deposits.json` holds the deposits, each with the `planet_id` of its planet, a `deposit_no`, and a `type` (FUEL, MTL, NMTL, or GOLD).
  A home star's deposits come from the template.
  Deposits per random planet follow 3d11 + 1 (about 16 on average): that distribution is scaled to the 688 planets and shuffled across them; the types are a 34:32:32:2 FUEL:MTL:NMTL:GOLD list shuffled across the deposits.
  Each random deposit has a `size` (usable ore) and `yield` (percentage of extracted ore that refines into a usable resource), each the sum of three dice spanning its range: size 1,000,000..20,000,000 for GOLD and 1,000,000..99,000,000 for the others; yield 1..12 GOLD, 1..22 FUEL, 1..28 MTL and NMTL.
- `races.json` holds one race per home world (`id` and `homeworld_id` are both the home world's planet ID).
- `turns/0/empires.json` holds 15 empires per home world, 150 in all (`id`, `handle`, `name`, `race_id`, `homeworld_id`), all of that home world's race.
  The name is `Empire #{id}` and the `handle` (8 to 12 lowercase ASCII letters, the player) is empty until an empire is given to a player.
- `turns/0/colonies.json` holds each empire's colonies, copied from the empire template onto its home world, with `empire_id` and `planet_id`, and the default `sol`, `drt`, and `brt`.

The templates are embedded from `cmd/ecgen/homestar-template.json` and `cmd/ecgen/empire-template.json`; the two template flags name a file to use instead. Loading either rejects unknown members and invalid values.
The home star template has 10 `orbits`, each with its `orbit_no`, an optional `note` for the game master (ignored), and a `planet` (`homeworld`, `habitable`, `viability`, `deposits`; a missing boolean or number is false or 0) or `null` for an empty orbit.
The home world is always the planet in orbit 3, marked `homeworld`, habitable with viability 25; no other planet may be marked.
The default is an Earth-like system with a planet in all 10 orbits (orbit 5 is an asteroid belt) and 84 deposits.
The empire template has `colonies`, each with `no`, `type` (an entity type: `CSHP`, `COPN`, `CENC`, or `CORB`), `tech_level`, `population` (`uem`, `usk`, `pro`, `sld`), `operational` and `cargo` units (`type`, optional `tech_level`, `qty`), and optional `farming_groups` and `factory_groups` (`no` and `units`, each `qty` and `tech_level`). Its default numbers come from the sample turn 1 report in the 1978 manual.
Empires start with no mining groups, so a template can't list any (`mining_groups` is rejected), and the default has no factory groups either: those units are in cargo, and empires must assemble the groups, claiming deposits for themselves.

Record IDs of every kind run from 1 with no gaps. JSON uses `encoding/json/v2`.
`ecview --store games/{code} stellia` reads a game and writes a self-contained HTML page (from `internal/view`, no network assets) with a rotatable 3D map and a table of the Stellia (home Stellia glow and are always labelled on the map, and are starred in a sortable Home column), to a temporary file or `--output`, then opens it in the default browser unless `--no-open` is given; `--store` is also `EC_STORE`.
`ecview --store games/{code} empires` does the same for a page listing the empires for the game master (ID, name, handle, race, home world and its location, population), which can be searched, filtered by race, and sorted; selecting an empire shows its colonies: population (units, people, and the FOOD, mass, space, and life support it takes), operational units, cargo, and farming, mining, and factory groups.

The plan:

- **Engine:** a Go command line tool to run a turn and generate reports for a game made by the generator.

## Commands

Go is 1.27 (`go.mod` says `go 1.27.4`).

```sh
go build ./...
go test ./...
go test ./internal/dotenv -run TestLoadPrecedence   # a single test
go vet ./...
go run ./cmd/ecgen create games/{code}          # the folder must already exist
go run ./cmd/ecgen create --homestar-template h.json --empire-template e.json games/{code}
go run ./cmd/ec turn report games/{code}            # write the current turn's reports
go run ./cmd/ec turn run games/{code}               # process the current turn
go run ./cmd/ec turn show games/{code}              # current turn and turn folders
go run ./cmd/ec turn rollback --confirm games/{code}   # undo the last run
go run ./cmd/ecview --store games/{code} stellia   # map of the Stellia in the browser
go run ./cmd/ecview --store games/{code} empires   # the empires and their colonies
```

## Code conventions

- Every Go file starts with the copyright and license header:
  ```go
  // Copyright (c) 2026 Michael D Henderson.
  // SPDX-License-Identifier: AGPL-3.0-or-later
  ```
- Sentinel errors are untyped constants of `cerrs.Error` (`internal/cerrs`), e.g. `const ErrX = cerrs.Error("x")`, and are matched with `errors.Is`.
- Tests are black-box (`package foo_test`) and table-driven with `t.Run`. Tests of a command (`package main`) are in the package, since a command can't be imported.
- `internal/dotenv` loads environment files for one of the environments `development`, `test`, `production`, or `agent`.
  `agent` is reserved for the coding agent's local work: use it when you need an environment.
  Precedence, highest first: real process environment, `.env.{env}.local`, `.env.local`, `.env.{env}`, `.env`.
  `.env.local` is loaded in `test` too, which is deliberately incompatible with bkeepers/dotenv.
  `*.local` env files hold secrets and are git-ignored; `.env` and `.env.{env}` are shared and never hold secrets.

## Data model (planned)

All coordinates are integers. Each level extends its parent's coordinates, and fields that don't apply are zero.

| Thing    | Coordinates                                          | Count                                 |
| -------- | ---------------------------------------------------- | ------------------------------------- |
| Stellium | `(x, y, z)`                                          | 100 per game, within 15 units of the origin, never at `(0,0,0)` |
| Star     | `(x, y, z, seq_no)`                                  | 1..5 per Stellium                     |
| Planet   | `(x, y, z, seq_no, orbit_no)`                        | 1..10 per Star                        |
| Entity   | `(x, y, z, seq_no, orbit_no, ox, oy, oz)`            | ship or colony                        |

The plural of Stellium is Stellia.
A ship orbiting a Stellium is `(x, y, z, 0, 0, 0, 0, 0)`; one orbiting a Star is `(x, y, z, seq_no, 0, 0, 0, 0)`; an open-air colony is `(x, y, z, seq_no, orbit_no, 0, 0, 0)`.
`ox, oy, oz` are small positive integers giving a position relative to the planet.

- **Planet:** habitability (bool; false forbids open-air colonies), viability (0..25, caps open-air farms), 1..31 deposits, each of type GOLD, FUEL, MTL (metallics), or NMTL (non-metallics).
- **Entity:** one of four types: CSHP (ship), COPN (open-air surface colony), CENC (enclosed surface colony), or CORB (enclosed orbital colony). It holds Population and Inventory units, both transferable between entities. All but COPN are enclosed. Every entity has an SOL (standard of living, default 1.0), a DRT (death rate, default 0.625%), and a BRT (birth rate, default 0.625%); the rates are stored as fractions (`"drt": 0.00625`).
- **Population:** four types of unit: UEM (unemployables), USK (unskilled workers), PRO (professionals), and SLD (soldiers). A unit is a block of 100 people with a mass of 1 MU; on a ship or in an enclosed colony it also takes 1 VU of space and 1 LU (life support unit). Each unit initially eats 0.25 FOOD per turn (`internal/game/population.go` rounds an entity's total up to a whole FOOD). Population quantities everywhere, including the empire template, are units, not people.
- **Cadres (not yet implemented):** CNW (construction workers) and SPY (spies) are population trained for a specific purpose. They must be assembled via orders, so they wait until orders are implemented; nothing starts with them.
- **Running a turn:** `ec turn run games/{code}` loads the state as of the current turn N from `turns/N`, runs the phases (`engine.RunTurn`), writes the new state to a new `turns/N+1` (built under a temporary name and renamed), and then replaces `game.json` to make N+1 the current turn. `turns/N` is never changed, and it is an error if `turns/N+1` already exists. Orders are not implemented yet.
- **Turn folders:** `turns/N` holds everything about turn N: the state as of its start (which is the result of turn N-1), `reports/`, the reports made from that state and sent to players, and `orders/`, the orders the players return for turn N. So state(N) + orders(N) -> state(N+1), then reports(N+1). Turn 0 holds the setup state.
- **Reports:** `ec turn report games/{code}` writes a plain text report per empire as of the current turn to `turns/N/reports/empire-{id}.txt` (`internal/report`), replacing any already there: the empire, each colony (population, units, cargo, groups), and the home world's deposits.
- **Show and rollback:** `ec turn show games/{code}` prints the current turn and each turn folder's report and order counts. `ec turn rollback --confirm games/{code}` undoes the last run (without `--confirm` it only says what it would delete, and fails): it makes N-1 current in `game.json` and deletes `turns/N` (state, reports, and orders); it fails at turn 0.
- **Turns:** the first turn of a game is turn 0, the players' setup turn, in which demographic changes don't apply. A year is 4 turns, and a turn whose number is greater than 0 and divides by 4 is an end-of-year turn. A turn is processed in phases, in `internal/engine`. So far there is one, demographic changes (`engine.Demographics`), run by `engine.RunTurn`: (1) if UEM is strictly more than 30% of an entity's population, 2% of UEM (rounded up) become USK; then (2) if the prior turn was an end-of-year turn (prior turn > 0 and divides by 4, so this happens in turns 5, 9, 13, ...), 5% of SLD (rounded up) become PRO.
- **Units:** manufactured or processed items with mass (MU) and volume (VU). Volume has assembled and unassembled values; most units are bigger assembled.
- **Production:** farming makes FOOD; mining extracts deposits into GOLD, FUEL, MTL, NMTL; manufacturing turns resource units into manufactured units (CNGD, SDRV, ...).

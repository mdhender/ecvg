# AGENTS.md

This file provides guidance to coding agents when working in this repository.

## Project

`ecvg` (module `github.com/mdhender/ecvg`) is another attempt at implementing Empyrean Challenge from the 1978 rulebook.
It is at the very beginning: the root `ecvg` package (`version.go`, a `semver.Version`), the `ec` command in `cmd/ec`, the `ecview` command in `cmd/ecview`, and internal packages.

`ec` uses `peterbourgon/ff/v4`. It loads env files for the environment named by `EC_ENV` (default `development`), then reads flags from the command line and from `EC_`-prefixed environment variables (`--seed` is `EC_SEED`).
Games live in `games/{code}` (git-ignored); `game.json` holds a game's master data, with its two seeds as 16-digit hex strings, `stellia.json` holds its Stellia, `stars.json` holds their stars, each with the `stellium_id` of its Stellium, `orbits.json` holds 10 orbits per star, each with the `star_id` of its star, and `planets.json` holds 621 planets (4.5 per star on average), each with the `orbit_id` of its orbit.
Every star gets one planet in a random orbit, then the rest go into random empty orbits, at most one planet per orbit.
Exactly 125 planets are `habitable`, all in orbits 2..6, split 25:50:45:20:15 (20, 41, 36, 16, 12 planets); within each orbit number the habitable planets are picked at random.
A planet's `viability` is 0 if it is not habitable, otherwise 1..25 on a three-dice bell curve.
`deposits.json` holds 9,936 deposits (16 per planet on average), each with the `planet_id` of its planet and a `type`: exactly 3,378 FUEL, 3,180 MTL, 3,180 NMTL, and 198 GOLD.
Deposits per planet come from a fixed bell-curve list (3d11 + 1, scaled to 621 planets and kept symmetric about 16), shuffled across the planets; the types are a fixed list shuffled across the deposits.
Each deposit also has a random `size` (usable ore) and `yield` (percentage of extracted ore that refines into a usable resource), each the sum of three dice spanning its range: size 1,000,000..20,000,000 for GOLD and 1,000,000..99,000,000 for the others; yield 1..12 GOLD, 1..22 FUEL, 1..28 MTL and NMTL.
Home Stellia are the Stellia with exactly 2 stars, so every game has 15 of them, and a game has at most 15 species.
Each home Stellium has a single home star, and each home star has a single home world (home planet).
`ec game homestar-template games/{code}` writes `homestar-template.json`, the template for every home star: 10 `orbits`, each with its `orbit_no`, an optional `note` for the game master (ignored by the game), and a `planet` (`homeworld`, `habitable`, `viability`, `deposits`; a missing boolean or number is false or 0) or `null` for an empty orbit.
The home world is always the planet in orbit 3, marked `homeworld`, habitable with viability 25; no other planet may be marked.
The default, edited by the game master, is an Earth-like system with a planet in all 10 orbits (orbit 5 is an asteroid belt) and 84 deposits; it lives in `internal/game/homestar-template.json`, embedded and written out byte for byte, so an edited template can be copied back to become the new default.
Loading it rejects unknown members and anything outside the planet rules; it is indented, because people edit it by hand, and never overwritten.
Stellia have a `homestellium` flag and stars a `homestar` flag, both false until the game master designates a home star.
`ec game designate-home --star {id} games/{code}` does that: the star must be the first star (`seq_no` 1) of a two-star Stellium that is not already a home Stellium.
It flags the star and its Stellium, deletes the star's orbits, their planets, and those planets' deposits, and creates new ones from the game's `homestar-template.json`; the home world is the new planet in orbit 3.
New records get IDs past the largest ID of their kind, so IDs are never reused and every file stays in ID order with gaps; all five changed files are written to temporary files and then renamed into place.
The counts above (orbits per star aside) describe a game as created; designating home stars changes them.
`ec game create` places the 100 Stellia by shuffling every point with x²+y²+z² ≤ 225 except the origin and taking the first 100; they get 5, 4, then 8×3, 15×2, and 75×1 stars.
JSON uses `encoding/json/v2`.
`ecview --store games/{code} stellia` reads a game and writes a self-contained HTML page (from `internal/view`, no network assets) with a rotatable 3D map and a table of the Stellia (home Stellia glow and are always labelled on the map, and are starred in a sortable Home column), to a temporary file or `--output`, then opens it in the default browser unless `--no-open` is given; `--store` is also `EC_STORE`.

The plan:

- **Engine:** a Go command line tool to initialize a new game, create empires, run a turn, and generate reports.

## Commands

Go is 1.27 (`go.mod` says `go 1.27.4`).

```sh
go build ./...
go test ./...
go test ./internal/dotenv -run TestLoadPrecedence   # a single test
go vet ./...
go run ./cmd/ec game create games/{code}       # the folder must already exist
go run ./cmd/ec game homestar-template games/{code}    # then edit homestar-template.json
go run ./cmd/ec game designate-home --star {id} games/{code}
go run ./cmd/ecview --store games/{code} stellia   # map of the Stellia in the browser
```

## Code conventions

- Every Go file starts with the copyright and license header:
  ```go
  // Copyright (c) 2026 Michael D Henderson.
  // SPDX-License-Identifier: AGPL-3.0-or-later
  ```
- Sentinel errors are untyped constants of `cerrs.Error` (`internal/cerrs`), e.g. `const ErrX = cerrs.Error("x")`, and are matched with `errors.Is`.
- Tests are black-box (`package foo_test`) and table-driven with `t.Run`.
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
- **Entity:** a ship, open-air colony, enclosed colony, or orbital colony. It holds Population and Inventory units, both transferable between entities.
- **Population:** groups USK, PRO, SLD, UEM; cadres CNW and SPY are population trained for a specific purpose.
- **Units:** manufactured or processed items with mass (MU) and volume (VU). Volume has assembled and unassembled values; most units are bigger assembled.
- **Production:** farming makes FOOD; mining extracts deposits into GOLD, FUEL, MTL, NMTL; manufacturing turns resource units into manufactured units (CNGD, SDRV, ...).

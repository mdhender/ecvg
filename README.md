# ecvg

An implementation of Empyrean Challenge, from the 1978 rulebook. It is alpha software: the file formats change without migration.

There are three commands:

| Command  | What it does                                              |
| -------- | --------------------------------------------------------- |
| `ecgen`  | Generates a new game.                                     |
| `ec`     | The engine: writes reports and runs turns.                |
| `ecview` | Shows a game in the browser, for the game master.         |

Run them with `go run ./cmd/{name}`. Every command takes `--help`.

## Which turn is this? (read this first)

The confusing part of any turn-based game is that "the end of turn 3" and "the start of turn 4" are the same moment. This project names that moment only one way:

> **The state of turn N is the state at the *start* of turn N.**

There is no "end of turn" state anywhere in the files or the code. Everything about turn N lives in the folder `turns/N`:

| In `turns/N` | What it is                                                             |
| ------------ | ---------------------------------------------------------------------- |
| state files  | The game as it stands at the start of turn N.                          |
| `reports/`   | Reports made *from* that state. They tell players what they have now.  |
| `orders/`    | The orders players send back *after reading* those reports.            |

Running turn N means:

```
state(N) + orders(N)  ->  state(N+1)
```

So the cycle, for every turn, is always the same four steps:

```
        turns/N                              turns/N+1
  +------------------+                 +------------------+
  | state(N)         |                 | state(N+1)       |
  |   |              |    run turn N   |   |              |
  |   v  report      |  ------------>  |   v  report      |
  | reports(N)       |                 | reports(N+1)     |
  |   |  (players)   |                 |                  |
  |   v              |                 |                  |
  | orders(N)        |                 |                  |
  +------------------+                 +------------------+
```

Things that follow from this, which are easy to get backwards:

- **A report numbered N shows the result of turn N-1.** The turn 1 report is what players see after turn 0 has been run. There is no separate "end of turn 0 report": it *is* the turn 1 report.
- **Orders numbered N are written while looking at report N**, and are consumed by running turn N.
- **The game's current turn** (`turn` in `game.json`) is the turn that has not been run yet. Its folder holds the newest state; its orders are the ones being collected.
- **Turn 0 is the setup turn.** `turns/0` holds the state the generator made. Players get the turn 0 report, send setup orders, and running turn 0 produces turn 1. Only orders are carried out in turn 0: nobody eats, starves, or changes job.
- **Running a turn never changes `turns/N`.** It only reads it and creates `turns/N+1`.

## Running a game

Create the game. The folder must exist and be empty of game files; its name is the game's code.

```sh
mkdir -p games/alpha
go run ./cmd/ecgen create games/alpha
```

`ecgen create` also takes `--seed`, `--homestar-template`, and `--empire-template`. The game starts at turn 0.

Then, for every turn, starting with turn 0:

```sh
# 1. Write the reports for the current turn and send them to the players.
go run ./cmd/ec turn report games/alpha

# 2. Collect the players' orders into turns/N/orders/, one file per empire,
#    named like the empire's report: empire-001.txt. See "Orders" below.

# 3. Run the turn. This creates turns/N+1 and makes it the current turn.
go run ./cmd/ec turn run games/alpha
```

`run` lists every order it rejected, as `file:line: order: reason`, and still runs the turn:

```
rejected empire-001.txt:3: set rations for 16 to 50%: entity 16: the empire does not control an entity with that ID
orders from 2 empires: 4 accepted, 1 rejected
ran turn 0 of game alpha; the state as of turn 1 is in games/alpha/turns/1
```

Every order and what became of it is saved in `turns/N+1/order-results.json`, and each empire's next report lists its own orders as accepted or rejected, with the reason. If a rejection was a typo, roll back, fix the file, and run again. A file in the orders folder that is not named for an empire in the game stops the run before anything changes.

Then go back to step 1, which now reports on the new turn.

To see where a game is:

```sh
go run ./cmd/ec turn show games/alpha
```

```
game alpha is at turn 1
turn   0:  150 reports,    0 orders
turn   1:  150 reports,    0 orders  <- current
```

### Undoing a turn

If a turn was run with bad orders, roll it back, fix the orders, and run it again:

```sh
go run ./cmd/ec turn rollback games/alpha             # says what it would delete; changes nothing
go run ./cmd/ec turn rollback --confirm games/alpha   # does it
```

If the current turn is N, rollback makes N-1 the current turn and **deletes `turns/N`**: its state, its reports, and any orders already collected for turn N. The orders for turn N-1 are kept. It can't go back past turn 0.

### Looking at a game

```sh
go run ./cmd/ecview --store games/alpha stellia   # 3D map of the Stellia
go run ./cmd/ecview --store games/alpha empires   # the empires and their colonies, as of the current turn
```

## Orders

An orders file is plain text, UTF-8, with one order per line. An order can't be split across lines, and a line can't hold two orders.

```
# Orders for Empire 1, turn 0.
set rations for 1 to 75%      // tighten belts
SET   RATIONS   FOR 1 TO 75%  # the same order: case and spacing don't matter
```

- Keywords ignore case. Any amount of white space separates the words.
- A comment starts with `#` or `//` and runs to the end of the line. It must start a word: `set rations#` is the two words `set` and `rations#`, not a comment. Neither starts a comment inside quoted text.
- Blank lines are ignored.

| Order | Effect |
| ----- | ------ |
| `set rations for EntityID to PercentageAmount` | Sets the rations of an entity (a ship or colony) the empire controls. EntityID is the "Entity ID" in the empire's report. PercentageAmount is a whole number from `1%` to `500%`; the `%` is required and must be joined to the number (`50%`, not `50 %`). A new entity is at 100%. If an entity gets more than one, the last wins. |
| `draft Quantity UnitOrCadre from EntityID` | Drafts people of an entity into soldiers or a cadre. Quantity is a whole number, with or without commas (`3600` or `3,600`). UnitOrCadre is `SLD` (1 USK each, who become soldiers), `CWRK` (1 PRO and 1 USK each), `TRNE` (1 USK each, and 1 PRO for every 100 or part of 100), or `SPY` (1 PRO and 1 SLD each). If the entity can't supply them all, it drafts as many as it can. Drafts happen after the turn's production, so the people drafted work that turn and the cadre is ready the next. People in a cadre no longer staff farms and mines. Example: `draft 5,000 CWRK from 16`. |
| `set idle for GroupKind group GroupNo of EntityID to PercentageAmount` | Orders part of a group to idle, so that it asks for no staff or FUEL. GroupKind is `farming`, `mining`, or `factory`, and GroupNo is the group's number in the report. PercentageAmount is `0%` to `100%` of the group's units, rounded up to a whole unit; `0%` puts the whole group back to work. It stands until you change it. Example: `set idle for farming group 1 of 16 to 94%`. |

Each turn after turn 0, the farms in a colony's farming groups make FOOD before anyone eats. FARM-2 to FARM-10 make 5 x their tech level in FOOD each turn. A FARM-1 makes 100 FOOD, but only after 4 turns of work in a row; if it misses a turn, the crop it had begun is lost and it starts again. Each farm needs 1 PRO and 3 USK to staff it (each turn the engine pools a colony's PRO and USK, plus TL x TL USK for every assembled AUTO unit, and staffs the farms from the pool), and burns FUEL from the colony's cargo: 0.5 x tech level a turn for FARM-1 to FARM-5, and 1 x tech level for FARM-6 to FARM-10. (FARM-2 to FARM-5 in an orbital colony in orbits 1 to 5 are solar powered and burn none.) Farms without staff or fuel sit idle, as do the farms you ordered to idle, and the next report says how many worked, how many were idle and why (by order, no staff, or no FUEL), what they burned, and what they made. Staff and FUEL go to groups in order of group number, lowest first.

Mines work the same way, after the farms. Each mining group works one deposit, and each of its MINE units needs 1 PRO, 3 USK, and 0.5 x tech level FUEL a turn. A working mine extracts 25 x its tech level x the deposit's yield of refined GOLD, FUEL, MTL, or NMTL, which goes into the colony's depot (an open-air colony) or cargo (an enclosed one) and comes out of the deposit. No order assembles a mining group yet.

Every colony has a capacity: the space its assembled structure (STRC and STRL) creates. Everything in the colony counts against it except the assembled structure itself, and the report shows the total under SPACE. A colony can't store more than it has room for: FOOD with nowhere to go is lost, and mined product with nowhere to go stays in the deposit, though the fuel is spent. An open-air colony also has a depot, outside the colony and unlimited, for everything except weapons, commodities (FOOD and CNGD), life support, and technology.

Each turn after turn 0, an entity's population eats 0.25 FOOD per population unit, times its rations, out of the entity's cargo. If there isn't enough FOOD, it eats what there is. When the ration the population actually gets is below 25%, part of it starves to death that turn: 20% of it on a 20% ration, 80% on a 5% ration, all of it with no food. The next report says how many starved.

## Where the files are

Games live in `games/{code}`, which git ignores.

```
games/alpha/
  game.json          code, seeds, and the current turn number
  stellia.json       the 100 Stellia and their coordinates
  stars.json         the stars in each Stellium
  orbits.json        10 orbits per star
  planets.json       the planets, with habitability and viability
  races.json         one race per home world
  turns/
    0/               the setup turn
      deposits.json  \
      empires.json    } state at the start of turn 0
      colonies.json  /
      order-results.json   what became of the orders given in the prior turn
      reports/
        empire-001.txt   one plain text report per empire
        ...
      orders/
        empire-001.txt   the orders each empire returned for turn 0
        ...
    1/               created by running turn 0
      ...
```

- Files directly in the game folder never change after the game is created, except `game.json`, whose `turn` moves.
- Files in `turns/N` are the state that changes: deposits get mined, empires gain players, colonies grow.
- Every file is JSON, readable with `jq`. For example, the population of the first colony at each turn:

  ```sh
  jq -c '.[0].population' games/alpha/turns/*/colonies.json
  ```

## More

- `AGENTS.md` describes the rules as implemented, the generator, and the code conventions.
- `WORKFLOW.md` describes versioning and the git workflow.

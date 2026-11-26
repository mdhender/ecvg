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
- **Turn 0 is the setup turn.** `turns/0` holds the state the generator made. Players get the turn 0 report, send setup orders, and running turn 0 produces turn 1. Demographic changes don't happen in turn 0.
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

If a rejection was a typo, roll back, fix the file, and run again. A file in the orders folder that is not named for an empire in the game stops the run before anything changes.

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

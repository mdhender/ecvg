// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command ec is the Empyrean Challenge game engine.
//
// It first loads environment files for the environment named by EC_ENV
// (default development), then reads flags from the command line and from
// EC_-prefixed environment variables, such as EC_SEED for --seed.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/mdhender/ecvg/internal/dotenv"
	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/report"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

// run executes the command line in args. It reports errors on stderr and
// returns a non-nil error if the command failed.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	env := os.Getenv("EC_ENV")
	if env == "" {
		env = "development"
	}
	if err := dotenv.Load(env); err != nil {
		fmt.Fprintf(stderr, "ec: environment %s: %v\n", env, err)
		return err
	}

	root := newRootCommand(stdout)
	if err := root.Parse(args, ff.WithEnvVarPrefix("EC")); err != nil {
		fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			return nil
		}
		fmt.Fprintf(stderr, "ec: %v\n", err)
		return err
	}
	if err := root.Run(ctx); err != nil {
		if errors.Is(err, ff.ErrNoExec) {
			fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		}
		fmt.Fprintf(stderr, "ec: %v\n", err)
		return err
	}
	return nil
}

func newRootCommand(stdout io.Writer) *ff.Command {
	rootFlags := ff.NewFlagSet("ec")
	root := &ff.Command{
		Name:  "ec",
		Usage: "ec [FLAGS] <SUBCOMMAND> ...",
		Flags: rootFlags,
	}

	turnFlags := ff.NewFlagSet("turn").SetParent(rootFlags)
	turnCmd := &ff.Command{
		Name:      "turn",
		Usage:     "ec turn <SUBCOMMAND> ...",
		ShortHelp: "process turns",
		Flags:     turnFlags,
	}
	root.Subcommands = append(root.Subcommands, turnCmd)

	runFlags := ff.NewFlagSet("run").SetParent(turnFlags)
	turnCmd.Subcommands = append(turnCmd.Subcommands, &ff.Command{
		Name:      "run",
		Usage:     "ec turn run <PATH>",
		ShortHelp: "run a game's current turn",
		LongHelp: "Run processes the current turn of the game in PATH, a folder made by ecgen.\n" +
			"It reads the state as of the current turn, N, from PATH/turns/N, runs the\n" +
			"turn's phases, writes the new state to PATH/turns/N+1, and then makes N+1\n" +
			"the current turn in game.json. PATH/turns/N is not changed. It fails if\n" +
			"PATH/turns/N+1 already exists.",
		Flags: runFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("turn run: want exactly one PATH, got %d arguments", len(args))
			}
			d, err := game.LoadData(args[0])
			if err != nil {
				return fmt.Errorf("turn run: %w", err)
			}
			turn := d.Game.Turn
			engine.RunTurn(&d)
			if err := d.WriteTurn(args[0]); err != nil {
				return fmt.Errorf("turn run: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "ran turn %d of game %s; the state as of turn %d is in %s\n",
				turn, d.Game.Code, d.Game.Turn, game.TurnDir(args[0], d.Game.Turn))
			return nil
		},
	})

	showFlags := ff.NewFlagSet("show").SetParent(turnFlags)
	turnCmd.Subcommands = append(turnCmd.Subcommands, &ff.Command{
		Name:      "show",
		Usage:     "ec turn show <PATH>",
		ShortHelp: "show a game's current turn and its turn folders",
		LongHelp: "Show prints the current turn of the game in PATH and, for every turn folder\n" +
			"in PATH/turns, how many reports and orders it holds.",
		Flags: showFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("turn show: want exactly one PATH, got %d arguments", len(args))
			}
			g, turns, err := game.Turns(args[0])
			if err != nil {
				return fmt.Errorf("turn show: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "game %s is at turn %d\n", g.Code, g.Turn)
			found := false
			for _, t := range turns {
				mark := ""
				if t.Current {
					mark, found = "  <- current", true
				}
				_, _ = fmt.Fprintf(stdout, "turn %3d: %4d reports, %4d orders%s\n", t.Turn, t.Reports, t.Orders, mark)
			}
			if !found {
				return fmt.Errorf("turn show: %s is missing", game.TurnDir(args[0], g.Turn))
			}
			return nil
		},
	})

	rollbackFlags := ff.NewFlagSet("rollback").SetParent(turnFlags)
	confirm := rollbackFlags.BoolLong("confirm", "delete the current turn's folder (without it, rollback only says what it would do)")
	turnCmd.Subcommands = append(turnCmd.Subcommands, &ff.Command{
		Name:      "rollback",
		Usage:     "ec turn rollback [--confirm] <PATH>",
		ShortHelp: "undo the last turn run",
		LongHelp: "Rollback undoes the last ec turn run of the game in PATH. If the current turn\n" +
			"is N, it makes N-1 the current turn in game.json and deletes PATH/turns/N:\n" +
			"the state as of turn N and any reports and orders in that folder. The orders\n" +
			"for turn N-1 are kept, so the turn can be run again. It fails at turn 0.\n" +
			"\n" +
			"Without --confirm, rollback changes nothing: it says what it would delete and\n" +
			"fails.",
		Flags: rollbackFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("turn rollback: want exactly one PATH, got %d arguments", len(args))
			}
			if !*confirm {
				g, turns, err := game.Turns(args[0])
				if err != nil {
					return fmt.Errorf("turn rollback: %w", err)
				}
				if g.Turn <= 0 {
					return fmt.Errorf("turn rollback: turn %d: %w", g.Turn, game.ErrNoPriorTurn)
				}
				reports, orders := 0, 0
				for _, t := range turns {
					if t.Current {
						reports, orders = t.Reports, t.Orders
					}
				}
				_, _ = fmt.Fprintf(stdout, "rollback would return game %s to turn %d and delete %s (the state as of turn %d, %d reports, %d orders)\n",
					g.Code, g.Turn-1, game.TurnDir(args[0], g.Turn), g.Turn, reports, orders)
				return fmt.Errorf("turn rollback: nothing changed; run again with --confirm to do it")
			}
			g, err := game.Rollback(args[0])
			if err != nil {
				return fmt.Errorf("turn rollback: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "rolled game %s back to turn %d; deleted %s\n",
				g.Code, g.Turn, game.TurnDir(args[0], g.Turn+1))
			return nil
		},
	})

	reportFlags := ff.NewFlagSet("report").SetParent(turnFlags)
	turnCmd.Subcommands = append(turnCmd.Subcommands, &ff.Command{
		Name:      "report",
		Usage:     "ec turn report <PATH>",
		ShortHelp: "write the reports for a game's current turn",
		LongHelp: "Report writes a plain text report for every empire of the game in PATH, as of\n" +
			"the current turn, N, into PATH/turns/N/reports, one file per empire named\n" +
			"empire-{id}.txt. It replaces reports already there.",
		Flags: reportFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("turn report: want exactly one PATH, got %d arguments", len(args))
			}
			d, err := game.LoadData(args[0])
			if err != nil {
				return fmt.Errorf("turn report: %w", err)
			}
			out, n, err := report.WriteAll(args[0], d)
			if err != nil {
				return fmt.Errorf("turn report: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "wrote %d reports for turn %d of game %s to %s\n", n, d.Game.Turn, d.Game.Code, out)
			return nil
		},
	})

	return root
}

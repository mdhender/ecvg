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

	return root
}

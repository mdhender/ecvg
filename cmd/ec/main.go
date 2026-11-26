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
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mdhender/ecvg/internal/dotenv"
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

	gameFlags := ff.NewFlagSet("game").SetParent(rootFlags)
	gameCmd := &ff.Command{
		Name:      "game",
		Usage:     "ec game <SUBCOMMAND> ...",
		ShortHelp: "manage games",
		Flags:     gameFlags,
	}
	root.Subcommands = append(root.Subcommands, gameCmd)

	createFlags := ff.NewFlagSet("create").SetParent(gameFlags)
	var seed *uint64
	createFlags.FuncLong("seed", func(s string) error {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return fmt.Errorf("want an unsigned 64-bit integer (0x prefix for hex)")
		}
		seed = &v
		return nil
	}, "first master seed (default random)")
	gameCmd.Subcommands = append(gameCmd.Subcommands, &ff.Command{
		Name:      "create",
		Usage:     "ec game create [FLAGS] <PATH>",
		ShortHelp: "create a new game in an existing folder",
		LongHelp: "Create writes game.json, stellia.json, stars.json, orbits.json, planets.json,\n" +
			"and deposits.json into PATH, an existing folder, by convention games/{code}.\n" +
			"The folder's name is the game code. Create never creates folders, and it fails\n" +
			"if PATH is missing, is not a folder, or already holds any of those files.",
		Flags: createFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("game create: want exactly one PATH, got %d arguments", len(args))
			}
			seed1 := rand.Uint64()
			if seed != nil {
				seed1 = *seed
			}
			d, err := game.Create(args[0], seed1)
			if err != nil {
				return fmt.Errorf("game create: %w", err)
			}
			fmt.Fprintf(stdout, "created game %s in %s: %d stellia, %d stars, %d orbits, %d planets, %d deposits\n",
				d.Game.Code, args[0], len(d.Stellia), len(d.Stars), len(d.Orbits), len(d.Planets), len(d.Deposits))
			return nil
		},
	})

	templateFlags := ff.NewFlagSet("homeworld-template").SetParent(gameFlags)
	gameCmd.Subcommands = append(gameCmd.Subcommands, &ff.Command{
		Name:      "homeworld-template",
		Usage:     "ec game homeworld-template <PATH>",
		ShortHelp: "write the default home world template for a game",
		LongHelp: "Homeworld-template writes homeworld-template.json into PATH, the folder of an\n" +
			"existing game. It describes every home world: the orbit of its home star, its\n" +
			"viability, and its deposits. Edit it before creating the home worlds. It is\n" +
			"never overwritten: delete it first to start again from the default.",
		Flags: templateFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("game homeworld-template: want exactly one PATH, got %d arguments", len(args))
			}
			t, err := game.CreateHomeWorldTemplate(args[0])
			if err != nil {
				return fmt.Errorf("game homeworld-template: %w", err)
			}
			fmt.Fprintf(stdout, "wrote %s: orbit %d, viability %d, %d deposits\n",
				filepath.Join(args[0], game.HomeWorldTemplateFileName), t.OrbitNo, t.Viability, len(t.Deposits))
			return nil
		},
	})

	return root
}

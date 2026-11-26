// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command ecgen is the Empyrean Challenge generator.
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
		_, _ = fmt.Fprintf(stderr, "ecgen: environment %s: %v\n", env, err)
		return err
	}

	root := newRootCommand(stdout)
	if err := root.Parse(args, ff.WithEnvVarPrefix("EC")); err != nil {
		_, _ = fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			return nil
		}
		_, _ = fmt.Fprintf(stderr, "ecgen: %v\n", err)
		return err
	}
	if err := root.Run(ctx); err != nil {
		if errors.Is(err, ff.ErrNoExec) {
			_, _ = fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		}
		_, _ = fmt.Fprintf(stderr, "ecgen: %v\n", err)
		return err
	}
	return nil
}

func newRootCommand(stdout io.Writer) *ff.Command {
	rootFlags := ff.NewFlagSet("ecgen")
	root := &ff.Command{
		Name:  "ecgen",
		Usage: "ecgen [FLAGS] <SUBCOMMAND> ...",
		Flags: rootFlags,
	}

	createFlags := ff.NewFlagSet("create").SetParent(rootFlags)
	var seed *uint64
	createFlags.FuncLong("seed", func(s string) error {
		v, err := strconv.ParseUint(s, 0, 64)
		if err != nil {
			return fmt.Errorf("want an unsigned 64-bit integer (0x prefix for hex)")
		}
		seed = &v
		return nil
	}, "first master seed (default random)")
	homeStarPath := createFlags.StringLong("homestar-template", "", "home star template file (default built in)")
	empirePath := createFlags.StringLong("empire-template", "", "empire template file (default built in)")
	root.Subcommands = append(root.Subcommands, &ff.Command{
		Name:      "create",
		Usage:     "ecgen create [FLAGS] <PATH>",
		ShortHelp: "generate a new game in an existing folder",
		LongHelp: "Create writes game.json, stellia.json, stars.json, orbits.json, planets.json,\n" +
			"deposits.json, races.json, empires.json, and colonies.json into PATH, an\n" +
			"existing folder, by convention games/{code}. The folder's name is the game\n" +
			"code. Create never creates folders, and it fails if PATH is missing, is not a\n" +
			"folder, or already holds any of those files.\n" +
			"\n" +
			"The game has 100 Stellia: 1 with 5 stars, 3 with 4, 10 with 3, 30 with 2, and\n" +
			"56 with 1. The first star of every 3-star Stellium is a home star, built from\n" +
			"the home star template; the planet in its orbit 3 is a home world. Each home\n" +
			"world gets one race and 15 empires, each built from the empire template.",
		Flags: createFlags,
		Exec: func(ctx context.Context, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("create: want exactly one PATH, got %d arguments", len(args))
			}
			seed1 := rand.Uint64()
			if seed != nil {
				seed1 = *seed
			}
			d, err := create(args[0], seed1, *homeStarPath, *empirePath)
			if err != nil {
				return fmt.Errorf("create: %w", err)
			}
			_, _ = fmt.Fprintf(stdout, "created game %s in %s: %d stellia, %d stars, %d orbits, %d planets, %d deposits, %d races, %d empires, %d colonies\n",
				d.Game.Code, args[0], len(d.Stellia), len(d.Stars), len(d.Orbits), len(d.Planets), len(d.Deposits),
				len(d.Races), len(d.Empires), len(d.Colonies))
			return nil
		},
	})

	return root
}

// create generates a game with first seed seed1 and writes it into the
// existing folder dir, whose name is the game code. The templates come from
// the files homeStarPath and empirePath, or the built-in defaults if a path is
// empty.
func create(dir string, seed1 uint64, homeStarPath, empirePath string) (game.Data, error) {
	code, err := game.Code(dir)
	if err != nil {
		return game.Data{}, err
	}
	data, err := loadTemplate(homeStarPath, defaultHomeStarTemplate)
	if err != nil {
		return game.Data{}, err
	}
	h, err := parseHomeStarTemplate(data)
	if err != nil {
		return game.Data{}, fmt.Errorf("home star template: %w", err)
	}
	if data, err = loadTemplate(empirePath, defaultEmpireTemplate); err != nil {
		return game.Data{}, err
	}
	e, err := parseEmpireTemplate(data)
	if err != nil {
		return game.Data{}, fmt.Errorf("empire template: %w", err)
	}
	d, err := generate(code, seed1, h, e)
	if err != nil {
		return game.Data{}, err
	}
	if err := d.Write(dir); err != nil {
		return game.Data{}, err
	}
	return d, nil
}

// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Command ecview shows a game's data as pages in a browser.
//
// Like ec, it first loads environment files for the environment named by
// EC_ENV (default development), then reads flags from the command line and
// from EC_-prefixed environment variables, such as EC_STORE for --store.
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/mdhender/ecvg/internal/dotenv"
	"github.com/mdhender/ecvg/internal/game"
	"github.com/mdhender/ecvg/internal/view"
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
		fmt.Fprintf(stderr, "ecview: environment %s: %v\n", env, err)
		return err
	}

	root := newRootCommand(stdout)
	if err := root.Parse(args, ff.WithEnvVarPrefix("EC")); err != nil {
		fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		if errors.Is(err, ff.ErrHelp) {
			return nil
		}
		fmt.Fprintf(stderr, "ecview: %v\n", err)
		return err
	}
	if err := root.Run(ctx); err != nil {
		if errors.Is(err, ff.ErrNoExec) {
			fmt.Fprintf(stderr, "%s\n", ffhelp.Command(root.GetSelected()))
		}
		fmt.Fprintf(stderr, "ecview: %v\n", err)
		return err
	}
	return nil
}

func newRootCommand(stdout io.Writer) *ff.Command {
	rootFlags := ff.NewFlagSet("ecview")
	store := rootFlags.StringLong("store", "", "game folder to read, by convention games/{code} (required)")
	root := &ff.Command{
		Name:  "ecview",
		Usage: "ecview --store <PATH> <SUBCOMMAND> ...",
		Flags: rootFlags,
	}

	// page adds a subcommand that writes one page and opens it in a browser.
	page := func(name, shortHelp, longHelp string, render func(w io.Writer, store string) (code string, err error)) {
		flags := ff.NewFlagSet(name).SetParent(rootFlags)
		output := flags.StringLong("output", "", "write the page to this file (default a new temporary file)")
		noOpen := flags.BoolLong("no-open", "write the page but don't open it in a browser")
		root.Subcommands = append(root.Subcommands, &ff.Command{
			Name:      name,
			Usage:     "ecview --store <PATH> " + name + " [FLAGS]",
			ShortHelp: shortHelp,
			LongHelp:  longHelp,
			Flags:     flags,
			Exec: func(ctx context.Context, args []string) error {
				if len(args) != 0 {
					return fmt.Errorf("%s: want no arguments, got %d", name, len(args))
				}
				if *store == "" {
					return fmt.Errorf("%s: --store is required", name)
				}
				path, err := writePage(*store, *output, name, render)
				if err != nil {
					return fmt.Errorf("%s: %w", name, err)
				}
				fmt.Fprintf(stdout, "wrote %s\n", path)
				if *noOpen {
					return nil
				}
				if err := openBrowser(ctx, path); err != nil {
					return fmt.Errorf("%s: open browser: %w", name, err)
				}
				return nil
			},
		})
	}

	page("stellia", "show a map of the game's Stellia",
		"Stellia reads game.json, stellia.json, and stars.json from the store and\n"+
			"writes a self-contained HTML page with a rotatable 3D map and a table of the\n"+
			"Stellia, then opens it in the default browser.",
		func(w io.Writer, store string) (string, error) {
			g, err := game.Load(store)
			if err != nil {
				return "", err
			}
			stellia, err := game.LoadStellia(store)
			if err != nil {
				return "", err
			}
			stars, err := game.LoadStars(store)
			if err != nil {
				return "", err
			}
			return g.Code, view.Stellia(w, g.Code, stellia, stars)
		})

	page("empires", "show the game's empires and their colonies",
		"Empires reads all of the game's files from the store and writes a\n"+
			"self-contained HTML page listing the empires, which can be searched, filtered\n"+
			"by race, and sorted. Selecting an empire shows its home world and its\n"+
			"colonies: population, units, cargo, and groups. It then opens the page in the\n"+
			"default browser.",
		func(w io.Writer, store string) (string, error) {
			d, err := game.LoadData(store)
			if err != nil {
				return "", err
			}
			return d.Game.Code, view.Empires(w, d)
		})

	return root
}

// writePage renders the page called name for the game in store to output, or
// to a new temporary file if output is empty, and returns the file's absolute
// path. Render writes the page and returns the game's code. The page is
// rendered in memory first, so a failure leaves no file behind and an existing
// output file untouched.
func writePage(store, output, name string, render func(w io.Writer, store string) (code string, err error)) (string, error) {
	var buf bytes.Buffer
	code, err := render(&buf, store)
	if err != nil {
		return "", err
	}
	var fd *os.File
	if output == "" {
		fd, err = os.CreateTemp("", "ecview-"+code+"-"+name+"-*.html")
	} else {
		fd, err = os.Create(output)
	}
	if err != nil {
		return "", err
	}
	_, err = fd.Write(buf.Bytes())
	if cerr := fd.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return "", err
	}
	return filepath.Abs(fd.Name())
}

// openBrowser opens the file at the absolute path in the default browser.
func openBrowser(ctx context.Context, path string) error {
	u := (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", u)
	case "windows":
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", u)
	}
	return cmd.Run()
}

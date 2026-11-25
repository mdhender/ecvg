// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package game reads and writes a game's data, kept as JSON files in the
// game's folder (by convention games/{code}): game.json holds the master data,
// stellia.json the Stellia, and stars.json their stars.
package game

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/prng"
)

// FileName is the name of the master data file in a game's folder.
const FileName = "game.json"

const (
	ErrGameExists   = cerrs.Error("game already exists")
	ErrInvalidCode  = cerrs.Error("invalid game code")
	ErrInvalidSeed  = cerrs.Error("invalid seed")
	ErrNotDirectory = cerrs.Error("not a directory")
)

// Game is a game's master data.
type Game struct {
	// Code identifies the game. It is the name of the game's folder.
	Code string `json:"code"`
	// Seed1 and Seed2 are the master seeds for prng.New.
	Seed1 Seed `json:"seed1"`
	Seed2 Seed `json:"seed2"`
}

// Seed is a master seed. It is stored in JSON as a string of 16 lowercase
// hex digits, because JSON readers often hold numbers as float64, which
// cannot represent every uint64.
type Seed uint64

// MarshalText implements encoding.TextMarshaler.
func (s Seed) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%016x", uint64(s)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. It accepts only the
// form MarshalText writes.
func (s *Seed) UnmarshalText(text []byte) error {
	if len(text) != 16 {
		return fmt.Errorf("%w: %q: want 16 hex digits", ErrInvalidSeed, text)
	}
	v, err := strconv.ParseUint(string(text), 16, 64)
	if err != nil {
		return fmt.Errorf("%w: %q: want 16 hex digits", ErrInvalidSeed, text)
	}
	*s = Seed(v)
	return nil
}

// Create creates a new game in dir and returns its master data, Stellia, and
// stars. It writes game.json, stellia.json, and stars.json. The game's code is
// the name of dir, its first seed is seed1, its second seed is
// prng.SplitMix64(seed1), and its Stellia and stars come from GenerateStellia.
//
// Create never creates folders. It fails if dir does not exist, is not a
// directory, or already holds any of the files. It writes all of the files or
// none of them.
func Create(dir string, seed1 uint64) (Game, []Stellium, []Star, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Game{}, nil, nil, err
	}
	sb, err := os.Stat(abs)
	if err != nil {
		return Game{}, nil, nil, err
	} else if !sb.IsDir() {
		return Game{}, nil, nil, fmt.Errorf("%s: %w", dir, ErrNotDirectory)
	}
	code := filepath.Base(abs)
	if code == string(filepath.Separator) || code == "." {
		return Game{}, nil, nil, fmt.Errorf("%s: %w", dir, ErrInvalidCode)
	}

	g := Game{Code: code, Seed1: Seed(seed1), Seed2: Seed(prng.SplitMix64(seed1))}
	stellia, stars := GenerateStellia(prng.New(uint64(g.Seed1), uint64(g.Seed2)))

	var written []string
	for _, file := range []struct {
		name string
		v    any
	}{
		{FileName, g},
		{StelliaFileName, stellia},
		{StarsFileName, stars},
	} {
		path := filepath.Join(dir, file.name)
		if err := writeNew(path, file.v); err != nil {
			for _, path := range written {
				_ = os.Remove(path)
			}
			return Game{}, nil, nil, err
		}
		written = append(written, path)
	}
	return g, stellia, stars, nil
}

// writeNew writes v as indented JSON to a new file at path. It fails with
// ErrGameExists if path exists, and removes the file if writing fails.
func writeNew(path string, v any) error {
	data, err := json.Marshal(v, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	data = append(data, '\n')

	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("%s: %w", path, ErrGameExists)
	} else if err != nil {
		return err
	}
	if _, err = fd.Write(data); err == nil {
		err = fd.Sync()
	}
	if cerr := fd.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(path) // don't leave a partial file behind
		return err
	}
	return nil
}

// Load reads game.json from dir.
func Load(dir string) (Game, error) {
	var g Game
	if err := loadJSON(dir, FileName, &g); err != nil {
		return Game{}, err
	}
	return g, nil
}

// loadJSON reads the file name in dir into v, rejecting unknown members.
func loadJSON(dir, name string, v any) error {
	path := filepath.Join(dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, v, json.RejectUnknownMembers(true)); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

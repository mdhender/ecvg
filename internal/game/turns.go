// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package game

import (
	"cmp"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/mdhender/ecvg/internal/cerrs"
)

// ReportsDirName and OrdersDirName are the names of the folders in a turn's
// folder that hold the reports made from the turn's state and the orders the
// players return for the turn.
const (
	ReportsDirName = "reports"
	OrdersDirName  = "orders"
)

const ErrNoPriorTurn = cerrs.Error("no prior turn")

// TurnInfo describes one turn's folder.
type TurnInfo struct {
	Turn int
	// Current is true for the game's current turn.
	Current bool
	// Reports and Orders count the files in the turn's reports and orders
	// folders.
	Reports, Orders int
}

// Turns describes the turn folders of the game in dir, in turn order. Entries
// of the turns folder that are not named by a turn number are ignored.
func Turns(dir string) (Game, []TurnInfo, error) {
	g, err := Load(dir)
	if err != nil {
		return Game{}, nil, err
	}
	entries, err := os.ReadDir(filepath.Join(dir, TurnsDirName))
	if err != nil {
		return Game{}, nil, err
	}
	var turns []TurnInfo
	for _, e := range entries {
		n, err := strconv.Atoi(e.Name())
		if err != nil || n < 0 || !e.IsDir() || strconv.Itoa(n) != e.Name() {
			continue
		}
		info := TurnInfo{Turn: n, Current: n == g.Turn}
		if info.Reports, err = countFiles(filepath.Join(TurnDir(dir, n), ReportsDirName)); err != nil {
			return Game{}, nil, err
		}
		if info.Orders, err = countFiles(filepath.Join(TurnDir(dir, n), OrdersDirName)); err != nil {
			return Game{}, nil, err
		}
		turns = append(turns, info)
	}
	slices.SortFunc(turns, func(a, b TurnInfo) int { return cmp.Compare(a.Turn, b.Turn) })
	return g, turns, nil
}

// countFiles returns the number of files in dir, or 0 if dir does not exist.
func countFiles(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() {
			n++
		}
	}
	return n, nil
}

// Rollback undoes the last turn run of the game in dir: it makes the prior
// turn the current turn in game.json and deletes the folder of what was the
// current turn, with its state and any reports and orders in it. It returns
// the game as it is afterwards. It fails with ErrNoPriorTurn at turn 0.
//
// The folder is first renamed out of the way and deleted only after game.json
// is replaced; if replacing game.json fails, the folder is put back.
func Rollback(dir string) (Game, error) {
	g, err := Load(dir)
	if err != nil {
		return Game{}, err
	}
	if g.Turn <= 0 {
		return Game{}, fmt.Errorf("turn %d: %w", g.Turn, ErrNoPriorTurn)
	}
	turnDir := TurnDir(dir, g.Turn)
	g.Turn--
	if _, err := os.Stat(TurnDir(dir, g.Turn)); err != nil {
		return Game{}, err
	}
	data, err := json.Marshal(&g)
	if err != nil {
		return Game{}, err
	}

	doomed, err := os.MkdirTemp(filepath.Dir(turnDir), ".rollback-*")
	if err != nil {
		return Game{}, err
	}
	moved := filepath.Join(doomed, filepath.Base(turnDir))
	if err := os.Rename(turnDir, moved); err != nil && !errors.Is(err, fs.ErrNotExist) {
		_ = os.Remove(doomed)
		return Game{}, err
	}
	if err := replaceFile(filepath.Join(dir, FileName), append(data, '\n')); err != nil {
		_ = os.Rename(moved, turnDir)
		_ = os.Remove(doomed)
		return Game{}, err
	}
	if err := os.RemoveAll(doomed); err != nil {
		return Game{}, err
	}
	return g, nil
}

// replaceFile writes data to a temporary file beside path and renames it over
// path.
func replaceFile(path string, data []byte) error {
	tmp := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

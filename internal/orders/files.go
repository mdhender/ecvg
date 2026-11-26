// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package orders

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mdhender/ecvg/internal/cerrs"
	"github.com/mdhender/ecvg/internal/game"
)

const (
	ErrInvalidFileName = cerrs.Error("not an orders file name; want empire-{id}.txt")
	ErrDuplicateFile   = cerrs.Error("more than one orders file for the empire")
)

// File is the orders one empire returned for a turn.
type File struct {
	// EmpireID is the ID of the empire, taken from the file's name.
	EmpireID int
	// Name is the file's name.
	Name string
	// Lines are the file's orders.
	Lines []Line
}

// FileName returns the name of the orders file for the empire with ID
// empireID. It is the same as the name of the empire's report.
func FileName(empireID int) string {
	return fmt.Sprintf("empire-%03d.txt", empireID)
}

// EmpireID returns the ID of the empire that the orders file called name
// belongs to. The name is "empire-", the ID, and ".txt"; the ID may have
// leading zeros.
func EmpireID(name string) (int, error) {
	digits, ok := strings.CutPrefix(name, "empire-")
	if ok {
		digits, ok = strings.CutSuffix(digits, ".txt")
	}
	id, isNumber := number(Token{Text: digits})
	if !ok || !isNumber || id < 1 {
		return 0, fmt.Errorf("%s: %w", name, ErrInvalidFileName)
	}
	return id, nil
}

// LoadTurn reads every orders file of one turn of the game in dir, from
// dir/turns/{turn}/orders, and returns them in empire ID order. A missing
// orders folder holds no orders. Folders and files whose names start with "."
// are ignored; any other file must be named for an empire, and an empire can
// have only one.
func LoadTurn(dir string, turn int) ([]File, error) {
	ordersDir := filepath.Join(game.TurnDir(dir, turn), game.OrdersDirName)
	entries, err := os.ReadDir(ordersDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		id, err := EmpireID(e.Name())
		if err != nil {
			return nil, err
		}
		if i := slices.IndexFunc(files, func(f File) bool { return f.EmpireID == id }); i >= 0 {
			return nil, fmt.Errorf("%s and %s: %w", files[i].Name, e.Name(), ErrDuplicateFile)
		}
		f, err := os.Open(filepath.Join(ordersDir, e.Name()))
		if err != nil {
			return nil, err
		}
		lines, err := Parse(f)
		_ = f.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		files = append(files, File{EmpireID: id, Name: e.Name(), Lines: lines})
	}
	slices.SortFunc(files, func(a, b File) int { return cmp.Compare(a.EmpireID, b.EmpireID) })
	return files, nil
}

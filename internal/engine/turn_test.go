// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package engine_test

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/mdhender/ecvg/internal/engine"
	"github.com/mdhender/ecvg/internal/game"
)

// The stages are the manual's B3.1 to B3.22, in order.
func TestStages(t *testing.T) {
	stages := engine.Stages()
	if len(stages) != 22 {
		t.Fatalf("got %d stages, want 22", len(stages))
	}
	for i, s := range stages {
		if s.No != i+1 || s.Name == "" {
			t.Errorf("stage %d: got No %d, Name %q", i+1, s.No, s.Name)
		}
	}
}

// RunTurn logs a line for each stage that isn't built, and only for those.
func TestRunTurnLogsStagesNotImplemented(t *testing.T) {
	var buf bytes.Buffer
	out, flags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() { log.SetOutput(out); log.SetFlags(flags) }()

	var d game.Data
	if _, err := engine.RunTurn(&d, nil); err != nil {
		t.Fatal(err)
	}
	var want strings.Builder
	for _, s := range engine.Stages() {
		if !s.Implemented() {
			fmt.Fprintf(&want, "stage %d: %s: not implemented\n", s.No, s.Name)
		}
	}
	if want.Len() == 0 {
		t.Fatal("every stage is implemented; this test is out of date")
	}
	if buf.String() != want.String() {
		t.Errorf("got log:\n%swant:\n%s", buf.String(), want.String())
	}
}

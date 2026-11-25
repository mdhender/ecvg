// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package prng

// Key is one element of a key path.
type Key int64

// Domain tags name the purpose of a draw and start every root key path, so two
// purposes never share a stream. Tag 0 is invalid, so a forgotten tag fails
// instead of aliasing another purpose.
//
// Append new tags to the end with explicit values and update lastTag. Never
// renumber a tag: that changes every game generated with it.
const (
	// TagRoot is a placeholder.
	TagRoot Key = 1
	// TagStellia places the Stellia when a game is created.
	TagStellia Key = 2
	// TagPlanets places the planets in orbits when a game is created.
	TagPlanets Key = 3
	// TagDeposits decides how many deposits each planet has and their types
	// when a game is created.
	TagDeposits Key = 4
	// TagDepositAmounts, followed by a deposit's ID, draws that deposit's size
	// and yield when a game is created.
	TagDepositAmounts Key = 5

	lastTag = TagDepositAmounts
)

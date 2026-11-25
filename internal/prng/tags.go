// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: MIT

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

	lastTag = TagRoot
)

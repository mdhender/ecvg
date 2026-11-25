// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package prng provides deterministic random streams that can be split into
// independent child streams, so generators produce the same game on any
// machine no matter what order they create objects in.
//
// New returns a stream seeded from a game's two master seeds. Derive splits off
// the stream addressed by a key path. The path's first element is a domain tag
// from tags.go; the rest identify the instance, such as a location's unique ID.
// Instance keys must be intrinsic to the game, such as coordinates, never
// database row ids, whose values depend on insertion order.
//
// A child's seeds are the first 16 bytes of a hash of its parent's seeds and
// the path:
//
//	digest = SHA-256(seed1, seed2, len(path), path...)   // each a big-endian 64-bit value
//
// A child depends only on its parent's seeds and its path, never on how many
// values have been drawn from the parent, so the same path always yields the
// same stream. A child can be split again; its paths are relative to it.
//
// The hash encoding and the tag numbers decide every generated game. Changes
// are allowed during alpha (kill and fill); after alpha they are frozen.
package prng

import "math/rand/v2"

// New returns the root Stream for the given master seeds. Its values come
// from a PCG seeded directly with seed1 and seed2; use Derive to split off
// streams for individual purposes.
func New(seed1, seed2 uint64) *Stream {
	return &Stream{pcg: rand.NewPCG(seed1, seed2), seeds: seeds{seed1: seed1, seed2: seed2}}
}

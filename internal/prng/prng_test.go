// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: MIT

package prng_test

import (
	"testing"

	"github.com/mdhender/ecvg/internal/prng"
)

const seed1, seed2 = 0x0123456789abcdef, 0xfedcba9876543210

func TestRootStreamMatchesPublishedVector(t *testing.T) {
	s := prng.New(seed1, seed2)
	for i, want := range []uint64{11748147493716979836, 13673077492115055638, 15398964789732017239, 6498083842776177775} {
		if got := s.Uint64(); got != want {
			t.Errorf("draw %d: got %d, want %d", i, got, want)
		}
	}
}

func TestDeriveMatchesPublishedVector(t *testing.T) {
	s := prng.New(seed1, seed2).Derive(prng.TagRoot)
	for i, want := range []uint64{11781881871367131112, 6486566634792659523, 12344156845670623292, 5274862108971197392} {
		if got := s.Uint64(); got != want {
			t.Errorf("draw %d: got %d, want %d", i, got, want)
		}
	}
}

func TestNestedDeriveMatchesPublishedVector(t *testing.T) {
	s := prng.New(seed1, seed2).Derive(prng.TagRoot).Derive(prng.TagRoot)
	if got, want := s.Uint64(), uint64(17035658063521021138); got != want {
		t.Errorf("got %d, want %d", got, want)
	}
}

func TestDeriveDependsOnlyOnAddress(t *testing.T) {
	root := prng.New(1, 2)
	want := root.Derive(prng.TagRoot, 5, 9, -2).Uint64()

	// Drawing from the parent or from other children does not move the address.
	root.Uint64()
	root.Derive(prng.TagRoot, 8, 1, 4).Uint64()
	if got := root.Derive(prng.TagRoot, 5, 9, -2).Uint64(); got != want {
		t.Errorf("draw changed after the parent was used: got %d, want %d", got, want)
	}

	// Path length is part of the address, and the order of elements matters.
	for _, path := range [][]prng.Key{{prng.TagRoot, 5, 9, -2, 0}, {prng.TagRoot, 9, 5, -2}} {
		if root.Derive(path...).Uint64() == want {
			t.Errorf("path %v draws the same as [1 5 9 -2]", path)
		}
	}

	// A nested path is relative to its parent, not flattened into one path.
	if root.Derive(prng.TagRoot).Derive(prng.TagRoot, 5, 9, -2).Uint64() == want {
		t.Error("nested derive draws the same as the flat path")
	}
}

func TestPathMustStartWithDomainTag(t *testing.T) {
	root := prng.New(1, 2)
	for name, call := range map[string]func(){
		"empty path":              func() { root.Derive() },
		"zero tag":                func() { root.Derive(0, 1) },
		"unregistered tag":        func() { root.Derive(99, 1) },
		"negative tag":            func() { root.Derive(-1, 1) },
		"nested zero tag":         func() { root.Derive(prng.TagRoot).Derive(0) },
		"nested unregistered tag": func() { root.Derive(prng.TagRoot).Derive(99) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("did not panic")
				}
			}()
			call()
		})
	}
}

func TestSplitMix64MatchesReference(t *testing.T) {
	// First output of the reference SplitMix64 seeded with 0.
	if got, want := prng.SplitMix64(0), uint64(0xe220a8397b1dcdaf); got != want {
		t.Errorf("got %#x, want %#x", got, want)
	}
}

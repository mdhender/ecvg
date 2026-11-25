// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: MIT

package prng

// SplitMix64 returns the first output of the SplitMix64 generator seeded with
// seed. A game's second master seed is SplitMix64 of its first.
func SplitMix64(seed uint64) uint64 {
	z := seed + 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

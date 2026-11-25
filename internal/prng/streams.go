// Copyright (c) 2026 Michael D Henderson.
// SPDX-License-Identifier: AGPL-3.0-or-later

package prng

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
)

// Stream is a deterministic draw source backed by a math/rand/v2 PCG. It
// implements math/rand/v2.Source, so rand.New(stream) gives the full
// *rand.Rand API. Use it through a *Stream: copies share draw state. A Stream
// is not safe for concurrent use.
type Stream struct {
	pcg   *rand.PCG
	seeds seeds
}

var _ rand.Source = (*Stream)(nil)

// seeds holds the two seeds a Stream was created from. Derive hashes them,
// not the stream's current state, to seed child streams.
type seeds struct {
	seed1, seed2 uint64
}

// Derive returns the Stream addressed by path, relative to s. Calls with the
// same path on streams with the same seeds return streams that produce the
// same values, regardless of how much has been drawn from s. Distinct paths
// give independent streams.
//
// The first element of path must be a domain tag from tags.go; Derive panics
// otherwise.
func (s *Stream) Derive(path ...Key) *Stream {
	digest := s.hash(path)
	seed1, seed2 := binary.BigEndian.Uint64(digest[0:8]), binary.BigEndian.Uint64(digest[8:16])
	return New(seed1, seed2)
}

// Uint64 returns the next 64 random bits.
func (s *Stream) Uint64() uint64 {
	return s.pcg.Uint64()
}

// hash returns SHA-256 over s's seed1, seed2, len(path), and each element of
// path, each written as a big-endian 64-bit value. The length prefix keeps
// paths of different depths from hashing alike. It panics if path does not
// start with a domain tag from tags.go.
func (s *Stream) hash(path []Key) [32]byte {
	if len(path) == 0 || path[0] < 1 || path[0] > lastTag {
		panic(fmt.Sprintf("prng: path must start with a domain tag from tags.go, got %v", path))
	}

	buf := make([]byte, 0, 8*(3+len(path)))
	buf = binary.BigEndian.AppendUint64(buf, s.seeds.seed1)
	buf = binary.BigEndian.AppendUint64(buf, s.seeds.seed2)
	buf = binary.BigEndian.AppendUint64(buf, uint64(len(path)))
	for _, k := range path {
		buf = binary.BigEndian.AppendUint64(buf, uint64(k))
	}
	return sha256.Sum256(buf)
}

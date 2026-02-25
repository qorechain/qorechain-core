package pqc

// SHAKE-256 merkle hash foundation — preparatory utilities for future
// post-quantum IAVL tree replacement. Pure Go, no FFI, no build tags.
//
// SHAKE-256 is an extendable-output function (XOF) from the SHA-3 family.
// It provides quantum-resistant hashing suitable for merkle tree construction.
// These helpers are used by the PQC module for hash commitments and will
// eventually replace SHA-256 in the state merkle tree.

import (
	"golang.org/x/crypto/sha3"
)

// SHAKE256Hash computes a SHAKE-256 digest of the given data with the
// specified output length in bytes. The output length can be arbitrary
// (unlike fixed-output SHA-3 variants).
func SHAKE256Hash(data []byte, outputLen int) []byte {
	h := sha3.NewShake256()
	h.Write(data)
	out := make([]byte, outputLen)
	h.Read(out)
	return out
}

// SHAKE256Hash32 computes a 32-byte (256-bit) SHAKE-256 digest.
// This is the standard output size for merkle tree nodes.
func SHAKE256Hash32(data []byte) [32]byte {
	h := sha3.NewShake256()
	h.Write(data)
	var out [32]byte
	h.Read(out[:])
	return out
}

// SHAKE256ConcatHash computes SHAKE-256(left || right) with a 32-byte output.
// Used for computing internal merkle tree node hashes from two child hashes.
// The concatenation is done without a domain separator — callers should ensure
// left and right are fixed-length (e.g., both 32 bytes) to prevent
// length-extension ambiguity.
func SHAKE256ConcatHash(left, right []byte) [32]byte {
	h := sha3.NewShake256()
	h.Write(left)
	h.Write(right)
	var out [32]byte
	h.Read(out[:])
	return out
}

// SHAKE256DomainHash computes SHAKE-256(domain || data) with a 32-byte output.
// The domain prefix provides context separation for different hash uses
// (e.g., "leaf:" vs "node:" in a merkle tree).
func SHAKE256DomainHash(domain string, data []byte) [32]byte {
	h := sha3.NewShake256()
	h.Write([]byte(domain))
	h.Write(data)
	var out [32]byte
	h.Read(out[:])
	return out
}

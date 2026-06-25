// Package qorehash defines QoreChain's canonical application-level hash.
//
// QoreChain's default hash is SHAKE-256, the extendable-output function (XOF)
// from the SHA-3 / Keccak family (FIPS 202). Together with Dilithium-5
// (signatures) and ML-KEM-1024 (key encapsulation), SHAKE-256 completes the
// chain's post-quantum cryptographic baseline: every commitment, identifier and
// Merkle root that QoreChain itself produces AND verifies is bound with a
// quantum-resistant hash by default.
//
// Scope — this package is the default for QoreChain-CONTROLLED hashing only
// (state/transition roots, withdrawal and DA commitments, cross-VM and SVM
// identifiers, QCA proposer seeding, the STARK transcript/Merkle, etc.), where
// the same QoreChain code is both the producer and the verifier. It is
// deliberately NOT used for:
//
//   - External-chain verification (bridge light-clients / proof verifiers must
//     match the foreign chain's own format: Bitcoin sha256d, Ethereum MPT
//     keccak256, Ethereum SSZ sha256, BLS/Pedersen, …). Those stay native — the
//     "hybrid only at network egress" rule.
//   - Framework / consensus hashing owned by Cosmos SDK / CometBFT / IAVL, EVM
//     ABI selectors (keccak256), and Solana SVM syscalls (sha256/keccak) that
//     external bytecode and tooling depend on.
//
// Pure Go, no cgo/FFI, no build tags — identical in the community and full
// builds so on-chain commitments are byte-for-byte reproducible everywhere.
package qorehash

import (
	"encoding/binary"
	"hash"

	"golang.org/x/crypto/sha3"
)

// Size is QoreChain's canonical digest length in bytes (256 bits).
const Size = 32

// BlockSize is the SHAKE-256 sponge rate in bytes (1088 bits).
const BlockSize = 136

// Sum256 returns the 32-byte SHAKE-256 digest of data. It is the canonical
// drop-in replacement for sha256.Sum256 at QoreChain-controlled call sites.
func Sum256(data []byte) [Size]byte {
	h := sha3.NewShake256()
	h.Write(data)
	var out [Size]byte
	h.Read(out[:])
	return out
}

// Sum returns the 32-byte SHAKE-256 digest of data as a slice.
func Sum(data []byte) []byte {
	out := Sum256(data)
	return out[:]
}

// SumN returns a SHAKE-256 digest of data with an arbitrary output length,
// exploiting the XOF nature of SHAKE-256.
func SumN(data []byte, outputLen int) []byte {
	h := sha3.NewShake256()
	h.Write(data)
	out := make([]byte, outputLen)
	h.Read(out)
	return out
}

// ConcatHash returns SHAKE-256 over left||right with a 4-byte big-endian length
// prefix on each operand, giving a second-preimage-resistant Merkle node hash
// even when the operands are variable length.
func ConcatHash(left, right []byte) [Size]byte {
	h := sha3.NewShake256()
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(left)))
	h.Write(lenBuf[:])
	h.Write(left)
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(right)))
	h.Write(lenBuf[:])
	h.Write(right)
	var out [Size]byte
	h.Read(out[:])
	return out
}

// DomainHash returns SHAKE-256 over domain||data with a 4-byte big-endian
// length prefix on each operand, providing context separation between distinct
// hash uses (e.g. "leaf:" vs "node:").
func DomainHash(domain string, data []byte) [Size]byte {
	h := sha3.NewShake256()
	domainBytes := []byte(domain)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(domainBytes)))
	h.Write(lenBuf[:])
	h.Write(domainBytes)
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(data)))
	h.Write(lenBuf[:])
	h.Write(data)
	var out [Size]byte
	h.Read(out[:])
	return out
}

// Hasher is a streaming SHAKE-256 hasher that satisfies the standard
// hash.Hash interface with a fixed 32-byte output, so it is a drop-in for
// sha256.New() at QoreChain-controlled call sites: Write the input, then call
// Sum(nil) (or Sum(prefix)) to obtain the 32-byte digest.
type Hasher struct {
	sh sha3.ShakeHash
}

// New returns a streaming SHAKE-256 Hasher.
func New() *Hasher {
	return &Hasher{sh: sha3.NewShake256()}
}

// Write absorbs more data into the running hash. It never returns an error.
func (h *Hasher) Write(p []byte) (int, error) { return h.sh.Write(p) }

// Sum appends the 32-byte SHAKE-256 digest of the data written so far to b and
// returns the result. Sum does not change the underlying state (it reads from a
// clone), so it may be called repeatedly, matching sha256 semantics.
func (h *Hasher) Sum(b []byte) []byte {
	dup := h.sh.Clone()
	var out [Size]byte
	dup.Read(out[:])
	return append(b, out[:]...)
}

// Reset resets the hasher to its initial state.
func (h *Hasher) Reset() { h.sh.Reset() }

// Size returns the canonical digest length, 32 bytes.
func (h *Hasher) Size() int { return Size }

// BlockSize returns the SHAKE-256 sponge rate, 136 bytes.
func (h *Hasher) BlockSize() int { return BlockSize }

// interface assertion: *Hasher is a standard hash.Hash.
var _ hash.Hash = (*Hasher)(nil)

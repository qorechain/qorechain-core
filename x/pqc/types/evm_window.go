package types

import (
	"encoding/binary"
	"fmt"

	sdkmath "cosmossdk.io/math"
)

// The post-quantum authorisation window for the EVM lane.
//
// THE PROBLEM IT SOLVES. The chain requires an ML-DSA-87 signature on the Cosmos
// lane and refuses the classical fallback. The EVM lane cannot carry one: MetaMask
// signs secp256k1 over an Ethereum transaction and has no way to attach a second
// signature, and it does not implement eth_signTransaction at all, so there is not
// even a moment between signing and broadcast in which one could be added. A
// post-quantum signature therefore cannot travel WITH an EVM transaction.
//
// Since it cannot travel with the transaction, it travels AHEAD of it. The account
// opens a window with a Cosmos-lane message, which is subject to the ordinary PQC
// requirement, and the EVM lane then admits only transactions covered by a live
// window. The post-quantum key is required, the classical key alone is never
// sufficient, and no EVM tooling has to change: inside a window an ordinary MetaMask
// transaction works unmodified, because the check reads chain state rather than
// anything in the transaction.
//
// The nearest familiar shape is an ERC-20 approval. You authorise a bounded amount
// for a bounded time, then transact normally against it.
//
// WHAT A WINDOW BOUNDS, and why each bound is there. A stolen classical key is only
// worth what an open window admits, so the window is bounded three ways at once:
//
//   - by BLOCKS, because an authorisation that never expires is not an
//     authorisation, it is a permanent downgrade to classical security;
//   - by TRANSACTION COUNT, because it is the bound a person can reason about
//     ("I am about to make three transactions");
//   - by VALUE, counting transferred value AND the maximum fee each transaction
//     could pay. Fees drain the account exactly as transfers do, and an attacker
//     holding the classical key can set any gas price, so bounding value alone would
//     leave the account drainable through fees inside an open window.
//
// Any of the three running out closes the window in practice. That is deliberate:
// each bound answers a different question, and the user should not have to reason
// about which one binds first.

// EVMAuthWindow is one account's authorisation for the EVM lane.
type EVMAuthWindow struct {
	// ExpiryHeight is the last block at which the window admits a transaction.
	ExpiryHeight uint64 `json:"expiry_height"`
	// MaxTxs and UsedTxs bound how many transactions the window admits.
	MaxTxs  uint64 `json:"max_txs"`
	UsedTxs uint64 `json:"used_txs"`
	// MaxValue and UsedValue bound the total outflow in uqor, counting value plus
	// the maximum fee of each admitted transaction.
	MaxValue  sdkmath.Int `json:"max_value"`
	UsedValue sdkmath.Int `json:"used_value"`
	// OpenedHeight is kept for auditing: it says when the authorisation was given,
	// which is the question asked first after a key compromise.
	OpenedHeight uint64 `json:"opened_height"`
}

// Bounds on what a single window may authorise. These are deliberately modest: a
// window is meant to cover a session of activity, not to be opened once and
// forgotten, and a generous default would quietly recreate the situation the window
// exists to end.
const (
	// MaxEVMWindowBlocks is about 24 hours at 5s blocks.
	MaxEVMWindowBlocks uint64 = 17280
	// MaxEVMWindowTxs bounds one authorisation's transaction count.
	MaxEVMWindowTxs uint64 = 1000
)

// EVMWindowPrefix is the KV prefix for stored windows.
var EVMWindowPrefix = []byte("pqc/evmwindow/")

// EVMWindowKey returns the store key for an account's window.
func EVMWindowKey(addr []byte) []byte {
	return append(append([]byte{}, EVMWindowPrefix...), addr...)
}

// IsLive reports whether the window still admits a transaction at this height with
// this much outflow.
//
// A window that has run out of ANY of its three bounds admits nothing. The caller
// gets a single answer rather than having to know which bound bound first.
func (w EVMAuthWindow) IsLive(height uint64, outflow sdkmath.Int) bool {
	if height > w.ExpiryHeight {
		return false
	}
	if w.UsedTxs >= w.MaxTxs {
		return false
	}
	if w.MaxValue.IsNil() || w.UsedValue.IsNil() {
		return false
	}
	return w.UsedValue.Add(outflow).LTE(w.MaxValue)
}

// Remaining reports what is left, for the error message and for the query. An
// operator who is refused needs to know which bound stopped them.
func (w EVMAuthWindow) Remaining(height uint64) (blocks, txs uint64, value sdkmath.Int) {
	if height <= w.ExpiryHeight {
		blocks = w.ExpiryHeight - height
	}
	if w.MaxTxs > w.UsedTxs {
		txs = w.MaxTxs - w.UsedTxs
	}
	value = sdkmath.ZeroInt()
	if !w.MaxValue.IsNil() && !w.UsedValue.IsNil() && w.MaxValue.GT(w.UsedValue) {
		value = w.MaxValue.Sub(w.UsedValue)
	}
	return blocks, txs, value
}

// Validate checks a window request. Every bound must be present and within the
// module's maximum: a request that omits one is refused rather than defaulted,
// because a default here is a security parameter chosen by whoever forgot it.
func (m MsgOpenEVMWindow) Validate() error {
	if m.Blocks == 0 {
		return fmt.Errorf("blocks must be positive")
	}
	if m.Blocks > MaxEVMWindowBlocks {
		return fmt.Errorf("blocks %d exceeds the maximum %d", m.Blocks, MaxEVMWindowBlocks)
	}
	if m.MaxTxs == 0 {
		return fmt.Errorf("max_txs must be positive; a window that admits nothing is a mistake")
	}
	if m.MaxTxs > MaxEVMWindowTxs {
		return fmt.Errorf("max_txs %d exceeds the maximum %d", m.MaxTxs, MaxEVMWindowTxs)
	}
	if m.MaxValue.IsNil() || !m.MaxValue.IsPositive() {
		return fmt.Errorf("max_value must be positive")
	}
	return nil
}

// WeiToUqorRoundUp converts an EVM-lane amount to the account base denom, rounding
// UP.
//
// The direction matters. The extended denom carries 18 decimals over a 6-decimal
// base, so a transfer can carry sub-uqor dust. Rounding DOWN would charge a window
// nothing for a long series of sub-uqor transfers, which is a way to drain an
// account through a window that never appears to be consumed. Rounding up can only
// ever close a window early, which is the safe direction to be wrong in.
func WeiToUqorRoundUp(wei sdkmath.Int) sdkmath.Int {
	if wei.IsNil() || !wei.IsPositive() {
		return sdkmath.ZeroInt()
	}
	scale := sdkmath.NewInt(1_000_000_000_000) // 10^12: 18 decimals down to 6
	q := wei.Quo(scale)
	if wei.Sub(q.Mul(scale)).IsPositive() {
		q = q.Add(sdkmath.OneInt())
	}
	return q
}

// EncodeHeight is used where a fixed-width big-endian height is stored.
func EncodeHeight(h uint64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], h)
	return b[:]
}

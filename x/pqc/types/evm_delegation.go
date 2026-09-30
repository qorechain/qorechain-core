package types

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Post-quantum consent for EIP-7702 delegations.
//
// THE PROBLEM. An EIP-7702 authorization tuple installs code on the account that
// signed it, and it is signed with the account's classical secp256k1 key alone.
// Anyone may carry the tuple in a type-4 transaction, so the EVM window, which
// checks the transaction's sender, never saw the account whose code changed.
// Once the delegation is installed its code acts for the account whenever
// anyone calls it. The classical key alone was therefore enough to change what
// the account is, which the window exists to prevent (QSR-2026-0307, -0310).
//
// THE RULE. From the v3.2.2 plan height the EVM lane admits a type-4
// transaction only if every authority whose tuple would take effect holds a
// post-quantum key and has consented, with a Cosmos-lane message signed with
// that key, to exactly the delegation the tuple installs. The consent is used
// up by the transaction that installs it. Clearing a delegation (the zero
// address) needs consent too, or a holder of the classical key could strip a
// user's smart account; MsgClearEVMDelegation clears it without any EVM
// transaction at all.
//
// The consent binds the delegate, not the tuple nonce. The account's Cosmos
// sequence and its EVM nonce are one counter, and the consent message itself
// moves it, so a client cannot know the nonce when it asks for consent. Single
// use and the protocol nonce already stop a tuple from being used twice.
//
// Sponsored delegation keeps working: the sponsor pays from its own window and
// the authority only consents beforehand, the same authorise-ahead pattern the
// window uses for MetaMask.

// EVMDelegationConsent is one account's consent to one delegation.
type EVMDelegationConsent struct {
	// Delegate is the 0x address the tuple must name, lower case. The zero
	// address is consent to clear the delegation.
	Delegate string `json:"delegate"`
	// GrantedHeight is kept for auditing.
	GrantedHeight uint64 `json:"granted_height"`
	// ExpiryHeight is the last block at which the consent can be used.
	ExpiryHeight uint64 `json:"expiry_height"`
}

// IsLive reports whether the consent can still be used at this height.
func (c EVMDelegationConsent) IsLive(height uint64) bool { return height <= c.ExpiryHeight }

// Matches reports whether the consent covers a tuple naming delegate.
func (c EVMDelegationConsent) Matches(delegate common.Address) bool {
	return strings.EqualFold(c.Delegate, delegate.Hex())
}

const (
	// MaxEVMDelegationConsentBlocks bounds how long a consent waits to be used;
	// the same bound as a window.
	MaxEVMDelegationConsentBlocks = MaxEVMWindowBlocks
	// MaxEVMAuthorizationsPerTx bounds the authorization list of a type-4
	// transaction on the EVM lane. Every tuple costs a signature recovery in
	// the ante handler before any gas is charged; eight covers every batching
	// wallet in use.
	MaxEVMAuthorizationsPerTx = 8
)

// EVMDelegationConsentPrefix is the KV prefix for stored consents.
var EVMDelegationConsentPrefix = []byte("pqc/evm7702/")

// EVMDelegationConsentKey returns the store key for an account's consent.
func EVMDelegationConsentKey(addr []byte) []byte {
	return append(append([]byte{}, EVMDelegationConsentPrefix...), addr...)
}

// ParseDelegate parses a delegate given as a 0x-prefixed 20-byte hex address.
func ParseDelegate(s string) (common.Address, error) {
	if !strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X") {
		return common.Address{}, fmt.Errorf("delegate %q must be a 0x-prefixed address", s)
	}
	if !common.IsHexAddress(s) {
		return common.Address{}, fmt.Errorf("delegate %q is not a 20-byte hex address", s)
	}
	return common.HexToAddress(s), nil
}

// Validate checks a consent request. Every bound must be given.
func (m MsgAuthorizeEVMDelegation) Validate() error {
	if _, err := ParseDelegate(m.Delegate); err != nil {
		return err
	}
	if m.Blocks == 0 {
		return fmt.Errorf("blocks must be positive")
	}
	if m.Blocks > MaxEVMDelegationConsentBlocks {
		return fmt.Errorf("blocks %d exceeds the maximum %d", m.Blocks, MaxEVMDelegationConsentBlocks)
	}
	return nil
}

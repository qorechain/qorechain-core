package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// TONBridge handles TON-specific bridge operations.
// Uses cross-chain messaging for bridge communication.
type TONBridge struct{}

// NewTONBridge creates a new TON bridge handler.
func NewTONBridge() *TONBridge {
	return &TONBridge{}
}

// ValidateDeposit validates a TON deposit proof.
// In production, this verifies TON block proofs and transaction inclusion.
func (b *TONBridge) ValidateDeposit(_ sdk.Context, op types.BridgeOperation) error {
	// Testnet: validation via PQC-signed attestations
	// Production TODO: TON lite client verification
	//   1. Verify block proof chain from masterchain
	//   2. Verify transaction inclusion in shard
	//   3. Verify message was sent to bridge contract
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("TON deposit requires source tx hash")
	}
	return nil
}

// ValidateWithdrawal validates a TON withdrawal request.
func (b *TONBridge) ValidateWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// TON uses raw or user-friendly addresses
	if len(op.Receiver) < 10 {
		return types.ErrInvalidDestination.Wrapf("invalid TON address format: %s", op.Receiver)
	}
	return nil
}

// EstimateConfirmationTime returns the expected confirmation time for TON.
func (b *TONBridge) EstimateConfirmationTime() int64 {
	// ~10 confirmations × 5 seconds per block = ~50 seconds
	return 50
}

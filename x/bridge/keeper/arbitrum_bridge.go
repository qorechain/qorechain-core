//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// ArbitrumBridge handles Arbitrum L2-specific bridge operations.
// Uses optimistic rollup finality with fast block times.
type ArbitrumBridge struct{}

// NewArbitrumBridge creates a new Arbitrum bridge handler.
func NewArbitrumBridge() *ArbitrumBridge {
	return &ArbitrumBridge{}
}

// ValidateDeposit validates an Arbitrum deposit proof.
// In production, this verifies Arbitrum transaction receipts and rollup proofs.
func (b *ArbitrumBridge) ValidateDeposit(_ sdk.Context, op types.BridgeOperation) error {
	// Testnet: validation via PQC-signed attestations
	// Production TODO: Arbitrum rollup verification
	//   1. Verify transaction inclusion in L2 block
	//   2. Verify rollup batch submission to L1
	//   3. Verify challenge period has elapsed for withdrawals
	//   4. Wait for 64 L2 confirmations
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("arbitrum deposit requires source tx hash")
	}
	return nil
}

// ValidateWithdrawal validates an Arbitrum withdrawal request.
func (b *ArbitrumBridge) ValidateWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Verify EVM address format (0x + 40 hex chars)
	if len(op.Receiver) != 42 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid Arbitrum address format: %s", op.Receiver)
	}
	return nil
}

// EstimateConfirmationTime returns the expected confirmation time for Arbitrum.
func (b *ArbitrumBridge) EstimateConfirmationTime() int64 {
	// 64 confirmations × 0.25 seconds per block = 16 seconds
	return 16
}

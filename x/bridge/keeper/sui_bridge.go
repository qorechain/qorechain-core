//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// SuiBridge handles Sui-specific bridge operations.
// Sui uses Move VM and checkpoint-based finality.
type SuiBridge struct{}

// NewSuiBridge creates a new Sui bridge handler.
func NewSuiBridge() *SuiBridge {
	return &SuiBridge{}
}

// ValidateDeposit validates a Sui deposit proof.
// In production, this verifies Sui checkpoint proofs and transaction inclusion.
func (b *SuiBridge) ValidateDeposit(_ sdk.Context, op types.BridgeOperation) error {
	// Testnet: validation via PQC-signed attestations
	// Production TODO: Sui checkpoint verification
	//   1. Verify transaction inclusion in checkpoint
	//   2. Verify checkpoint committee signatures (BFT quorum)
	//   3. Verify object state changes match expected bridge events
	//   4. Wait for 3 checkpoint confirmations
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("sui deposit requires source tx digest")
	}
	return nil
}

// ValidateWithdrawal validates a Sui withdrawal request.
func (b *SuiBridge) ValidateWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Sui addresses are 32 bytes hex-encoded with 0x prefix (0x + 64 hex chars)
	if len(op.Receiver) != 66 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid Sui address format: %s", op.Receiver)
	}
	return nil
}

// EstimateConfirmationTime returns the expected confirmation time for Sui.
func (b *SuiBridge) EstimateConfirmationTime() int64 {
	// 3 checkpoints × ~3 seconds per checkpoint = ~9 seconds
	return 9
}

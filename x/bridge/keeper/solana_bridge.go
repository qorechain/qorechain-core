//go:build proprietary

package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// SolanaBridge handles Solana-specific bridge operations.
// Compatible with Wormhole-style light client verification.
type SolanaBridge struct{}

// NewSolanaBridge creates a new Solana bridge handler.
func NewSolanaBridge() *SolanaBridge {
	return &SolanaBridge{}
}

// ValidateDeposit validates a Solana deposit proof.
// In production, this verifies Solana VAAs (Verified Action Approvals).
func (b *SolanaBridge) ValidateDeposit(_ sdk.Context, op types.BridgeOperation) error {
	// Testnet: validation via PQC-signed attestations
	// Production TODO: Wormhole-compatible VAA verification
	//   1. Verify Guardian signatures on VAA
	//   2. Verify transaction finality (32 confirmations)
	//   3. Verify correct program ID and instruction data
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("solana deposit requires source tx signature")
	}
	return nil
}

// ValidateWithdrawal validates a Solana withdrawal request.
func (b *SolanaBridge) ValidateWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Verify Solana address format (base58, 32-44 chars)
	if len(op.Receiver) < 32 || len(op.Receiver) > 44 {
		return types.ErrInvalidDestination.Wrapf("invalid Solana address format: %s", op.Receiver)
	}
	return nil
}

// EstimateConfirmationTime returns the expected confirmation time for Solana.
func (b *SolanaBridge) EstimateConfirmationTime() int64 {
	// ~32 confirmations × 0.4 seconds per slot = ~13 seconds
	return 13
}

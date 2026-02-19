package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// EVMBridge handles generic EVM chain bridge operations (BSC, Avalanche, etc.).
// Shares the lock-mint model with Ethereum but with chain-specific parameters.
type EVMBridge struct {
	chainID string
}

// NewEVMBridge creates a new generic EVM bridge handler.
func NewEVMBridge(chainID string) *EVMBridge {
	return &EVMBridge{chainID: chainID}
}

// ValidateDeposit validates a deposit proof from an EVM chain.
func (b *EVMBridge) ValidateDeposit(_ sdk.Context, op types.BridgeOperation) error {
	// Testnet: validation via PQC-signed attestations
	// Production TODO: Chain-specific light client verification
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrapf("%s deposit requires source tx hash", b.chainID)
	}
	return nil
}

// ValidateWithdrawal validates a withdrawal to an EVM chain.
func (b *EVMBridge) ValidateWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Verify EVM address format (0x + 40 hex chars)
	if len(op.Receiver) != 42 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid EVM address format for %s: %s", b.chainID, op.Receiver)
	}
	return nil
}

// EstimateConfirmationTime returns the expected confirmation time for this EVM chain.
func (b *EVMBridge) EstimateConfirmationTime() int64 {
	// Default for generic EVM: ~15 confirmations × 3 seconds = ~45 seconds
	// Chain-specific values can override this
	switch b.chainID {
	case "bsc":
		return 45 // 15 confirmations × 3s
	case "avalanche":
		return 6 // 12 confirmations × 0.5s (C-Chain)
	default:
		return 45
	}
}

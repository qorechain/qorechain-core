//go:build proprietary

package keeper

import (
	"regexp"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// QCBHandler is the unified QoreChain Bridge handler.
// It dispatches internally by ChainType — all chain-specific validation,
// address checks, and confirmation estimates live in this single handler.
type QCBHandler struct{}

// NewQCBHandler creates a new unified QCB bridge handler.
func NewQCBHandler() *QCBHandler {
	return &QCBHandler{}
}

// ValidateDeposit validates a deposit proof based on the source chain type.
// The handler dispatches to the appropriate chain-specific validation logic.
func (h *QCBHandler) ValidateDeposit(ctx sdk.Context, op types.BridgeOperation, chainType types.ChainType) error {
	switch chainType {
	case types.ChainTypeEVM:
		return h.validateEVMDeposit(ctx, op)
	case types.ChainTypeSolana:
		return h.validateSolanaDeposit(ctx, op)
	case types.ChainTypeTON:
		return h.validateTONDeposit(ctx, op)
	case types.ChainTypeSui:
		return h.validateSuiDeposit(ctx, op)
	case types.ChainTypeAptos:
		return h.validateAptosDeposit(ctx, op)
	case types.ChainTypeBitcoin:
		return h.validateBitcoinDeposit(ctx, op)
	case types.ChainTypeNEAR:
		return h.validateNEARDeposit(ctx, op)
	case types.ChainTypeCardano:
		return h.validateCardanoDeposit(ctx, op)
	case types.ChainTypePolkadot:
		return h.validatePolkadotDeposit(ctx, op)
	case types.ChainTypeTezos:
		return h.validateTezosDeposit(ctx, op)
	case types.ChainTypeTron:
		return h.validateTronDeposit(ctx, op)
	case types.ChainTypeIBC:
		// IBC transfers handled by IBC module directly; no manual deposit validation needed
		return nil
	default:
		return types.ErrChainNotSupported.Wrapf("unsupported chain type: %s", chainType)
	}
}

// ValidateWithdrawal validates a withdrawal request based on the destination chain type.
// Checks destination address format for each chain family.
func (h *QCBHandler) ValidateWithdrawal(ctx sdk.Context, op types.BridgeOperation, chainType types.ChainType, chainID string) error {
	switch chainType {
	case types.ChainTypeEVM:
		return h.validateEVMWithdrawal(ctx, op, chainID)
	case types.ChainTypeSolana:
		return h.validateSolanaWithdrawal(ctx, op)
	case types.ChainTypeTON:
		return h.validateTONWithdrawal(ctx, op)
	case types.ChainTypeSui:
		return h.validateSuiWithdrawal(ctx, op)
	case types.ChainTypeAptos:
		return h.validateAptosWithdrawal(ctx, op)
	case types.ChainTypeBitcoin:
		return h.validateBitcoinWithdrawal(ctx, op)
	case types.ChainTypeNEAR:
		return h.validateNEARWithdrawal(ctx, op)
	case types.ChainTypeCardano:
		return h.validateCardanoWithdrawal(ctx, op)
	case types.ChainTypePolkadot:
		return h.validatePolkadotWithdrawal(ctx, op)
	case types.ChainTypeTezos:
		return h.validateTezosWithdrawal(ctx, op)
	case types.ChainTypeTron:
		return h.validateTronWithdrawal(ctx, op)
	case types.ChainTypeIBC:
		return nil
	default:
		return types.ErrChainNotSupported.Wrapf("unsupported chain type: %s", chainType)
	}
}

// EstimateConfirmationTime returns the expected confirmation time in seconds
// for the given chain type and chain ID.
func (h *QCBHandler) EstimateConfirmationTime(chainType types.ChainType, chainID string) int64 {
	switch chainType {
	case types.ChainTypeEVM:
		return h.estimateEVMConfirmationTime(chainID)
	case types.ChainTypeSolana:
		return 13 // ~32 confirmations × 0.4s per slot
	case types.ChainTypeTON:
		return 50 // ~10 confirmations × 5s per block
	case types.ChainTypeSui:
		return 9 // 3 checkpoints × ~3s per checkpoint
	case types.ChainTypeAptos:
		return 6 // 6 confirmations × ~1s per block
	case types.ChainTypeBitcoin:
		return 3600 // 6 confirmations × 600s (10 min) per block
	case types.ChainTypeNEAR:
		return 4 // 3 confirmations × ~1.3s per block
	case types.ChainTypeCardano:
		return 300 // 15 confirmations × 20s per block
	case types.ChainTypePolkadot:
		return 72 // 12 confirmations × 6s per block
	case types.ChainTypeTezos:
		return 30 // 2 confirmations × 15s per block
	case types.ChainTypeTron:
		return 60 // 20 confirmations × 3s per block
	case types.ChainTypeIBC:
		return 15 // Typically 1-2 block finality
	default:
		return 60 // Conservative default
	}
}

// ---- EVM chain validation ----

func (h *QCBHandler) validateEVMDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrapf("%s deposit requires source tx hash", op.SourceChain)
	}
	return nil
}

func (h *QCBHandler) validateEVMWithdrawal(_ sdk.Context, op types.BridgeOperation, chainID string) error {
	// EVM address: 0x + 40 hex chars
	if len(op.Receiver) != 42 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid EVM address format for %s: %s", chainID, op.Receiver)
	}
	return nil
}

func (h *QCBHandler) estimateEVMConfirmationTime(chainID string) int64 {
	switch chainID {
	case "ethereum":
		return 144 // 12 confirmations × 12s
	case "bsc":
		return 45 // 15 confirmations × 3s
	case "avalanche":
		return 6 // 12 confirmations × 0.5s (C-Chain)
	case "polygon":
		return 256 // 128 confirmations × 2s
	case "arbitrum":
		return 16 // 64 confirmations × 0.25s
	case "optimism":
		return 20 // 10 confirmations × 2s (L2 fast blocks)
	case "base":
		return 20 // 10 confirmations × 2s (L2 fast blocks)
	default:
		return 45 // Conservative default for unknown EVM chains
	}
}

// ---- Solana validation ----

func (h *QCBHandler) validateSolanaDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("solana deposit requires source tx signature")
	}
	return nil
}

func (h *QCBHandler) validateSolanaWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Solana address: base58, 32-44 chars
	if len(op.Receiver) < 32 || len(op.Receiver) > 44 {
		return types.ErrInvalidDestination.Wrapf("invalid Solana address format: %s", op.Receiver)
	}
	return nil
}

// ---- TON validation ----

func (h *QCBHandler) validateTONDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("TON deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateTONWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// TON uses raw or user-friendly addresses (at least 10 chars)
	if len(op.Receiver) < 10 {
		return types.ErrInvalidDestination.Wrapf("invalid TON address format: %s", op.Receiver)
	}
	return nil
}

// ---- Sui validation ----

func (h *QCBHandler) validateSuiDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("sui deposit requires source tx digest")
	}
	return nil
}

func (h *QCBHandler) validateSuiWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Sui address: 0x + 64 hex chars (32 bytes)
	if len(op.Receiver) != 66 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid Sui address format: %s", op.Receiver)
	}
	return nil
}

// ---- Aptos validation ----

func (h *QCBHandler) validateAptosDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("aptos deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateAptosWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	// Aptos address: 0x + 64 hex chars (32 bytes, like Sui but different chain)
	if len(op.Receiver) != 66 || op.Receiver[:2] != "0x" {
		return types.ErrInvalidDestination.Wrapf("invalid Aptos address format: %s", op.Receiver)
	}
	return nil
}

// ---- Bitcoin validation ----

var (
	// bech32Pattern matches bech32 addresses: bc1... or tb1... (mainnet/testnet)
	bech32Pattern = regexp.MustCompile(`^(bc1|tb1)[a-z0-9]{25,87}$`)
	// p2shPattern matches P2SH addresses: start with 3 or 2 (testnet)
	p2shPattern = regexp.MustCompile(`^[23][a-km-zA-HJ-NP-Z1-9]{25,34}$`)
	// legacyPattern matches legacy addresses: start with 1 or m/n (testnet)
	legacyPattern = regexp.MustCompile(`^[1mn][a-km-zA-HJ-NP-Z1-9]{25,34}$`)
)

func (h *QCBHandler) validateBitcoinDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("bitcoin deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateBitcoinWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// Accept bech32, P2SH, or legacy format
	if bech32Pattern.MatchString(addr) || p2shPattern.MatchString(addr) || legacyPattern.MatchString(addr) {
		return nil
	}
	return types.ErrInvalidDestination.Wrapf("invalid Bitcoin address format: %s", addr)
}

// ---- NEAR validation ----

func (h *QCBHandler) validateNEARDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("NEAR deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateNEARWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// NEAR account names: lowercase alphanumeric + _ + - + . , min 2, max 64 chars
	// Named accounts end with .near or .testnet; implicit accounts are 64 hex chars
	if len(addr) < 2 || len(addr) > 64 {
		return types.ErrInvalidDestination.Wrapf("invalid NEAR address format: %s", addr)
	}
	// Implicit accounts: exactly 64 hex chars
	if len(addr) == 64 {
		for _, c := range addr {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return types.ErrInvalidDestination.Wrapf("invalid NEAR implicit account: %s", addr)
			}
		}
		return nil
	}
	// Named accounts: alphanumeric + separators, must contain '.'
	if !strings.Contains(addr, ".") {
		return types.ErrInvalidDestination.Wrapf("invalid NEAR named account (no separator): %s", addr)
	}
	return nil
}

// ---- Cardano validation ----

func (h *QCBHandler) validateCardanoDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("cardano deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateCardanoWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// Cardano Shelley addresses: addr1... (mainnet) or addr_test1... (testnet)
	// Stake addresses: stake1... or stake_test1...
	if strings.HasPrefix(addr, "addr1") || strings.HasPrefix(addr, "addr_test1") ||
		strings.HasPrefix(addr, "stake1") || strings.HasPrefix(addr, "stake_test1") {
		// Bech32 Shelley address: typically 58-128 chars
		if len(addr) >= 40 && len(addr) <= 128 {
			return nil
		}
	}
	return types.ErrInvalidDestination.Wrapf("invalid Cardano address format: %s", addr)
}

// ---- Polkadot validation ----

func (h *QCBHandler) validatePolkadotDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("polkadot deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validatePolkadotWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// SS58 format: base58 encoded, typically 46-48 chars
	// Polkadot addresses start with 1, Kusama with letters
	if len(addr) < 40 || len(addr) > 50 {
		return types.ErrInvalidDestination.Wrapf("invalid Polkadot SS58 address format (length): %s", addr)
	}
	return nil
}

// ---- Tezos validation ----

func (h *QCBHandler) validateTezosDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("tezos deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateTezosWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// Tezos addresses: tz1 (Ed25519), tz2 (secp256k1), tz3 (P-256), KT1 (contract)
	if strings.HasPrefix(addr, "tz1") || strings.HasPrefix(addr, "tz2") ||
		strings.HasPrefix(addr, "tz3") || strings.HasPrefix(addr, "KT1") {
		// Base58check encoded, typically 36 chars
		if len(addr) >= 34 && len(addr) <= 40 {
			return nil
		}
	}
	return types.ErrInvalidDestination.Wrapf("invalid Tezos address format: %s", addr)
}

// ---- Tron validation ----

func (h *QCBHandler) validateTronDeposit(_ sdk.Context, op types.BridgeOperation) error {
	if op.SourceTxHash == "" {
		return types.ErrInvalidAttestation.Wrap("tron deposit requires source tx hash")
	}
	return nil
}

func (h *QCBHandler) validateTronWithdrawal(_ sdk.Context, op types.BridgeOperation) error {
	addr := op.Receiver
	// TRON addresses: T + base58, exactly 34 chars
	if len(addr) != 34 || addr[0] != 'T' {
		return types.ErrInvalidDestination.Wrapf("invalid TRON address format: %s", addr)
	}
	return nil
}

//go:build proprietary

package pqc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/keeper"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCVerifyDecorator sits in the AnteHandler chain before standard signature verification.
// It checks for PQC signatures on transactions and verifies them via the Rust FFI library.
// v0.6.0: Supports multi-algorithm dispatch based on the account's registered AlgorithmID.
type PQCVerifyDecorator struct {
	pqcKeeper keeper.Keeper
	ffiClient ffi.PQCClient
}

// NewPQCVerifyDecorator creates a new PQC verification ante handler decorator.
func NewPQCVerifyDecorator(k keeper.Keeper, client ffi.PQCClient) PQCVerifyDecorator {
	return PQCVerifyDecorator{
		pqcKeeper: k,
		ffiClient: client,
	}
}

func (d PQCVerifyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	params := d.pqcKeeper.GetParams(ctx)
	hybridMode := params.HybridSignatureMode

	// Extract signers from the transaction
	sigTx, ok := tx.(sdk.FeeTx)
	if !ok {
		// Not a fee transaction, skip PQC verification
		return next(ctx, tx, simulate)
	}

	msgs := sigTx.GetMsgs()
	for _, msg := range msgs {
		signers, _, err := d.getSigners(msg)
		if err != nil {
			continue
		}

		for _, signer := range signers {
			addr := sdk.AccAddress(signer).String()

			// Check if this account has a registered PQC key
			acct, hasPQC := d.pqcKeeper.GetPQCAccount(ctx, addr)
			if !hasPQC {
				// No PQC key registered - check if classical fallback is allowed
				if params.AllowClassicalFallback {
					d.pqcKeeper.IncrementClassicalFallbacks(ctx)
					continue
				}
				return ctx, types.ErrClassicalFallback.Wrap("account has no PQC key and classical fallback is disabled")
			}

			// Account has PQC key - verify based on key type
			switch acct.KeyType {
			case types.KeyTypeClassicalOnly:
				// Classical only - skip PQC verification, let standard handler manage
				continue

			case types.KeyTypeHybrid, types.KeyTypePQCOnly:
				// Verify the algorithm is still active or migrating
				algoStatus, err := d.checkAlgorithmStatus(ctx, acct.AlgorithmID)
				if err != nil {
					return ctx, err
				}

				switch algoStatus {
				case types.StatusActive, types.StatusMigrating:
					// Algorithm is operational — actual PQC signature verification
					// happens in PQCHybridVerifyDecorator. This decorator validates
					// algorithm status is acceptable for transaction processing.
					d.pqcKeeper.IncrementPQCVerifications(ctx)

					ctx.EventManager().EmitEvent(sdk.NewEvent(
						"pqc_verify",
						sdk.NewAttribute("address", addr),
						sdk.NewAttribute("key_type", acct.KeyType),
						sdk.NewAttribute("algorithm_id", acct.AlgorithmID.String()),
						sdk.NewAttribute("hybrid_mode", hybridMode.String()),
						sdk.NewAttribute("status", "algorithm_active"),
					))

				case types.StatusDeprecated:
					// Deprecated but still verifiable - warn via event
					d.pqcKeeper.IncrementPQCVerifications(ctx)

					ctx.EventManager().EmitEvent(sdk.NewEvent(
						"pqc_verify",
						sdk.NewAttribute("address", addr),
						sdk.NewAttribute("key_type", acct.KeyType),
						sdk.NewAttribute("algorithm_id", acct.AlgorithmID.String()),
						sdk.NewAttribute("hybrid_mode", hybridMode.String()),
						sdk.NewAttribute("status", "deprecated_warning"),
					))

				case types.StatusDisabled:
					// Algorithm has been disabled - reject unless classical fallback
					if acct.KeyType == types.KeyTypeHybrid && params.AllowClassicalFallback {
						d.pqcKeeper.IncrementClassicalFallbacks(ctx)

						ctx.EventManager().EmitEvent(sdk.NewEvent(
							"pqc_verify",
							sdk.NewAttribute("address", addr),
							sdk.NewAttribute("key_type", acct.KeyType),
							sdk.NewAttribute("algorithm_id", acct.AlgorithmID.String()),
							sdk.NewAttribute("hybrid_mode", hybridMode.String()),
							sdk.NewAttribute("status", "algorithm_disabled_fallback"),
						))
						continue
					}
					return ctx, types.ErrAlgorithmDisabled.Wrapf(
						"algorithm %s is disabled and no classical fallback available",
						acct.AlgorithmID,
					)
				}
			}
		}
	}

	return next(ctx, tx, simulate)
}

// checkAlgorithmStatus returns the current status of the specified algorithm.
// If the algorithm is not found in the registry, it defaults to Active for
// built-in algorithms (Dilithium-5, ML-KEM-1024) to maintain backward compatibility.
func (d PQCVerifyDecorator) checkAlgorithmStatus(ctx sdk.Context, id types.AlgorithmID) (types.AlgorithmStatus, error) {
	algo, err := d.pqcKeeper.GetAlgorithm(ctx, id)
	if err != nil {
		// For built-in algorithms not yet registered in the store (pre-v0.6.0 genesis),
		// treat as active for backward compatibility.
		if id == types.AlgorithmDilithium5 || id == types.AlgorithmMLKEM1024 {
			return types.StatusActive, nil
		}
		return 0, types.ErrInvalidAlgorithm.Wrapf("unknown algorithm %s", id)
	}
	return algo.Status, nil
}

// getSigners extracts signer addresses from a message.
func (d PQCVerifyDecorator) getSigners(msg sdk.Msg) ([][]byte, []string, error) {
	// In SDK v0.53, messages implement the HasGetSigners interface
	// or use the proto reflection-based signer extraction.
	type hasGetSigners interface {
		GetSigners() []sdk.AccAddress
	}
	if m, ok := msg.(hasGetSigners); ok {
		addrs := m.GetSigners()
		signers := make([][]byte, len(addrs))
		for i, addr := range addrs {
			signers[i] = addr
		}
		return signers, nil, nil
	}
	return nil, nil, nil
}

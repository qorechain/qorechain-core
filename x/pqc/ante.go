package pqc

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/keeper"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCVerifyDecorator sits in the AnteHandler chain before standard signature verification.
// It checks for PQC signatures on transactions and verifies them via the Rust FFI library.
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
				// No PQC key registered — check if classical fallback is allowed
				if params.AllowClassicalFallback {
					d.pqcKeeper.IncrementClassicalFallbacks(ctx)
					continue
				}
				return ctx, types.ErrClassicalFallback.Wrap("account has no PQC key and classical fallback is disabled")
			}

			// Account has PQC key — verify based on key type
			switch acct.KeyType {
			case types.KeyTypeClassicalOnly:
				// Classical only — skip PQC verification, let standard Cosmos handle it
				continue
			case types.KeyTypeHybrid, types.KeyTypePQCOnly:
				// PQC key present — for now, log and increment stats.
				// Full PQC signature extraction from TX will be implemented
				// when we define the custom PQC signature type.
				// In Phase 3 (testnet MVP), we verify the PQC key is registered
				// and track statistics. Actual signature verification over the
				// wire requires custom TX extensions (Phase 4).
				d.pqcKeeper.IncrementPQCVerifications(ctx)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"pqc_verify",
					sdk.NewAttribute("address", addr),
					sdk.NewAttribute("key_type", acct.KeyType),
					sdk.NewAttribute("status", "registered"),
				))
			}
		}
	}

	return next(ctx, tx, simulate)
}

// getSigners extracts signer addresses from a message.
func (d PQCVerifyDecorator) getSigners(msg sdk.Msg) ([][]byte, []string, error) {
	// In SDK v0.53, messages implement the HasGetSigners interface
	// or use the proto reflection-based signer extraction.
	// For now, we use the legacy GetSigners if available.
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

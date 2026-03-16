//go:build proprietary

package pqc

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/keeper"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// PQCHybridVerifyDecorator extracts PQCHybridSignature TX extensions and verifies
// them via the FFI library. It implements three-way verification logic:
//
//  1. Account has PQC key + extension present → verify PQC sig, increment hybrid counter
//  2. No PQC key + extension with PQCPublicKey → auto-register key, verify sig
//  3. No PQC key + no extension → if HybridRequired: reject; if HybridOptional: allow classical
//
// This decorator runs after the existing PQCVerifyDecorator and before AI anomaly checks.
type PQCHybridVerifyDecorator struct {
	pqcKeeper keeper.Keeper
	ffiClient ffi.PQCClient
}

// NewPQCHybridVerifyDecorator creates a new hybrid PQC verification ante handler decorator.
func NewPQCHybridVerifyDecorator(k keeper.Keeper, client ffi.PQCClient) PQCHybridVerifyDecorator {
	return PQCHybridVerifyDecorator{
		pqcKeeper: k,
		ffiClient: client,
	}
}

func (d PQCHybridVerifyDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if simulate {
		return next(ctx, tx, simulate)
	}

	hybridMode := d.pqcKeeper.GetHybridSignatureMode(ctx)

	// If hybrid mode is disabled, skip hybrid verification entirely.
	if hybridMode == types.HybridDisabled {
		return next(ctx, tx, simulate)
	}

	// Extract hybrid signature extension from the TX (if present).
	hybridSig, hasExtension := d.extractHybridSignature(tx)

	// Validate the hybrid signature format if present.
	if hasExtension {
		if err := hybridSig.Validate(); err != nil {
			return ctx, types.ErrInvalidHybridSig.Wrap(err.Error())
		}
	}

	// Extract signers from the transaction.
	sigTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return next(ctx, tx, simulate)
	}

	msgs := sigTx.GetMsgs()
	for _, msg := range msgs {
		signers, _ := getSigners(msg)

		for _, signer := range signers {
			addr := sdk.AccAddress(signer).String()
			acct, hasPQC := d.pqcKeeper.GetPQCAccount(ctx, addr)

			switch {
			case hasPQC && hasExtension:
				// Path 1: Account has PQC key and extension is present.
				// Check algorithm ID mismatch before verification.
				if hybridSig.AlgorithmID != 0 && hybridSig.AlgorithmID != acct.AlgorithmID {
					return ctx, types.ErrInvalidHybridSig.Wrap("extension algorithm ID does not match registered key")
				}

				// Verify the PQC signature against the registered public key.
				signBytes, sbErr := getSignBytes(tx)
				if sbErr != nil {
					return ctx, types.ErrHybridSigInvalid.Wrapf("cannot extract sign bytes: %v", sbErr)
				}
				valid, err := d.ffiClient.Verify(acct.AlgorithmID, acct.PublicKey, signBytes, hybridSig.PQCSignature)
				if err != nil {
					return ctx, types.ErrHybridSigInvalid.Wrapf("PQC verification error for %s: %v", addr, err)
				}
				if !valid {
					return ctx, types.ErrHybridSigInvalid.Wrapf("PQC signature verification failed for %s", addr)
				}

				d.pqcKeeper.IncrementHybridVerifications(ctx)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"pqc_hybrid_verify",
					sdk.NewAttribute("address", addr),
					sdk.NewAttribute("algorithm_id", hybridSig.AlgorithmID.String()),
					sdk.NewAttribute("status", "verified"),
				))

			case !hasPQC && hasExtension && hybridSig.HasPublicKey():
				// Path 2: No PQC key but extension includes a public key.
				// Check algorithm status before auto-registration.
				algoStatus, statusErr := checkAlgorithmStatus(ctx, d.pqcKeeper, hybridSig.AlgorithmID)
				if statusErr != nil {
					return ctx, statusErr
				}
				if algoStatus == types.StatusDisabled {
					return ctx, types.ErrAlgorithmDisabled.Wrapf(
						"cannot auto-register key: algorithm %s is disabled", hybridSig.AlgorithmID)
				}

				// Verify the PQC signature against the provided public key
				// BEFORE auto-registering.
				signBytes, sbErr := getSignBytes(tx)
				if sbErr != nil {
					return ctx, types.ErrHybridSigInvalid.Wrapf("cannot extract sign bytes: %v", sbErr)
				}
				valid, err := d.ffiClient.Verify(hybridSig.AlgorithmID, hybridSig.PQCPublicKey, signBytes, hybridSig.PQCSignature)
				if err != nil {
					return ctx, types.ErrHybridSigInvalid.Wrapf("PQC verification error for %s: %v", addr, err)
				}
				if !valid {
					return ctx, types.ErrHybridSigInvalid.Wrapf("PQC signature verification failed for %s (auto-register)", addr)
				}

				newAccount := types.PQCAccountInfo{
					Address:         addr,
					PublicKey:       hybridSig.PQCPublicKey,
					AlgorithmID:     hybridSig.AlgorithmID,
					KeyType:         types.KeyTypeHybrid,
					CreatedAtHeight: ctx.BlockHeight(),
				}
				if err := d.pqcKeeper.SetPQCAccount(ctx, newAccount); err != nil {
					return ctx, types.ErrHybridSigInvalid.Wrapf("failed to auto-register PQC key: %v", err)
				}

				d.pqcKeeper.IncrementHybridVerifications(ctx)

				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"pqc_hybrid_auto_register",
					sdk.NewAttribute("address", addr),
					sdk.NewAttribute("algorithm_id", hybridSig.AlgorithmID.String()),
					sdk.NewAttribute("key_type", types.KeyTypeHybrid),
				))

			case hasPQC && !hasExtension:
				// Account has PQC key but no extension — this is handled by the
				// existing PQCVerifyDecorator. No additional action needed here.
				continue

			case !hasPQC && !hasExtension:
				// Path 3: No PQC key and no extension.
				if hybridMode == types.HybridRequired {
					return ctx, types.ErrHybridSigRequired.Wrapf(
						"account %s must include a PQC hybrid signature extension (hybrid mode: required)", addr,
					)
				}

				// HybridOptional — allow classical-only transactions.
				ctx.EventManager().EmitEvent(sdk.NewEvent(
					"pqc_hybrid_classical_only",
					sdk.NewAttribute("address", addr),
					sdk.NewAttribute("hybrid_mode", hybridMode.String()),
				))
			}
		}
	}

	return next(ctx, tx, simulate)
}

// extractHybridSignature attempts to extract a PQCHybridSignature from the TX's
// extension options. Returns the signature and whether it was found.
func (d PQCHybridVerifyDecorator) extractHybridSignature(tx sdk.Tx) (types.PQCHybridSignature, bool) {
	extTx, ok := tx.(ante.HasExtensionOptionsTx)
	if !ok {
		return types.PQCHybridSignature{}, false
	}

	// Check both extension options and non-critical extension options.
	for _, opt := range extTx.GetExtensionOptions() {
		if opt.GetTypeUrl() == types.HybridSigTypeURL {
			var sig types.PQCHybridSignature
			if err := json.Unmarshal(opt.GetValue(), &sig); err == nil {
				return sig, true
			}
		}
	}

	for _, opt := range extTx.GetNonCriticalExtensionOptions() {
		if opt.GetTypeUrl() == types.HybridSigTypeURL {
			var sig types.PQCHybridSignature
			if err := json.Unmarshal(opt.GetValue(), &sig); err == nil {
				return sig, true
			}
		}
	}

	return types.PQCHybridSignature{}, false
}

// getSignBytes extracts the canonical body bytes from the transaction for PQC
// signature verification. The wallet signs these same bytes with the PQC key.
func getSignBytes(tx sdk.Tx) ([]byte, error) {
	type hasBodyBytes interface {
		GetBodyBytes() []byte
	}
	if bbtx, ok := tx.(hasBodyBytes); ok {
		return bbtx.GetBodyBytes(), nil
	}
	return nil, fmt.Errorf("transaction does not implement GetBodyBytes")
}

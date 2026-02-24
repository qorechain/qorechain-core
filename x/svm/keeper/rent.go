//go:build proprietary

package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// GetMinimumBalance returns the minimum lamports required for an account with
// the given data length to be rent-exempt.
//
//	minimumBalance = (128 + dataLen) * lamportsPerByte * rentExemptionMultiplier
//
// The 128-byte overhead covers the fixed account header.
func (k *Keeper) GetMinimumBalance(dataLen uint64) uint64 {
	params := types.DefaultParams()
	// NOTE: We use DefaultParams here because GetMinimumBalance has no context
	// in the SVMKeeper interface. For context-aware rent checks, CollectRent
	// reads the actual params from state.
	return computeMinimumBalance(dataLen, params.LamportsPerByte, params.RentExemptionMulti)
}

// computeMinimumBalance calculates the rent-exempt minimum balance.
func computeMinimumBalance(dataLen, lamportsPerByte uint64, exemptionMulti float64) uint64 {
	base := (128 + dataLen) * lamportsPerByte
	return uint64(float64(base) * exemptionMulti)
}

// IsRentExempt returns true if the account's lamport balance meets or exceeds
// the minimum balance for rent exemption.
func (k *Keeper) IsRentExempt(ctx sdk.Context, account *types.SVMAccount) bool {
	params := k.GetParams(ctx)
	minBalance := computeMinimumBalance(account.DataLen, params.LamportsPerByte, params.RentExemptionMulti)
	return account.Lamports >= minBalance
}

// CollectRent collects rent from a non-exempt account. If the account's
// lamport balance falls below the per-epoch rent cost the account is deleted.
// Rent-exempt accounts and executable (program) accounts are skipped.
func (k *Keeper) CollectRent(ctx sdk.Context, addr [32]byte) error {
	account, err := k.GetAccount(ctx, addr)
	if err != nil {
		return err
	}

	// Executable accounts (programs) are always rent-exempt.
	if account.Executable {
		return nil
	}

	params := k.GetParams(ctx)
	minBalance := computeMinimumBalance(account.DataLen, params.LamportsPerByte, params.RentExemptionMulti)

	// Already rent-exempt — nothing to do.
	if account.Lamports >= minBalance {
		return nil
	}

	// Calculate per-epoch rent: (128 + dataLen) * lamportsPerByte.
	rentPerEpoch := (128 + account.DataLen) * params.LamportsPerByte

	if account.Lamports < rentPerEpoch {
		// Insufficient balance to pay rent — garbage-collect the account.
		k.logger.Info("rent collection: account garbage-collected",
			"address", types.Base58Encode(addr),
			"lamports", account.Lamports,
			"rent_due", rentPerEpoch,
		)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"svm_rent_gc",
			sdk.NewAttribute("address", types.Base58Encode(addr)),
			sdk.NewAttribute("lamports", fmt.Sprintf("%d", account.Lamports)),
		))

		return k.DeleteAccount(ctx, addr)
	}

	// Deduct rent.
	account.Lamports -= rentPerEpoch
	account.RentEpoch++

	if err := k.SetAccount(ctx, account); err != nil {
		return fmt.Errorf("failed to update account after rent collection: %w", err)
	}

	k.logger.Debug("rent collected",
		"address", types.Base58Encode(addr),
		"rent", rentPerEpoch,
		"remaining", account.Lamports,
	)

	return nil
}

package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Native-QOR ↔ lamport unification.
//
// QoreChain has a single native asset (QOR / uqor, 6 decimals). The SVM surface
// presents balances in lamports (Solana's 9-decimal convention). We fix an exact
// ratio so a wallet's SVM balance and its x/bank balance are the SAME value:
//
//	1 QOR   = 1_000_000 uqor            (chain base, 6 decimals)
//	1 QOR   = 1_000_000_000 lamports    (SVM convention, 9 decimals)
//	1 uqor  = 1_000 lamports            (LamportsPerUqor)
//
// Sub-uqor "dust" (0..999 lamports that do not fill a whole uqor) is tracked in
// the SVM store's dust ledger and reconciled up to whole uqor on settlement, so
// no value is lost when moving between the ledgers.
const (
	// NativeDenom is the chain base denom that backs SVM lamports.
	NativeDenom = "uqor"
	// LamportsPerUqor is the fixed conversion ratio (10^9 / 10^6).
	LamportsPerUqor = 1000
)

// BankKeeper defines the x/bank methods the SVM module uses to back native
// (wallet) account balances with real QOR, so one account holds a single
// balance usable from Cosmos, EVM and SVM alike. The concrete
// bankkeeper.BaseKeeper satisfies this interface.
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoins(ctx context.Context, from, to sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, from sdk.AccAddress, module string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error
}

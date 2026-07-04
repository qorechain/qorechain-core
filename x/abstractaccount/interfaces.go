package abstractaccount

import (
	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// AbstractAccountKeeper is the interface for the x/abstractaccount module keeper.
type AbstractAccountKeeper interface {
	Logger() log.Logger

	// Config
	GetConfig(ctx sdk.Context) types.AbstractAccountConfig
	SetConfig(ctx sdk.Context, config types.AbstractAccountConfig) error
	IsEnabled(ctx sdk.Context) bool

	// Accounts
	GetAccount(ctx sdk.Context, address string) (types.AbstractAccount, bool)
	SetAccount(ctx sdk.Context, acc types.AbstractAccount) error
	GetAllAccounts(ctx sdk.Context) []types.AbstractAccount

	// Authenticator resolution (consumed by x/svm via a primitive interface, so
	// no cross-module type import is needed): maps a foreign-scheme wallet key
	// (e.g. Phantom ed25519) to the canonical account it acts for + verifies sigs.
	ResolveAuthenticatorAddr(ctx sdk.Context, scheme string, pubkey []byte) (account []byte, permissions []string, ok bool)
	VerifyForeignSignature(scheme string, pubkey, msg, sig []byte) bool

	// AuthorizeAction (v3.1.84) is the single authorization gate for an
	// authenticator-authorized action: verify sig → resolve active authenticator
	// → check requiredPerm (canonical taxonomy, fail-closed) → enforce SpendingRule
	// against the base-unit outflow (uqor) → return the canonical account.
	AuthorizeAction(ctx sdk.Context, scheme string, pubkey, msg, sig []byte, requiredPerm string, outflow sdk.Coins) (account []byte, err error)

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

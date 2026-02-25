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

	// Genesis
	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState
}

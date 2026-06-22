package bridge

import (
	"context"

	"cosmossdk.io/log"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// BridgeKeeper is the interface for the x/bridge module's keeper.
type BridgeKeeper interface {
	Logger() log.Logger

	GetConfig(ctx sdk.Context) types.BridgeConfig
	SetConfig(ctx sdk.Context, config types.BridgeConfig) error
	GetChainConfig(ctx sdk.Context, chainID string) (types.ChainConfig, bool)
	SetChainConfig(ctx sdk.Context, cc types.ChainConfig) error
	GetAllChainConfigs(ctx sdk.Context) []types.ChainConfig
	GetBridgeValidator(ctx sdk.Context, address string) (types.BridgeValidator, bool)
	SetBridgeValidator(ctx sdk.Context, v types.BridgeValidator) error
	GetAllBridgeValidators(ctx sdk.Context) []types.BridgeValidator
	GetActiveValidatorsForChain(ctx sdk.Context, chainID string) []types.BridgeValidator
	GetOperation(ctx sdk.Context, id string) (types.BridgeOperation, bool)
	SetOperation(ctx sdk.Context, op types.BridgeOperation) error
	GetAllOperations(ctx sdk.Context) []types.BridgeOperation
	NextOperationID(ctx sdk.Context) string
	GetLockedAmount(ctx sdk.Context, chain, asset string) types.LockedAmount
	SetLockedAmount(ctx sdk.Context, la types.LockedAmount) error
	GetAllLockedAmounts(ctx sdk.Context) []types.LockedAmount
	GetCircuitBreaker(ctx sdk.Context, chain string) types.CircuitBreakerState
	SetCircuitBreaker(ctx sdk.Context, cb types.CircuitBreakerState) error
	GetAllCircuitBreakers(ctx sdk.Context) []types.CircuitBreakerState

	InitGenesis(ctx sdk.Context, gs types.GenesisState)
	ExportGenesis(ctx sdk.Context) *types.GenesisState

	// SetLicenseChecker wires the license keeper post-construction so that
	// bridge-validator registration is gated on per-chain validator licenses.
	SetLicenseChecker(lc BridgeLicenseChecker)

	// SetBankKeeper wires the bank keeper post-construction so the bridge can
	// actually mint/burn bridged tokens on verified deposits/withdrawals.
	SetBankKeeper(bk BridgeBankKeeper)
}

// BridgeLicenseChecker is the license surface the bridge module consumes.
type BridgeLicenseChecker interface {
	HasActiveLicense(ctx sdk.Context, grantee, featureID string) bool
}

// BridgeBankKeeper is the bank surface the bridge module consumes (matches the
// SDK bank keeper; sdk.Context satisfies context.Context).
type BridgeBankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

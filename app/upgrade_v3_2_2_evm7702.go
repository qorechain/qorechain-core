package app

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// upgradeV3_2_2EVM7702 is this area's one-time work at the v3.2.2 plan height.
//
// The EIP-7702 fixes need no state preparation: the consent store starts empty,
// and delegated-actor settlement keys on the code accounts carry at execution
// time, so a delegation installed before the plan height is covered from the
// first block after it. What operators do need to know is whether any account
// already carries a delegation, because from this height such an account can
// only act, or lose value, under its own post-quantum window. The scan logs
// every one it finds; it writes nothing, so it cannot affect consensus.
func (app *QoreChainApp) upgradeV3_2_2EVM7702(ctx sdk.Context) error {
	if app.EVMKeeper == nil {
		return nil
	}
	found := 0
	app.EVMKeeper.IterateContracts(ctx, func(addr common.Address, codeHash common.Hash) bool {
		if target, ok := ethtypes.ParseDelegation(app.EVMKeeper.GetCode(ctx, codeHash)); ok {
			found++
			ctx.Logger().Info("v3.2.2: account carries an EIP-7702 delegation; from this height it acts only under its own window",
				"account", addr.Hex(), "delegate", target.Hex())
		}
		return false
	})
	ctx.Logger().Info("v3.2.2: EIP-7702 delegation scan done", "delegated_accounts", found)
	return nil
}

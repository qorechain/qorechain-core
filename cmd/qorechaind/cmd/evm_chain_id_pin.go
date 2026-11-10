package cmd

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/server"

	"github.com/qorechain/qorechain-core/app"
)

// wireRPCEVMChainIDPin makes the start command serve JSON-RPC with the
// keeper's EVM chain ID (app.PinRPCEVMChainID) instead of whatever app.toml
// says. It runs before the server reads its configuration. Node-local: the
// keeper used the pinned value already, so consensus is unchanged.
func wireRPCEVMChainIDPin(rootCmd *cobra.Command) {
	for _, c := range rootCmd.Commands() {
		if c.Name() != "start" {
			continue
		}
		prev := c.PreRunE
		c.PreRunE = func(cmd *cobra.Command, args []string) error {
			if prev != nil {
				if err := prev(cmd, args); err != nil {
					return err
				}
			}
			serverCtx := server.GetServerContextFromCmd(cmd)
			if from, to, changed := app.PinRPCEVMChainID(serverCtx.Viper); changed {
				serverCtx.Logger.Error("app.toml [evm] evm-chain-id does not match this network; "+
					"JSON-RPC will use the network's EVM chain ID, fix app.toml",
					"app_toml", from, "evm_chain_id", to)
			}
			return nil
		}
	}
}

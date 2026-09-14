package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetCmdQueryEVMWindow returns the command that shows an address's EVM-lane
// authorisation window.
func GetCmdQueryEVMWindow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evm-window [address]",
		Short: "Show an address's EVM-lane authorisation window and what remains of it",
		Long: `Show whether an address has an open post-quantum authorisation window for
the EVM lane, and how much of it is left: blocks, transactions and value.

An address with no window, or an exhausted one, cannot send EVM transactions.
Open one with 'tx pqc open-evm-window'.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).EVMWindow(cmd.Context(),
				&types.QueryEVMWindowRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

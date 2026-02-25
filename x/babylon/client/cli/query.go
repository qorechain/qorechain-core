package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// GetQueryCmd returns the CLI query commands for the babylon module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the babylon module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryConfig(),
		CmdQueryStakingPosition(),
	)

	return cmd
}

// CmdQueryConfig returns the command to query the babylon module config.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the BTC restaking configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			return cmd.Help()
		},
	}
	return cmd
}

// CmdQueryStakingPosition returns the command to query a BTC staking position.
func CmdQueryStakingPosition() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "position [staker-address]",
		Short: "Query a BTC restaking position by staker address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			_ = args[0]
			return cmd.Help()
		},
	}
	return cmd
}

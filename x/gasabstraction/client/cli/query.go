package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// GetQueryCmd returns the CLI query commands for the gasabstraction module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the gasabstraction module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryConfig(),
		CmdQueryAcceptedTokens(),
	)

	return cmd
}

// CmdQueryConfig returns the command to query the gas abstraction config.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the gas abstraction configuration",
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

// CmdQueryAcceptedTokens returns the command to list accepted fee tokens.
func CmdQueryAcceptedTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accepted-tokens",
		Short: "List all accepted fee tokens and their conversion rates",
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

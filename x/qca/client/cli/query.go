package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/qca/types"
)

// GetQueryCmd returns the CLI query commands for the qca module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the qca module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryConfig())

	return cmd
}

// CmdQueryConfig returns the command to query the QCA module configuration.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the current QCA module configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			route := fmt.Sprintf("custom/%s/config", types.ModuleName)
			resBz, _, err := clientCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var cfg types.QCAConfig
			if err := json.Unmarshal(resBz, &cfg); err != nil {
				return fmt.Errorf("failed to unmarshal config: %w", err)
			}

			return clientCtx.PrintObjectLegacy(cfg)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

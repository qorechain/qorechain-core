package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// GetQueryCmd returns the CLI query commands for the reputation module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the reputation module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdQueryParams())

	return cmd
}

// CmdQueryParams returns the command to query reputation module parameters.
func CmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current reputation module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			route := fmt.Sprintf("custom/%s/params", types.ModuleName)
			resBz, _, err := clientCtx.QueryWithData(route, nil)
			if err != nil {
				return err
			}

			var params types.ReputationParams
			if err := json.Unmarshal(resBz, &params); err != nil {
				return fmt.Errorf("failed to unmarshal params: %w", err)
			}

			return clientCtx.PrintObjectLegacy(params)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetQueryCmd returns the CLI query commands for the rdk module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the RDK (Rollup Development Kit) module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryRollup(),
		CmdQueryListRollups(),
		CmdQueryBatch(),
		CmdQueryConfig(),
		CmdSuggestProfile(),
	)
	return cmd
}

// CmdQueryRollup queries a specific rollup by ID.
func CmdQueryRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollup [rollup-id]",
		Short: "Query a rollup by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Rollup(cmd.Context(), &types.QueryRollupRequest{RollupId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryListRollups lists all registered rollups.
func CmdQueryListRollups() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-rollups",
		Short: "List all rollups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Rollups(cmd.Context(), &types.QueryRollupsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryBatch queries a settlement batch (latest, or a specific --index).
func CmdQueryBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [rollup-id]",
		Short: "Query a settlement batch (latest by default, or --index)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			qc := types.NewQueryClient(clientCtx)
			index, _ := cmd.Flags().GetInt64("index")
			if index >= 0 {
				res, err := qc.Batch(cmd.Context(), &types.QueryBatchRequest{RollupId: args[0], BatchIndex: uint64(index)})
				if err != nil {
					return err
				}
				return clientCtx.PrintProto(res)
			}
			res, err := qc.LatestBatch(cmd.Context(), &types.QueryLatestBatchRequest{RollupId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Int64("index", -1, "Batch index (default: latest)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryConfig queries the RDK module parameters.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query RDK module parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdSuggestProfile queries the AI-suggested rollup profile.
func CmdSuggestProfile() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suggest-profile [use-case]",
		Short: "Get an AI-suggested rollup profile for a use case",
		Long:  "Returns a recommended rollup configuration profile (defi, gaming, nft, enterprise) based on the specified use case.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := client.GetClientQueryContext(cmd); err != nil {
				return err
			}
			fmt.Printf("Suggested profile for '%s': use the qor_suggestRollupProfile RPC\n", args[0])
			return nil
		},
	}
	return cmd
}

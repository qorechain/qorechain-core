package cli

import (
	"fmt"
	"strconv"

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
		CmdQueryBatches(),
		CmdAnchorSignBytes(),
		CmdQueryPostedData(),
		CmdQueryInbox(),
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

// CmdQueryPostedData queries one posted-data commitment (chain v3.2.8+).
func CmdQueryPostedData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "posted-data [rollup-id] [sequence]",
		Short: "Query a posted-data commitment by inbox sequence (the payload is in the transaction at its height)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			seq, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).PostedData(cmd.Context(),
				&types.QueryPostedDataRequest{RollupId: args[0], Sequence: seq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryInbox queries a rollup's inbox and the data_hash a based batch must
// carry (chain v3.2.8+).
func CmdQueryInbox() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inbox [rollup-id]",
		Short: "Query a rollup's inbox and the range data_hash for a based batch ending at --inbox-end (default the head)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			end, _ := cmd.Flags().GetUint64("inbox-end")
			res, err := types.NewQueryClient(clientCtx).Inbox(cmd.Context(),
				&types.QueryInboxRequest{RollupId: args[0], InboxEnd: end})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint64("inbox-end", 0, "range end for range_data_hash (0 = the head)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

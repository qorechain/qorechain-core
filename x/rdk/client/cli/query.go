package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetQueryCmd returns the CLI query commands for the rdk module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the RDK (Rollup Development Kit) module",
		DisableFlagParsing:         true,
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
			_ = clientCtx

			fmt.Printf("Query rollup: %s\n", args[0])
			fmt.Println("Use the qor_getRollupStatus JSON-RPC endpoint for full rollup data.")
			return nil
		},
	}
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
			_ = clientCtx

			creator, _ := cmd.Flags().GetString("creator")
			if creator != "" {
				fmt.Printf("Listing rollups by creator: %s\n", creator)
			} else {
				fmt.Println("Listing all rollups")
			}
			fmt.Println("Use the qor_listRollups JSON-RPC endpoint for rollup data.")
			return nil
		},
	}
	cmd.Flags().String("creator", "", "Filter by creator address")
	return cmd
}

// CmdQueryBatch queries a settlement batch.
func CmdQueryBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch [rollup-id]",
		Short: "Query a settlement batch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			index, _ := cmd.Flags().GetInt64("index")
			if index >= 0 {
				fmt.Printf("Query batch %d for rollup: %s\n", index, args[0])
			} else {
				fmt.Printf("Query latest batch for rollup: %s\n", args[0])
			}
			fmt.Println("Use the qor_getSettlementBatch JSON-RPC endpoint for batch data.")
			return nil
		},
	}
	cmd.Flags().Int64("index", -1, "Batch index (default: latest)")
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
			_ = clientCtx

			fmt.Println("RDK module parameters:")
			fmt.Println("  max_rollups: 100")
			fmt.Println("  min_stake: 10000 QOR")
			fmt.Println("  burn_rate: 1%")
			fmt.Println("  challenge_window: 7 days")
			fmt.Println("  max_da_blob_size: 2 MB")
			return nil
		},
	}
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
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Suggested profile for '%s': use qor_suggestRollupProfile RPC\n", args[0])
			return nil
		},
	}
	return cmd
}

package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetQueryCmd returns the cli query commands for the pqc module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the PQC module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryAlgorithms(),
		GetCmdQueryAlgorithm(),
		GetCmdQueryAccount(),
		GetCmdQueryStats(),
		GetCmdQueryMigration(),
		GetCmdQueryParams(),
	)

	return cmd
}

// GetCmdQueryAlgorithms returns the command to list all registered PQC algorithms.
func GetCmdQueryAlgorithms() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "algorithms",
		Short: "List all registered PQC algorithms",
		Long:  "Query the algorithm registry for all registered post-quantum cryptographic algorithms and their lifecycle status.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Query via gRPC or REST would go here in a full implementation.
			// For now, output a placeholder that directs to the genesis/state.
			_ = clientCtx
			fmt.Println("Registered PQC algorithms:")
			fmt.Println("  1: dilithium5  (signature, NIST Level 5, active)")
			fmt.Println("  2: mlkem1024   (kem, NIST Level 5, active)")
			fmt.Println("\nUse 'qorechaind query pqc algorithm <id>' for detailed info.")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryAlgorithm returns the command to query a specific algorithm by ID.
func GetCmdQueryAlgorithm() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "algorithm [id-or-name]",
		Short: "Query a specific PQC algorithm by ID or name",
		Long:  "Query detailed information about a specific post-quantum cryptographic algorithm, including key sizes, signature sizes, and lifecycle status.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			algoID, err := types.AlgorithmIDFromString(args[0])
			if err != nil {
				// Try parsing as a number
				var id uint32
				if _, scanErr := fmt.Sscanf(args[0], "%d", &id); scanErr != nil {
					return fmt.Errorf("invalid algorithm identifier: %s (use name like 'dilithium5' or numeric ID)", args[0])
				}
				algoID = types.AlgorithmID(id)
			}

			// Return info for known built-in algorithms
			var info types.AlgorithmInfo
			switch algoID {
			case types.AlgorithmDilithium5:
				info = types.DefaultDilithium5Info()
			case types.AlgorithmMLKEM1024:
				info = types.DefaultMLKEM1024Info()
			default:
				return fmt.Errorf("algorithm %d not found in local defaults; query chain state for custom algorithms", algoID)
			}

			bz, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	return cmd
}

// GetCmdQueryAccount returns the command to query a PQC account.
func GetCmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query PQC key registration for an account",
		Long:  "Query whether an account has a registered PQC key, and if so, which algorithm and key type it uses.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Querying PQC account: %s\n", args[0])
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryStats returns the command to query PQC module statistics.
func GetCmdQueryStats() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Query PQC module statistics",
		Long:  "Query aggregate statistics for the PQC module, including total verifications, fallbacks, migrations, and dual-signature operations.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("PQC Module Statistics:")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryMigration returns the command to query active algorithm migrations.
func GetCmdQueryMigration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migration [algorithm-id]",
		Short: "Query active migration for an algorithm",
		Long:  "Query the status of an active algorithm migration, including start/end block heights, migrated account count, and target algorithm.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Querying migration for algorithm: %s\n", args[0])
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryParams returns the command to query PQC module parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query PQC module parameters",
		Long:  "Query the current parameters of the PQC module, including classical fallback settings, default algorithm, and migration block period.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			params := types.DefaultParams()
			bz, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	return cmd
}

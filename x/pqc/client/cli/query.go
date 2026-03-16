package cli

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

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
		GetCmdQueryHybridMode(),
	)

	return cmd
}

// queryStoreKey performs a direct ABCI store query for the given key.
func queryStoreKey(clientCtx client.Context, key []byte) ([]byte, error) {
	resp, err := clientCtx.QueryABCI(abci.RequestQuery{
		Path: fmt.Sprintf("store/%s/key", types.StoreKey),
		Data: key,
	})
	if err != nil {
		return nil, err
	}
	return resp.Value, nil
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

			// Query all algorithms by iterating over the algorithm prefix.
			// The subspace query may not be supported; fall back to listing known IDs.
			var algos []types.AlgorithmInfo

			// Query known algorithm IDs individually
			for _, id := range []types.AlgorithmID{types.AlgorithmDilithium5, types.AlgorithmMLKEM1024} {
				bz, qErr := queryStoreKey(clientCtx, types.AlgorithmKey(id))
				if qErr != nil || len(bz) == 0 {
					continue
				}
				var algo types.AlgorithmInfo
				if jErr := json.Unmarshal(bz, &algo); jErr != nil {
					continue
				}
				algos = append(algos, algo)
			}

			if len(algos) == 0 {
				fmt.Println("No algorithms found in on-chain state (node may be unreachable or genesis not yet applied).")
				return nil
			}

			bz, err := json.MarshalIndent(algos, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
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

			algoID, err := types.AlgorithmIDFromString(args[0])
			if err != nil {
				// Try parsing as a number
				var id uint32
				if _, scanErr := fmt.Sscanf(args[0], "%d", &id); scanErr != nil {
					return fmt.Errorf("invalid algorithm identifier: %s (use name like 'dilithium5' or numeric ID)", args[0])
				}
				algoID = types.AlgorithmID(id)
			}

			bz, err := queryStoreKey(clientCtx, types.AlgorithmKey(algoID))
			if err != nil {
				return fmt.Errorf("failed to query algorithm %d: %w", algoID, err)
			}
			if len(bz) == 0 {
				return fmt.Errorf("algorithm %d not found in on-chain state", algoID)
			}

			var info types.AlgorithmInfo
			if err := json.Unmarshal(bz, &info); err != nil {
				return fmt.Errorf("failed to decode algorithm info: %w", err)
			}

			out, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
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

			key := append(types.AccountPrefix, []byte(args[0])...)
			bz, err := queryStoreKey(clientCtx, key)
			if err != nil {
				return fmt.Errorf("failed to query account %s: %w", args[0], err)
			}
			if len(bz) == 0 {
				fmt.Printf("No PQC key registered for account: %s\n", args[0])
				return nil
			}

			var acct types.PQCAccountInfo
			if err := json.Unmarshal(bz, &acct); err != nil {
				return fmt.Errorf("failed to decode account info: %w", err)
			}

			out, err := json.MarshalIndent(acct, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
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

			bz, err := queryStoreKey(clientCtx, types.StatsKey)
			if err != nil {
				return fmt.Errorf("failed to query stats: %w", err)
			}
			if len(bz) == 0 {
				fmt.Println("No PQC statistics recorded yet.")
				return nil
			}

			// Decode stats: binary (48 bytes) or JSON fallback
			var stats types.PQCStats
			if len(bz) == 48 {
				stats = types.PQCStats{
					TotalPQCVerifications:    binary.LittleEndian.Uint64(bz[0:8]),
					TotalClassicalFallbacks:  binary.LittleEndian.Uint64(bz[8:16]),
					TotalMLKEMOperations:     binary.LittleEndian.Uint64(bz[16:24]),
					TotalDualSigVerifies:     binary.LittleEndian.Uint64(bz[24:32]),
					TotalKeyMigrations:       binary.LittleEndian.Uint64(bz[32:40]),
					TotalHybridVerifications: binary.LittleEndian.Uint64(bz[40:48]),
				}
			} else {
				if err := json.Unmarshal(bz, &stats); err != nil {
					return fmt.Errorf("failed to decode stats: %w", err)
				}
			}

			out, err := json.MarshalIndent(stats, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
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

			algoID, err := types.AlgorithmIDFromString(args[0])
			if err != nil {
				var id uint32
				if _, scanErr := fmt.Sscanf(args[0], "%d", &id); scanErr != nil {
					return fmt.Errorf("invalid algorithm identifier: %s", args[0])
				}
				algoID = types.AlgorithmID(id)
			}

			bz, err := queryStoreKey(clientCtx, types.MigrationKey(algoID))
			if err != nil {
				return fmt.Errorf("failed to query migration for algorithm %d: %w", algoID, err)
			}
			if len(bz) == 0 {
				fmt.Printf("No active migration for algorithm %d.\n", algoID)
				return nil
			}

			var mig types.MigrationInfo
			if err := json.Unmarshal(bz, &mig); err != nil {
				return fmt.Errorf("failed to decode migration info: %w", err)
			}

			out, err := json.MarshalIndent(mig, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryHybridMode returns the command to query the current hybrid signature mode.
func GetCmdQueryHybridMode() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hybrid-mode",
		Short: "Query the current hybrid signature mode",
		Long: `Query the PQC module's hybrid signature mode which controls how dual
Ed25519 + ML-DSA-87 signatures are handled:

  0 (disabled)  — Only classical signatures accepted; PQC extensions ignored.
  1 (optional)  — PQC extensions verified if present; classical-only allowed.
  2 (required)  — Both classical and PQC signatures mandatory on all transactions.

The mode is set via governance through PQC module parameters.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			bz, err := queryStoreKey(clientCtx, types.ParamsKey)
			if err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			var params types.Params
			if len(bz) == 0 {
				// Fall back to defaults if no on-chain params
				params = types.DefaultParams()
			} else {
				if err := json.Unmarshal(bz, &params); err != nil {
					return fmt.Errorf("failed to decode params: %w", err)
				}
			}

			mode := params.HybridSignatureMode
			fmt.Printf("Hybrid Signature Mode: %d (%s)\n", mode, mode.String())
			fmt.Printf("Description: %s\n", mode.Description())
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
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

			bz, err := queryStoreKey(clientCtx, types.ParamsKey)
			if err != nil {
				return fmt.Errorf("failed to query params: %w", err)
			}

			var params types.Params
			if len(bz) == 0 {
				params = types.DefaultParams()
			} else {
				if err := json.Unmarshal(bz, &params); err != nil {
					return fmt.Errorf("failed to decode params: %w", err)
				}
			}

			out, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

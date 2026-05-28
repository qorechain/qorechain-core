package cli

import (
	"github.com/spf13/cobra"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// GetQueryCmd returns the AMM module's query subcommand tree.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ammtypes.ModuleName,
		Short:                      "Querying commands for the AMM module",
		SuggestionsMinimumDistance: 2,
		RunE:                       func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	cmd.AddCommand(
		queryPlaceholder("params", "Show the current AMM module params"),
		queryPlaceholder("pool", "Show a single pool by ID"),
		queryPlaceholder("pools", "List all pools (paginated)"),
		queryPlaceholder("pool-by-denoms", "Find a pool by its (denomA, denomB) tuple"),
		queryPlaceholder("lp-balance", "Query an account's LP balance for a pool"),
		queryPlaceholder("quote-exact-in", "Quote the output for a fixed-input swap"),
		queryPlaceholder("quote-exact-out", "Quote the input required for a fixed-output swap"),
	)
	return cmd
}

func queryPlaceholder(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("AMM query CLI is wired but proto-bound query handlers are not yet generated in this build.")
			return nil
		},
	}
}

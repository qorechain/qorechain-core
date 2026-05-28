package cli

import (
	"github.com/spf13/cobra"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// GetTxCmd returns the AMM module's transaction subcommand tree.
//
// Subcommands are placeholders that will be wired to the proto-generated
// MsgClient once buf-generate runs. Until then they print a notice so
// operators can discover the surface area.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ammtypes.ModuleName,
		Short:                      "AMM transaction subcommands",
		SuggestionsMinimumDistance: 2,
		RunE:                       func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}

	cmd.AddCommand(
		placeholder("create-pool", "Create a new liquidity pool"),
		placeholder("add-liquidity", "Add proportional liquidity to a pool"),
		placeholder("remove-liquidity", "Burn LP tokens and withdraw reserves"),
		placeholder("swap-exact-in", "Swap a fixed input amount, enforcing a minimum output"),
		placeholder("swap-exact-out", "Swap to a fixed output amount, enforcing a maximum input"),
	)
	return cmd
}

func placeholder(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Println("AMM CLI is wired but proto-bound message handlers are not yet generated in this build.")
			return nil
		},
	}
}

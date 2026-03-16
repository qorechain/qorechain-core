package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// GetTxCmd returns the transaction commands for the reputation module.
// The reputation module has no user-facing transactions; scores are
// computed automatically by the EndBlocker.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the reputation module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	return cmd
}

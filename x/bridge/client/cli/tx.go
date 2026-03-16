package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// GetTxCmd returns the transaction commands for the bridge module.
// Bridge operations are relayer-driven; user-facing TX commands will
// be added when the deposit/withdraw message types are finalized.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the bridge module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	return cmd
}

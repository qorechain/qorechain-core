package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/babylon/types"
)

// GetTxCmd returns the CLI transaction commands for the babylon module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the babylon module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdSubmitCheckpoint(),
		CmdBTCRestake(),
	)

	return cmd
}

// CmdSubmitCheckpoint returns the command to submit a BTC checkpoint.
func CmdSubmitCheckpoint() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-checkpoint [epoch] [checkpoint-hash]",
		Short: "Submit a BTC checkpoint for a given epoch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			return cmd.Help()
		},
	}
	return cmd
}

// CmdBTCRestake returns the command to restake BTC.
func CmdBTCRestake() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restake [btc-tx-hash] [amount]",
		Short: "Restake BTC via a finalized Bitcoin transaction",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			return cmd.Help()
		},
	}
	return cmd
}

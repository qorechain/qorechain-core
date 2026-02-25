package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// GetTxCmd returns the CLI transaction commands for the abstractaccount module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the abstractaccount module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdCreateAbstractAccount(),
		CmdUpdateSpendingRules(),
	)

	return cmd
}

// CmdCreateAbstractAccount returns the command to create an abstract account.
func CmdCreateAbstractAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [contract-address] [account-type]",
		Short: "Create a new abstract account linked to a smart contract",
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

// CmdUpdateSpendingRules returns the command to update spending rules.
func CmdUpdateSpendingRules() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-rules [address]",
		Short: "Update spending rules for an abstract account",
		Args:  cobra.ExactArgs(1),
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

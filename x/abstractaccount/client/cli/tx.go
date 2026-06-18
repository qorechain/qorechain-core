package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// GetTxCmd returns the CLI transaction commands for the abstractaccount module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the abstractaccount module",
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
		Use:   "create [account-type]",
		Short: "Create a new abstract account (type: multisig|social_recovery|session_based)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgCreateAbstractAccount{
				Owner:       clientCtx.GetFromAddress().String(),
				AccountType: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateSpendingRules returns the command to update spending rules.
func CmdUpdateSpendingRules() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-rules [account-address] [rules-json]",
		Short: "Update spending rules (rules-json: array of SpendingRule objects)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			var rules []types.SpendingRule
			if err := json.Unmarshal([]byte(args[1]), &rules); err != nil {
				return err
			}
			msg := &types.MsgUpdateSpendingRules{
				Owner:          clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				Rules:          rules,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

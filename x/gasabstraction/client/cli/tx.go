package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// GetTxCmd returns the transaction commands for the gasabstraction module.
// Config updates are governance proposals (MsgUpdateConfig in a proposal
// file); only funding the float is a direct transaction.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the gasabstraction module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(CmdFundFloat())
	return cmd
}

// CmdFundFloat returns the command to move native tokens into the float.
func CmdFundFloat() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-float [amount]",
		Short: "Move native tokens (e.g. 1000000uqor) into the fee conversion float",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			amount, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgFundFloat{Sender: clientCtx.GetFromAddress().String(), Amount: amount}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

package cli

import (
	"encoding/hex"

	"github.com/spf13/cobra"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// GetTxCmd returns the transaction commands for the crossvm module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the crossvm module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(cmdCrossVMCall(), cmdProcessQueue())
	return cmd
}

func cmdCrossVMCall() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "call [source-vm] [target-vm] [target-contract]",
		Short: "Submit a cross-VM message (vm: evm|cosmwasm|svm). Payload via --payload (hex).",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			payloadHex, _ := cmd.Flags().GetString("payload")
			payload, err := hex.DecodeString(payloadHex)
			if err != nil {
				return err
			}
			fundsStr, _ := cmd.Flags().GetString("funds")
			var funds sdk.Coins
			if fundsStr != "" {
				funds, err = sdk.ParseCoinsNormalized(fundsStr)
				if err != nil {
					return err
				}
			}
			msg := &types.MsgCrossVMCall{
				Sender:         clientCtx.GetFromAddress().String(),
				SourceVM:       types.VMType(args[0]),
				TargetVM:       types.VMType(args[1]),
				TargetContract: args[2],
				Payload:        payload,
				Funds:          funds,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("payload", "", "hex-encoded call payload")
	cmd.Flags().String("funds", "", "coins to attach, e.g. 100uqor")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdProcessQueue() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process-queue",
		Short: "Manually process the pending cross-VM message queue",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgProcessQueue{Authority: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

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
		Use:   "call [target-vm] [target-contract]",
		Short: "Call a contract on another VM (vm: evm|cosmwasm|svm). Payload via --payload (hex).",
		Args:  cobra.ExactArgs(2),
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
			async, _ := cmd.Flags().GetBool("async")
			// No source-vm argument: the chain derives the origin lane from the
			// execution context, so asking the caller to name it would only invite a
			// value the node discards.
			msg := &types.MsgCrossVMCall{
				Sender:         clientCtx.GetFromAddress().String(),
				TargetVM:       types.VMType(args[0]),
				TargetContract: args[1],
				Payload:        payload,
				Funds:          funds,
				Async:          async,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("payload", "", "hex-encoded call payload")
	cmd.Flags().String("funds", "", "coins to attach, e.g. 100uqor (CosmWasm targets only)")
	cmd.Flags().Bool("async", false, "queue the call instead of executing it now; "+
		"the queue is dispatched by governance, so the result is not returned")
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

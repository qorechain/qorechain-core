package cli

import (
	"encoding/hex"
	"fmt"
	"math"

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
			asyncGas, _ := cmd.Flags().GetUint64("async-gas-limit")
			maxFactor, _ := cmd.Flags().GetFloat64("async-max-factor")
			if maxFactor < 0 || maxFactor > float64(types.MaxWasmCongestionMaxFactor) {
				return fmt.Errorf("--async-max-factor %v is outside 0..%d", maxFactor, types.MaxWasmCongestionMaxFactor)
			}
			// No source-vm argument: the chain derives the origin lane from the
			// execution context, so asking the caller to name it would only invite a
			// value the node discards. The field still gets a placeholder, because
			// ValidateBasic (here and on every node) refuses an empty one
			// (QSR-2026-0379).
			targetVM := types.VMType(args[0])
			msg := &types.MsgCrossVMCall{
				Sender:         clientCtx.GetFromAddress().String(),
				SourceVM:       types.PlaceholderSourceVM(targetVM),
				TargetVM:       targetVM,
				TargetContract: args[1],
				Payload:        payload,
				Funds:          funds,
				Async:          async,
				AsyncGasLimit:  asyncGas,
				AsyncMaxFactor: uint64(math.Round(maxFactor * float64(types.CongestionFactorOne))),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("payload", "", "hex-encoded call payload")
	cmd.Flags().String("funds", "", "coins to attach, e.g. 100uqor (CosmWasm targets only)")
	cmd.Flags().Bool("async", false, "queue the call instead of executing it now; "+
		"it runs at the end of the block (from v3.2.5 on a prepaid budget of 1,000,000 gas) and the result is not returned")
	cmd.Flags().Uint64("async-gas-limit", 0, "for an async call into CosmWasm, the contract execution it runs on in base "+
		"gas units (from v3.2.8; 0 means 500,000); it prepays this times the wasm congestion factor")
	cmd.Flags().Float64("async-max-factor", 0, "for an async call into CosmWasm, the highest wasm congestion factor it will pay, "+
		"e.g. 2.5 (from v3.2.8; 0 means twice the factor at submission); its execution is held in escrow at this factor, "+
		"it runs while the factor is at or below it, and what it does not use is refunded")
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

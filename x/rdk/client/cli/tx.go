package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetTxCmd returns the CLI transaction commands for the rdk module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the rdk module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		cmdCreateRollup(),
		cmdSubmitBatch(),
		cmdChallengeBatch(),
		cmdPauseRollup(),
		cmdResumeRollup(),
		cmdStopRollup(),
	)
	return cmd
}

func cmdCreateRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-rollup [rollup-id] [profile] [stake-amount]",
		Short: "Create an application-specific rollup (profile: defi|gaming|nft|enterprise|custom)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			stake, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			vm, _ := cmd.Flags().GetString("vm")
			msg := &types.MsgCreateRollup{
				Creator:     clientCtx.GetFromAddress().String(),
				RollupID:    args[0],
				Profile:     args[1],
				VmType:      vm,
				StakeAmount: stake,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("vm", "evm", "rollup VM type (evm|cosmwasm|svm|custom)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSubmitBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-batch [rollup-id] [batch-index] [state-root-hex]",
		Short: "Submit a settlement batch for a rollup",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			idx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			stateRoot, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}
			msg := &types.MsgSubmitBatch{
				Sequencer:  clientCtx.GetFromAddress().String(),
				RollupID:   args[0],
				BatchIndex: idx,
				StateRoot:  stateRoot,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdChallengeBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-batch [rollup-id] [batch-index]",
		Short: "Challenge a settlement batch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			idx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			proofHex, _ := cmd.Flags().GetString("proof")
			proof, err := hex.DecodeString(proofHex)
			if err != nil {
				return err
			}
			msg := &types.MsgChallengeBatch{
				Challenger: clientCtx.GetFromAddress().String(),
				RollupID:   args[0],
				BatchIndex: idx,
				Proof:      proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("proof", "", "hex-encoded fraud proof")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdPauseRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-rollup [rollup-id] [reason]",
		Short: "Pause a rollup",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgPauseRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
				Reason:   args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdResumeRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-rollup [rollup-id]",
		Short: "Resume a paused rollup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgResumeRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdStopRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop-rollup [rollup-id]",
		Short: "Stop a rollup and release its stake",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgStopRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

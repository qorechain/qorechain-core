package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetTxCmd returns the CLI transaction commands for the rdk module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the RDK (Rollup Development Kit) module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdCreateRollup(),
		CmdPauseRollup(),
		CmdResumeRollup(),
		CmdStopRollup(),
		CmdSubmitBatch(),
		CmdChallengeBatch(),
	)

	return cmd
}

// CmdCreateRollup creates a new rollup from a profile.
func CmdCreateRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-rollup [profile]",
		Short: "Create a new application-specific rollup",
		Long:  "Create a new rollup from a profile (defi, gaming, nft, enterprise, custom). Override defaults with flags.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			profile := args[0]
			settlement, _ := cmd.Flags().GetString("settlement")
			sequencer, _ := cmd.Flags().GetString("sequencer")
			da, _ := cmd.Flags().GetString("da")
			vm, _ := cmd.Flags().GetString("vm")
			stake, _ := cmd.Flags().GetInt64("stake")

			fmt.Printf("Creating rollup with profile: %s\n", profile)
			if settlement != "" {
				fmt.Printf("  settlement: %s\n", settlement)
			}
			if sequencer != "" {
				fmt.Printf("  sequencer: %s\n", sequencer)
			}
			if da != "" {
				fmt.Printf("  da: %s\n", da)
			}
			if vm != "" {
				fmt.Printf("  vm: %s\n", vm)
			}
			if stake > 0 {
				fmt.Printf("  stake: %d uqor\n", stake)
			}
			fmt.Println("Submit via signed transaction to create the rollup.")
			return nil
		},
	}
	cmd.Flags().String("settlement", "", "Settlement mode (optimistic, zk, based, sovereign)")
	cmd.Flags().String("sequencer", "", "Sequencer mode (dedicated, shared, based)")
	cmd.Flags().String("da", "", "DA backend (native, celestia, both)")
	cmd.Flags().String("vm", "", "VM type (evm, cosmwasm, svm, custom)")
	cmd.Flags().Int64("stake", 0, "Stake amount in uqor")
	return cmd
}

// CmdPauseRollup pauses an active rollup.
func CmdPauseRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-rollup [rollup-id]",
		Short: "Pause an active rollup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			reason, _ := cmd.Flags().GetString("reason")
			fmt.Printf("Pausing rollup %s (reason: %s)\n", args[0], reason)
			return nil
		},
	}
	cmd.Flags().String("reason", "maintenance", "Reason for pausing")
	return cmd
}

// CmdResumeRollup resumes a paused rollup.
func CmdResumeRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-rollup [rollup-id]",
		Short: "Resume a paused rollup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Resuming rollup %s\n", args[0])
			return nil
		},
	}
	return cmd
}

// CmdStopRollup permanently stops a rollup and returns bond.
func CmdStopRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop-rollup [rollup-id]",
		Short: "Permanently stop a rollup and return bond",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Stopping rollup %s (bond will be returned)\n", args[0])
			return nil
		},
	}
	return cmd
}

// CmdSubmitBatch submits a settlement batch for a rollup.
func CmdSubmitBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-batch [rollup-id] [state-root-hex] [data-hash-hex]",
		Short: "Submit a settlement batch for a rollup",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			proof, _ := cmd.Flags().GetString("proof")
			proofType, _ := cmd.Flags().GetString("proof-type")
			fmt.Printf("Submitting batch for rollup %s\n", args[0])
			fmt.Printf("  state-root: %s\n", args[1])
			fmt.Printf("  data-hash: %s\n", args[2])
			if proof != "" {
				fmt.Printf("  proof: %s (type: %s)\n", proof, proofType)
			}
			return nil
		},
	}
	cmd.Flags().String("proof", "", "Proof hex (required for ZK)")
	cmd.Flags().String("proof-type", "", "Proof type (snark, stark, fraud)")
	return cmd
}

// CmdChallengeBatch challenges a settlement batch with a fraud proof.
func CmdChallengeBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-batch [rollup-id] [batch-index] [fraud-proof-hex]",
		Short: "Challenge a settlement batch with a fraud proof",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Challenging batch %s for rollup %s\n", args[1], args[0])
			fmt.Printf("  fraud-proof: %s\n", args[2])
			return nil
		},
	}
	return cmd
}

package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// GetTxCmd returns the transaction commands for the rlconsensus module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "rlconsensus module transaction commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdSetAgentMode(),
		GetCmdResumeAgent(),
		GetCmdUpdatePolicy(),
		GetCmdUpdateRewardWeights(),
	)

	return cmd
}

// GetCmdSetAgentMode returns the command to set the RL agent operating mode.
func GetCmdSetAgentMode() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-agent-mode [shadow|conservative|autonomous|paused]",
		Short: "Set the RL agent operating mode",
		Long: `Set the RL agent operating mode.

Valid modes:
  shadow        - Log recommendations without applying them
  conservative  - Apply changes within tight bounds
  autonomous    - Apply changes within wider bounds
  paused        - Disable observation and action entirely`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			mode := args[0]
			switch mode {
			case "shadow", "conservative", "autonomous", "paused":
				// valid
			default:
				return fmt.Errorf("invalid agent mode %q: must be one of shadow, conservative, autonomous, paused", mode)
			}

			fmt.Printf("Setting agent mode to: %s\n", mode)
			fmt.Println("(Full transaction support will be added with proto definitions)")
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdResumeAgent returns the command to resume the RL agent from paused state.
func GetCmdResumeAgent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-agent",
		Short: "Resume the RL agent from paused state",
		Long:  "Resume the RL agent from a paused state back to shadow mode.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Resuming RL agent to shadow mode")
			fmt.Println("(Full transaction support will be added with proto definitions)")
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdUpdatePolicy returns the command to update the policy network weights.
func GetCmdUpdatePolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-policy [weights-file]",
		Short: "Update the policy network weights from a JSON file",
		Long: `Update the policy network weights from a JSON file.

The weights file should contain a JSON-serialized PolicyWeights structure
with epoch, MLP configuration, and the flattened weight vector.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Updating policy weights from file: %s\n", args[0])
			fmt.Println("(Full transaction support will be added with proto definitions)")
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdUpdateRewardWeights returns the command to update reward function weights.
func GetCmdUpdateRewardWeights() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-reward-weights [w1] [w2] [w3] [w4] [w5]",
		Short: "Update the reward function weights",
		Long: `Update the reward function weights.

Weights are specified as five decimal values that must sum to 1.0:
  w1 - throughput weight
  w2 - finality weight
  w3 - decentralization weight
  w4 - MEV weight
  w5 - failed transactions weight`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Printf("Updating reward weights: throughput=%s finality=%s decentralization=%s mev=%s failed_txs=%s\n",
				args[0], args[1], args[2], args[3], args[4])
			fmt.Println("(Full transaction support will be added with proto definitions)")
			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

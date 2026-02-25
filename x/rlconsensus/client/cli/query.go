package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/rlconsensus/types"
)

// GetQueryCmd returns the CLI query commands for the rlconsensus module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the rlconsensus module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryAgentStatus(),
		GetCmdQueryObservation(),
		GetCmdQueryReward(),
		GetCmdQueryParams(),
		GetCmdQueryPolicy(),
	)

	return cmd
}

// GetCmdQueryAgentStatus returns the command to query the RL agent status.
func GetCmdQueryAgentStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-status",
		Short: "Query the current RL agent status",
		Long:  "Query the current RL agent operating mode, epoch, total steps, and circuit breaker state.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Query agent status")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryObservation returns the command to query the latest observation.
func GetCmdQueryObservation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observation",
		Short: "Query the latest observation vector",
		Long:  "Query the most recent 25-dimensional observation vector captured by the RL agent.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Query latest observation")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryReward returns the command to query the latest reward.
func GetCmdQueryReward() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reward",
		Short: "Query the latest reward signal",
		Long:  "Query the most recent reward signal computed by the RL agent including per-component deltas.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Query latest reward")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryParams returns the command to query the rlconsensus module parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query rlconsensus module parameters",
		Long:  "Query the current parameters of the rlconsensus module including agent mode, observation interval, and reward weights.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Query module params")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryPolicy returns the command to query the current policy weights.
func GetCmdQueryPolicy() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Query the current policy network weights",
		Long:  "Query the current MLP policy network weights, epoch, and architecture configuration.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Query current policy")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

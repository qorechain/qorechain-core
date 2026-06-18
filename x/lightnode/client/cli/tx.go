package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/lightnode/types"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the lightnode module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdRegister(),
		CmdHeartbeat(),
		CmdDeregister(),
		CmdClaimRewards(),
	)

	return cmd
}

func CmdRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register [node-type] [version]",
		Short: "Register a new light node (node-type: sx or ux)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			capsCSV, _ := cmd.Flags().GetString("capabilities")
			var caps []string
			if capsCSV != "" {
				caps = strings.Split(capsCSV, ",")
			}
			msg := &types.MsgRegisterLightNode{
				Operator:     clientCtx.GetFromAddress().String(),
				NodeType:     args[0],
				Version:      args[1],
				Capabilities: caps,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("capabilities", "", "comma-separated capability list")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdHeartbeat() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "heartbeat",
		Short: "Submit a liveness heartbeat for a registered light node",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgHeartbeat{Operator: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdDeregister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deregister",
		Short: "Deregister a light node from the network",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgDeregisterLightNode{Operator: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdClaimRewards() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim-rewards",
		Short: "Claim accumulated light node rewards",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgClaimLightNodeRewards{Operator: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

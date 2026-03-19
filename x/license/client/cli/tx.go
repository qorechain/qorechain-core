package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// GetTxCmd returns the transaction commands for the license module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the license module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdGrant(),
		CmdRevoke(),
		CmdSuspend(),
		CmdResume(),
	)

	return cmd
}

// CmdGrant returns the command to grant a license to a validator.
func CmdGrant() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant [validator-addr] [feature-id]",
		Short: "Grant a license to a validator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			expiresAt, _ := cmd.Flags().GetString("expires-at")
			metadata, _ := cmd.Flags().GetString("metadata")
			fmt.Fprintf(clientCtx.Output, "Granting license %s to %s (expires: %s, metadata: %s) from %s\n",
				args[1], args[0], expiresAt, metadata, clientCtx.GetFromAddress())
			return nil
		},
	}
	cmd.Flags().String("expires-at", "0", "Block height when license expires (0 for perpetual)")
	cmd.Flags().String("metadata", "", "Optional metadata JSON")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevoke returns the command to revoke a license from a validator.
func CmdRevoke() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke [validator-addr] [feature-id]",
		Short: "Revoke a license from a validator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Revoking license %s from %s (from %s)\n",
				args[1], args[0], clientCtx.GetFromAddress())
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSuspend returns the command to suspend a license.
func CmdSuspend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suspend [validator-addr] [feature-id]",
		Short: "Suspend a license",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Suspending license %s for %s (from %s)\n",
				args[1], args[0], clientCtx.GetFromAddress())
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdResume returns the command to resume a suspended license.
func CmdResume() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [validator-addr] [feature-id]",
		Short: "Resume a suspended license",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Resuming license %s for %s (from %s)\n",
				args[1], args[0], clientCtx.GetFromAddress())
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

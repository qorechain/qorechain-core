package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

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

// CmdGrant returns the command to grant a license to a grantee.
func CmdGrant() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "grant [grantee-addr] [feature-id]",
		Short: "Grant a license to a grantee (authority = --from)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			expiresStr, _ := cmd.Flags().GetString("expires-at")
			expiresAt, err := strconv.ParseInt(expiresStr, 10, 64)
			if err != nil {
				return err
			}
			metadata, _ := cmd.Flags().GetString("metadata")
			msg := &types.MsgGrantLicense{
				Authority: clientCtx.GetFromAddress().String(),
				Grantee:   args[0],
				FeatureID: args[1],
				ExpiresAt: expiresAt,
				Metadata:  metadata,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("expires-at", "0", "Block height when license expires (0 for perpetual)")
	cmd.Flags().String("metadata", "", "Optional metadata JSON")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevoke returns the command to revoke a license from a grantee.
func CmdRevoke() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke [grantee-addr] [feature-id]",
		Short: "Revoke a license from a grantee (authority = --from)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgRevokeLicense{
				Authority: clientCtx.GetFromAddress().String(),
				Grantee:   args[0],
				FeatureID: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdSuspend returns the command to suspend a license.
func CmdSuspend() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suspend [grantee-addr] [feature-id]",
		Short: "Suspend a license (authority = --from)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgSuspendLicense{
				Authority: clientCtx.GetFromAddress().String(),
				Grantee:   args[0],
				FeatureID: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdResume returns the command to resume a suspended license.
func CmdResume() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume [grantee-addr] [feature-id]",
		Short: "Resume a suspended license (authority = --from)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgResumeLicense{
				Authority: clientCtx.GetFromAddress().String(),
				Grantee:   args[0],
				FeatureID: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

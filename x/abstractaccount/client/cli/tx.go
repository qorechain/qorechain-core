package cli

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// GetTxCmd returns the CLI transaction commands for the abstractaccount module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the abstractaccount module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdCreateAbstractAccount(),
		CmdUpdateSpendingRules(),
		CmdRegisterAuthenticator(),
		CmdRevokeAuthenticator(),
	)

	return cmd
}

// CmdRegisterAuthenticator links a foreign-scheme wallet key to an account.
func CmdRegisterAuthenticator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-authenticator [account-address] [scheme] [pubkey-base64] [permissions-csv] [expiry-unix] [label]",
		Short: "Link a foreign-scheme wallet key (e.g. Phantom ed25519) to an account (owner-signed)",
		Args:  cobra.RangeArgs(5, 6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pk, err := base64.StdEncoding.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("pubkey must be base64: %w", err)
			}
			exp, err := strconv.ParseInt(args[4], 10, 64)
			if err != nil {
				return fmt.Errorf("expiry-unix must be an integer: %w", err)
			}
			label := ""
			if len(args) == 6 {
				label = args[5]
			}
			msg := &types.MsgRegisterAuthenticator{
				Owner:          clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				Scheme:         args[1],
				Pubkey:         pk,
				Permissions:    strings.Split(args[3], ","),
				ExpiryUnix:     exp,
				Label:          label,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdRevokeAuthenticator revokes a previously linked wallet key.
func CmdRevokeAuthenticator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-authenticator [account-address] [scheme] [pubkey-base64]",
		Short: "Revoke a previously linked wallet key (owner-signed)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pk, err := base64.StdEncoding.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("pubkey must be base64: %w", err)
			}
			msg := &types.MsgRevokeAuthenticator{
				Owner:          clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				Scheme:         args[1],
				Pubkey:         pk,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdCreateAbstractAccount returns the command to create an abstract account.
func CmdCreateAbstractAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [account-type]",
		Short: "Create a new abstract account (type: multisig|social_recovery|session_based)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgCreateAbstractAccount{
				Owner:       clientCtx.GetFromAddress().String(),
				AccountType: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUpdateSpendingRules returns the command to update spending rules.
func CmdUpdateSpendingRules() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-rules [account-address] [rules-json]",
		Short: "Update spending rules (rules-json: array of SpendingRule objects)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			var rules []types.SpendingRule
			if err := json.Unmarshal([]byte(args[1]), &rules); err != nil {
				return err
			}
			msg := &types.MsgUpdateSpendingRules{
				Owner:          clientCtx.GetFromAddress().String(),
				AccountAddress: args[0],
				Rules:          rules,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

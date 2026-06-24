package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// GetQueryCmd returns the CLI query commands for the abstractaccount module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the abstractaccount module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryConfig(),
		CmdQueryAccount(),
		CmdQueryAccounts(),
	)
	return cmd
}

// CmdQueryConfig queries the abstract account config.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the abstract account configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Config(cmd.Context(), &types.QueryConfigRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryAccount queries an abstract account by address.
func CmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [address]",
		Short: "Query an abstract account by address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Account(cmd.Context(), &types.QueryAccountRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryAccounts lists all abstract accounts.
func CmdQueryAccounts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List all abstract accounts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Accounts(cmd.Context(), &types.QueryAccountsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

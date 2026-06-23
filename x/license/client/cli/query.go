package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// GetQueryCmd returns the CLI query commands for the license module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Query commands for the license module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		CmdQueryList(),
		CmdQueryCheck(),
		CmdQueryHolders(),
	)
	return cmd
}

// CmdQueryList lists all licenses held by a grantee.
func CmdQueryList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [grantee-addr]",
		Short: "List all licenses held by a grantee",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).List(cmd.Context(), &types.QueryListRequest{Grantee: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryCheck checks a single (grantee, feature) license grant.
func CmdQueryCheck() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [grantee-addr] [feature-id]",
		Short: "Check a grantee's license for a feature",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Check(cmd.Context(),
				&types.QueryCheckRequest{Grantee: args[0], FeatureId: args[1]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryHolders lists all grantees holding a given feature license.
func CmdQueryHolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "holders [feature-id]",
		Short: "List all grantees holding a given feature license",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Holders(cmd.Context(),
				&types.QueryHoldersRequest{FeatureId: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

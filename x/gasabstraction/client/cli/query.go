package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/gasabstraction/types"
)

// GetQueryCmd returns the CLI query commands for the gasabstraction module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the gasabstraction module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryConfig(),
		CmdQueryAcceptedTokens(),
		CmdQueryConversionRate(),
		CmdQueryFloat(),
	)

	return cmd
}

// CmdQueryConfig returns the command to query the gas abstraction config.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the gas abstraction configuration",
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

// CmdQueryAcceptedTokens returns the command to list accepted fee tokens.
func CmdQueryAcceptedTokens() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "accepted-tokens",
		Short: "List all accepted fee tokens and how they are priced",
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
			return clientCtx.PrintProto(&res.Config)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryConversionRate returns the command to quote a fee token.
func CmdQueryConversionRate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "conversion-rate [denom]",
		Short: "What one unit of a fee token is worth in the native denom now, and whether it can pay fees",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).ConversionRate(cmd.Context(), &types.QueryConversionRateRequest{Denom: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryFloat returns the command to query the conversion float.
func CmdQueryFloat() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "float",
		Short: "Query the fee conversion float's balances and this block's usage",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Float(cmd.Context(), &types.QueryFloatRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

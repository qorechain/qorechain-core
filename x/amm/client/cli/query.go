package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// GetQueryCmd returns the AMM module's query subcommand tree.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ammtypes.ModuleName,
		Short:                      "Querying commands for the AMM module",
		SuggestionsMinimumDistance: 2,
		RunE:                       func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	cmd.AddCommand(
		cmdParams(),
		cmdPool(),
		cmdPools(),
		cmdPoolByDenoms(),
		cmdLPBalance(),
		cmdQuoteExactIn(),
		cmdQuoteExactOut(),
	)
	return cmd
}

func cmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Show the current AMM module params",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).Params(cmd.Context(), &ammtypes.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool [pool-id]",
		Short: "Show a single pool by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).Pool(cmd.Context(), &ammtypes.QueryPoolRequest{PoolId: id})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdPools() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pools",
		Short: "List all pools",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).Pools(cmd.Context(), &ammtypes.QueryPoolsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdPoolByDenoms() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool-by-denoms [denom-a] [denom-b]",
		Short: "Find a pool by its (denomA, denomB) tuple",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).PoolByDenoms(cmd.Context(),
				&ammtypes.QueryPoolByDenomsRequest{DenomA: args[0], DenomB: args[1]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdLPBalance() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lp-balance [pool-id] [address]",
		Short: "Query an account's LP balance for a pool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).LPBalance(cmd.Context(),
				&ammtypes.QueryLPBalanceRequest{PoolId: id, Address: args[1]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQuoteExactIn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote-exact-in [pool-id] [denom-in] [amount-in]",
		Short: "Quote the output for a fixed-input swap",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).QuoteExactIn(cmd.Context(),
				&ammtypes.QueryQuoteExactInRequest{PoolId: id, DenomIn: args[1], AmountIn: args[2]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdQuoteExactOut() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quote-exact-out [pool-id] [denom-out] [amount-out]",
		Short: "Quote the input required for a fixed-output swap",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			res, err := ammtypes.NewQueryClient(clientCtx).QuoteExactOut(cmd.Context(),
				&ammtypes.QueryQuoteExactOutRequest{PoolId: id, DenomOut: args[1], AmountOut: args[2]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

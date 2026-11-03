package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// GetQueryCmd returns the stakingtier query commands.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Query delegation locks and the bonus pool",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(cmdParams(), cmdLocks(), cmdPool(), cmdApr(), cmdDelegatorApr())
	return cmd
}

func cmdParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Show the staking tier parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdLocks() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locks [holder-addr]",
		Short: "Show a holder's locks, their projected bonus and what can be claimed",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Locks(cmd.Context(), &types.QueryLocksRequest{Holder: args[0]})
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
		Use:   "pool",
		Short: "Show the bonus pool: balance, outstanding, surplus and the last settlement",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).Pool(cmd.Context(), &types.QueryPoolRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

const flagValidator = "validator"

func cmdApr() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apr",
		Short: "Show the base APR the chain last measured and what each lock term adds",
		Long: "Show the base APR from the latest bonus settlement (gross, before commission), how that\n" +
			"settlement was funded, and each lock term's bonus after the max_total_apy_bps ceiling.\n" +
			"With --validator the base is that validator's delegators' base, net of its commission.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			val, _ := cmd.Flags().GetString(flagValidator)
			res, err := types.NewQueryClient(clientCtx).Apr(cmd.Context(), &types.QueryAprRequest{Validator: val})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String(flagValidator, "", "state the base net of this validator's commission (qorvaloper)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

func cmdDelegatorApr() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delegator-apr [delegator-addr]",
		Short: "Show a delegator's effective APR per delegation, lock and unlocked rest, with the reason",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			val, _ := cmd.Flags().GetString(flagValidator)
			res, err := types.NewQueryClient(clientCtx).DelegatorApr(cmd.Context(),
				&types.QueryDelegatorAprRequest{Delegator: args[0], Validator: val})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().String(flagValidator, "", "limit the answer to the delegation on this validator (qorvaloper)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

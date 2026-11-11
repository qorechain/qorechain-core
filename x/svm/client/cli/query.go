package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// svmQueryClient builds a gRPC query client from the CLI client context.
func svmQueryClient(cmd *cobra.Command) (client.Context, types.QueryClient, error) {
	clientCtx, err := client.GetClientQueryContext(cmd)
	if err != nil {
		return client.Context{}, nil, err
	}
	return clientCtx, types.NewQueryClient(clientCtx), nil
}

// GetQueryCmd returns the CLI query commands for the SVM module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the SVM module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryAccount(),
		GetCmdQueryProgram(),
		GetCmdQueryParams(),
		GetCmdQuerySlot(),
		GetCmdQueryClusterInfo(),
	)

	return cmd
}

// GetCmdQueryAccount returns the command to query an SVM account by base58 address.
func GetCmdQueryAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account [base58-address]",
		Short: "Query an SVM account by its base58 address",
		Long:  "Query an SVM account's lamports, data size, owner, and executable status by its base58-encoded address.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, qc, err := svmQueryClient(cmd)
			if err != nil {
				return err
			}
			if _, err := types.Base58DecodeCanonical(args[0]); err != nil {
				return fmt.Errorf("invalid base58 address: %w", err)
			}
			res, err := qc.Account(cmd.Context(), &types.QueryAccountRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryProgram returns the command to query an SVM program by base58 address.
func GetCmdQueryProgram() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "program [base58-address]",
		Short: "Query an SVM program by its base58 address",
		Long:  "Query a deployed SVM program's metadata, including deployer, bytecode hash, and deployment height.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, qc, err := svmQueryClient(cmd)
			if err != nil {
				return err
			}
			if _, err := types.Base58DecodeCanonical(args[0]); err != nil {
				return fmt.Errorf("invalid base58 address: %w", err)
			}
			res, err := qc.Program(cmd.Context(), &types.QueryProgramRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryParams returns the command to query SVM module parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query SVM module parameters",
		Long: "Query the SVM runtime parameters in force, every field (compute budget, rent settings, " +
			"program size limits, the custody ceilings), in the shape MsgUpdateParams takes.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, qc, err := svmQueryClient(cmd)
			if err != nil {
				return err
			}
			res, err := qc.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQuerySlot returns the command to query the current SVM slot.
func GetCmdQuerySlot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "slot",
		Short: "Query the current SVM slot number",
		Long:  "Query the current SVM virtual slot. Full gRPC query support will be added with proto definitions.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, qc, err := svmQueryClient(cmd)
			if err != nil {
				return err
			}
			res, err := qc.Slot(cmd.Context(), &types.QuerySlotRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryClusterInfo returns the command to query the SVM cluster identity
// (genesis hash and Wallet Standard chain string).
func GetCmdQueryClusterInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-info",
		Short: "Query the SVM genesis hash and Wallet Standard chain id",
		Long: "Query the SVM cluster identity: the base58 genesis hash the Solana-compatible " +
			"getGenesisHash method returns and the \"solana:<first 32 chars>\" chain string " +
			"Solana Wallet Standard wallets use. Both are derived from the chain id and never change.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, qc, err := svmQueryClient(cmd)
			if err != nil {
				return err
			}
			res, err := qc.ClusterInfo(cmd.Context(), &types.QueryClusterInfoRequest{})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

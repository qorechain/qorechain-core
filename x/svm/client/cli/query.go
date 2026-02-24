package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// GetQueryCmd returns the CLI query commands for the SVM module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the SVM module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryAccount(),
		GetCmdQueryProgram(),
		GetCmdQueryParams(),
		GetCmdQuerySlot(),
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
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			addr, err := types.Base58Decode(args[0])
			if err != nil {
				return fmt.Errorf("invalid base58 address: %w", err)
			}
			_ = addr

			fmt.Printf("Querying SVM account: %s\n", args[0])
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

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
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			addr, err := types.Base58Decode(args[0])
			if err != nil {
				return fmt.Errorf("invalid base58 address: %w", err)
			}
			_ = addr

			fmt.Printf("Querying SVM program: %s\n", args[0])
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

// GetCmdQueryParams returns the command to query SVM module parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query SVM module parameters",
		Long:  "Query the current parameters of the SVM module, including compute budget, rent settings, and program size limits.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			params := types.DefaultParams()
			bz, err := json.MarshalIndent(params, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(bz))
			return nil
		},
	}

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
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx

			fmt.Println("Current SVM slot:")
			fmt.Println("(Full gRPC query support will be added with proto definitions)")
			return nil
		},
	}

	return cmd
}

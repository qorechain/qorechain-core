package cli

import (
	"fmt"

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

// CmdQueryList returns the command to list all licenses for a validator.
func CmdQueryList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [validator-addr]",
		Short: "List all licenses for a validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Querying licenses for %s...\n", args[0])
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryCheck returns the command to check if a validator has an active license.
func CmdQueryCheck() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [validator-addr] [feature-id]",
		Short: "Check if a validator has an active license",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Checking license %s for %s...\n", args[1], args[0])
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdQueryHolders returns the command to list all validators with a given license.
func CmdQueryHolders() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "holders [feature-id]",
		Short: "List all validators with a given license",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(clientCtx.Output, "Querying holders of %s...\n", args[0])
			return nil
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

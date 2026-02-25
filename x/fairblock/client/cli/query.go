package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"

	"github.com/qorechain/qorechain-core/x/fairblock/types"
)

// GetQueryCmd returns the CLI query commands for the fairblock module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the fairblock module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		CmdQueryConfig(),
		CmdQueryStatus(),
	)

	return cmd
}

// CmdQueryConfig returns the command to query the fairblock config.
func CmdQueryConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Query the FairBlock threshold IBE configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			return cmd.Help()
		},
	}
	return cmd
}

// CmdQueryStatus returns the command to query the fairblock status.
func CmdQueryStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Query whether FairBlock tIBE decryption is active",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			_ = clientCtx
			return cmd.Help()
		},
	}
	return cmd
}

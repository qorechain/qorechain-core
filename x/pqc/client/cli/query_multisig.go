package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetCmdQueryMultisig returns the command that shows a multisig account's
// registered post-quantum member set (v3.2.7).
func GetCmdQueryMultisig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig [address]",
		Short: "Show a multisig account's post-quantum member set (threshold and member keys)",
		Long: `Show the post-quantum member set a multisig account registered: the
threshold k, the n member ML-DSA-87 keys in member order (the order of the
bitmap in every PQCMultisigSignature), and how many times the set was replaced.
found is false for an account without one.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewMultisigQueryClient(clientCtx).Multisig(cmd.Context(),
				&types.QueryMultisigRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

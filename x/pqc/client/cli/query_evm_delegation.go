package cli

import (
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetCmdQueryEVMDelegationConsent returns the command that shows an address's
// EIP-7702 delegation consent and the state a client needs to sign the tuple.
func GetCmdQueryEVMDelegationConsent() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "evm-delegation-consent [address]",
		Short: "Show an address's EIP-7702 delegation consent, current delegation and EVM nonce",
		Long: `Show whether an address has a live post-quantum consent to an EIP-7702
delegation, which delegate it names, the delegation currently installed on the
account and the account's EVM nonce.

A sponsored authorization tuple carries that nonce; a tuple in the account's own
type-4 transaction carries it plus one. Accepts a bech32 or 0x address.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).EVMDelegationConsent(cmd.Context(),
				&types.QueryEVMDelegationConsentRequest{Address: args[0]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

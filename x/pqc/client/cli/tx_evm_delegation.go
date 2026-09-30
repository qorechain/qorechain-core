package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetCmdAuthorizeEVMDelegation returns the command that records the account's
// post-quantum consent to one EIP-7702 delegation.
func GetCmdAuthorizeEVMDelegation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "authorize-evm-delegation [delegate] [blocks]",
		Short: "Consent, with your post-quantum key, to one EIP-7702 delegation of your account",
		Long: `Record your post-quantum consent to one EIP-7702 delegation of your own account.

An EIP-7702 authorization is signed with the classical key alone, and it changes
what your account is: once the delegation is installed, its code acts for your
account whenever anyone calls it. The EVM lane therefore admits a type-4
transaction carrying your authorization only if you consented to exactly that
delegate here, with the post-quantum key. The consent is used up by the
transaction that installs the delegation, whoever sends it, so sponsored and
relayed upgrades keep working.

  delegate  the 0x address your authorization tuple names. The zero address
            (0x0000000000000000000000000000000000000000) consents to clearing
            your delegation.
  blocks    how long the consent waits to be used.

A new consent replaces the previous one; close-evm-window revokes it. Sign the
tuple AFTER this transaction is included, with the nonce 'query pqc
evm-delegation-consent' reports (plus one if your own type-4 transaction carries
it), since this transaction moves your nonce.`,
		Example: `  qorechaind tx pqc authorize-evm-delegation 0x63c0c19a282a1B52b07dD5a65b58948A07DAE32B 600 --from me --generate-only > c.json
  qorechaind tx pqc cosign c.json --from me --pqc-key me-mldsa --chain-id qorechain-vladi`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			delegate, err := types.ParseDelegate(args[0])
			if err != nil {
				return err
			}
			blocks, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("blocks: %w", err)
			}
			msg := &types.MsgAuthorizeEVMDelegation{
				Sender:   clientCtx.GetFromAddress().String(),
				Delegate: delegate.Hex(),
				Blocks:   blocks,
			}
			if err := msg.Validate(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// GetCmdClearEVMDelegation returns the command that removes the account's
// EIP-7702 delegation on the Cosmos lane.
func GetCmdClearEVMDelegation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clear-evm-delegation",
		Short: "Remove your account's EIP-7702 delegation, with your post-quantum key",
		Long: `Remove the EIP-7702 delegation installed on your account, directly on the
Cosmos lane. No EVM transaction, window or classical key is involved, so this is
the recovery path when the classical key is suspected compromised. It also
revokes any pending delegation consent.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgClearEVMDelegation{Sender: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

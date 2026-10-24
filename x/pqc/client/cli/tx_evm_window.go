package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetCmdOpenEVMWindow returns the command that authorises the EVM lane for a
// bounded period.
//
// This is a Cosmos-lane transaction, so on a network that requires the
// post-quantum signature it is produced with --generate-only and then cosigned:
//
//	qorechaind tx pqc open-evm-window 7200 20 1000qor --from me --generate-only > w.json
//	qorechaind tx pqc cosign w.json --from me --pqc-key me-mldsa --chain-id <id>
//
// That two-step shape is the whole point: the window cannot be opened without the
// ML-DSA key, which is what makes the classical key alone insufficient on the EVM
// lane.
func GetCmdOpenEVMWindow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open-evm-window [blocks] [max-txs] [max-value]",
		Short: "Authorise your EVM lane for a bounded number of blocks, transactions and value",
		Long: `Open a post-quantum authorisation window for the EVM lane.

Inside the window an ordinary MetaMask transaction works unmodified. Outside it,
or once any bound is used up, the EVM lane refuses your transactions. The classical
key alone can never open one: this message requires the post-quantum signature
like every other Cosmos-lane message.

  blocks     how long the window stays open, counted from the block that opens it
  max-txs    how many EVM transactions it admits
  max-value  the total it admits, counting transferred value AND the maximum fee
             each transaction could pay. Accepts uqor or qor, e.g. 1000qor.

The arguments are optional; omitted ones take the defaults below, sized for one
contract deployment session from Hardhat or Foundry (a deploy, its
initialisation calls, a retry or two):

  blocks     3600 (about an hour on the testnet, three on mainnet)
  max-txs    10
  max-value  25qor. The fee bound of a transaction is its gas limit
             times its fee cap. With the current base fee (0.1 uqor per gas)
             and the fee cap wallets set (twice the base fee plus the tip), a
             3,000,000 gas deploy counts about 0.64 QOR; 25 QOR leaves room for
             several large deploys or a base fee several times higher. Add the
             value of any QOR the deployment transfers.

The value bound counts native QOR only. A token transfer, an approval or any
other contract call carries value zero and is not counted against it. To bound
those too, narrow the window's scope (from the evm-pqc-window-targets feature on):

  --native-only        admit plain QOR transfers only: no calldata, no contract
                       creation, no recipient that carries code
  --allowed-targets    admit only these recipients (comma-separated 0x addresses);
                       contract creation is refused

Opening replaces any existing window. Close one early with close-evm-window.`,
		Example: `  qorechaind tx pqc open-evm-window 7200 20 1000qor --from me --generate-only > w.json
  qorechaind tx pqc cosign w.json --from me --pqc-key me-mldsa --chain-id qorechain-vladi

  # a deploy session with the default bounds (3600 blocks, 10 txs, 25qor)
  qorechaind tx pqc open-evm-window --from me --generate-only > w.json`,
		Args: cobra.RangeArgs(0, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			args = append(args, defaultEVMWindowArgs[len(args):]...)
			blocks, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("blocks: %w", err)
			}
			maxTxs, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("max-txs: %w", err)
			}
			maxValue, err := parseValueUqor(args[2])
			if err != nil {
				return fmt.Errorf("max-value: %w", err)
			}

			nativeOnly, err := cmd.Flags().GetBool(flagNativeOnly)
			if err != nil {
				return err
			}
			targets, err := cmd.Flags().GetStringSlice(flagAllowedTargets)
			if err != nil {
				return err
			}

			msg := &types.MsgOpenEVMWindow{
				Sender:         clientCtx.GetFromAddress().String(),
				Blocks:         blocks,
				MaxTxs:         maxTxs,
				MaxValue:       maxValue,
				NativeOnly:     nativeOnly,
				AllowedTargets: targets,
			}
			// Fail here rather than on chain: the bounds are a security parameter
			// and a refusal should carry the reason, not a gas bill.
			if err := msg.Validate(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Bool(flagNativeOnly, false, "admit plain native QOR transfers only")
	cmd.Flags().StringSlice(flagAllowedTargets, nil, "admit only these recipients (comma-separated 0x addresses)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// defaultEVMWindowArgs are the bounds open-evm-window uses for arguments left
// out: blocks, max-txs, max-value. See the command's help for the sizing.
var defaultEVMWindowArgs = []string{"3600", "10", "25qor"}

const (
	flagNativeOnly     = "native-only"
	flagAllowedTargets = "allowed-targets"
)

// GetCmdCloseEVMWindow returns the command that revokes the window immediately.
func GetCmdCloseEVMWindow() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "close-evm-window",
		Short: "Revoke your EVM-lane authorisation window immediately",
		Long: `Close your EVM authorisation window without waiting for it to expire.

Use this the moment you suspect the classical key is compromised: whatever the
window still admitted is refused from the next block on.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgCloseEVMWindow{Sender: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseValueUqor accepts "1000qor", "1000000000uqor" or a bare integer of uqor.
//
// The display unit is accepted because a person thinking about how much to
// authorise thinks in QOR, and getting the six zeros wrong on a security bound is
// exactly the kind of mistake this command should not make easy.
func parseValueUqor(s string) (sdkmath.Int, error) {
	if coin, err := sdk.ParseCoinNormalized(s); err == nil {
		switch coin.Denom {
		case "uqor":
			return coin.Amount, nil
		case "qor":
			return coin.Amount.Mul(sdkmath.NewInt(1_000_000)), nil
		default:
			return sdkmath.Int{}, fmt.Errorf("unknown denom %q; use qor or uqor", coin.Denom)
		}
	}
	n, ok := sdkmath.NewIntFromString(s)
	if !ok {
		return sdkmath.Int{}, fmt.Errorf("%q is not an amount; use e.g. 1000qor or 1000000000uqor", s)
	}
	return n, nil
}

package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/stakingtier/types"
)

// GetTxCmd returns the stakingtier transaction commands.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Delegation locks and their bonus yield",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(CmdLock(), CmdUnlockEarly(), CmdClaim())
	return cmd
}

// CmdLock locks part of an existing delegation.
func CmdLock() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock [validator-addr] [amount-uqor] [months]",
		Short: "Lock part of your delegation to a validator for 3, 6 or 12 months",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			amount, ok := math.NewIntFromString(args[1])
			if !ok {
				return fmt.Errorf("amount %q is not an integer number of uqor", args[1])
			}
			months, err := strconv.ParseUint(args[2], 10, 32)
			if err != nil {
				return fmt.Errorf("months: %w", err)
			}
			msg := &types.MsgLockDelegation{
				Delegator: clientCtx.GetFromAddress().String(),
				Validator: args[0],
				Amount:    amount,
				Months:    uint32(months),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdUnlockEarly ends a lock before it expires.
func CmdUnlockEarly() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock-early [lock-id]",
		Short: "End a lock early: forfeits its unpaid bonus and burns the early-unlock penalty",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			id, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("lock id: %w", err)
			}
			msg := &types.MsgUnlockEarly{Delegator: clientCtx.GetFromAddress().String(), LockId: id}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// CmdClaim claims every bonus credited to the signer.
func CmdClaim() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claim",
		Short: "Claim the staking tier bonus credited to you",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgClaimBonus{Holder: clientCtx.GetFromAddress().String()}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

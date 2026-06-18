package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ammtypes "github.com/qorechain/qorechain-core/x/amm/types"
)

// GetTxCmd returns the AMM module's transaction subcommand tree.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        ammtypes.ModuleName,
		Short:                      "AMM transaction subcommands",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		cmdCreatePool(),
		cmdAddLiquidity(),
		cmdRemoveLiquidity(),
		cmdSwapExactIn(),
		cmdSwapExactOut(),
		cmdPausePool(),
		cmdResumePool(),
	)
	return cmd
}

func cmdCreatePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-pool [pool-type] [deposit-a] [deposit-b]",
		Short: "Create a pool (pool-type: constant_product|stable_swap; deposits as coins e.g. 1000uqor)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			depA, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			depB, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}
			amp, _ := cmd.Flags().GetUint32("amp")
			msg := &ammtypes.MsgCreatePool{
				Creator:                  clientCtx.GetFromAddress().String(),
				PoolType:                 ammtypes.PoolType(args[0]),
				InitialDepositA:          depA,
				InitialDepositB:          depB,
				AmplificationCoefficient: amp,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint32("amp", 0, "amplification coefficient (stable_swap only)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdAddLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-liquidity [pool-id] [amount-a] [amount-b] [min-lp-out]",
		Args:  cobra.ExactArgs(4),
		Short: "Add liquidity to a pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			amtA, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			amtB, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}
			minLP, ok := math.NewIntFromString(args[3])
			if !ok {
				return ammtypes.ErrInvalidLPAmount
			}
			msg := &ammtypes.MsgAddLiquidity{
				Sender: clientCtx.GetFromAddress().String(), PoolID: pid,
				AmountA: amtA, AmountB: amtB, MinLPOut: minLP,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdRemoveLiquidity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-liquidity [pool-id] [lp-amount] [min-a] [min-b]",
		Args:  cobra.ExactArgs(4),
		Short: "Burn LP tokens and withdraw reserves",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			lp, ok := math.NewIntFromString(args[1])
			if !ok {
				return ammtypes.ErrInvalidLPAmount
			}
			minA, ok := math.NewIntFromString(args[2])
			if !ok {
				return ammtypes.ErrInvalidAmount
			}
			minB, ok := math.NewIntFromString(args[3])
			if !ok {
				return ammtypes.ErrInvalidAmount
			}
			msg := &ammtypes.MsgRemoveLiquidity{
				Sender: clientCtx.GetFromAddress().String(), PoolID: pid,
				LPAmount: lp, MinAmountA: minA, MinAmountB: minB,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSwapExactIn() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap-exact-in [pool-id] [token-in] [denom-out] [min-out]",
		Args:  cobra.ExactArgs(4),
		Short: "Swap a fixed input amount",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			tokenIn, err := sdk.ParseCoinNormalized(args[1])
			if err != nil {
				return err
			}
			minOut, ok := math.NewIntFromString(args[3])
			if !ok {
				return ammtypes.ErrInvalidAmount
			}
			msg := &ammtypes.MsgSwapExactIn{
				Sender: clientCtx.GetFromAddress().String(), PoolID: pid,
				TokenIn: tokenIn, DenomOut: args[2], MinOut: minOut,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSwapExactOut() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "swap-exact-out [pool-id] [denom-in] [token-out] [max-in]",
		Args:  cobra.ExactArgs(4),
		Short: "Swap to a fixed output amount",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			tokenOut, err := sdk.ParseCoinNormalized(args[2])
			if err != nil {
				return err
			}
			maxIn, ok := math.NewIntFromString(args[3])
			if !ok {
				return ammtypes.ErrInvalidAmount
			}
			msg := &ammtypes.MsgSwapExactOut{
				Sender: clientCtx.GetFromAddress().String(), PoolID: pid,
				DenomIn: args[1], TokenOut: tokenOut, MaxIn: maxIn,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdPausePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-pool [pool-id] [reason]",
		Args:  cobra.ExactArgs(2),
		Short: "Pause a pool (authority = --from)",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			msg := &ammtypes.MsgPausePool{
				Authority: clientCtx.GetFromAddress().String(), PoolID: pid, Reason: args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdResumePool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-pool [pool-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Resume a paused pool (authority = --from)",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pid, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}
			msg := &ammtypes.MsgResumePool{
				Authority: clientCtx.GetFromAddress().String(), PoolID: pid,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

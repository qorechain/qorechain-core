package cli

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/multilayer/types"
)

// GetTxCmd returns the transaction commands for the multilayer module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the multilayer module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		cmdRegisterSidechain(),
		cmdRegisterPaychain(),
		cmdAnchorState(),
		cmdRouteTransaction(),
		cmdUpdateLayerStatus(),
		cmdChallengeAnchor(),
	)
	return cmd
}

func cmdRegisterSidechain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-sidechain [layer-id] [description]",
		Short: "Register a new sidechain layer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			blockTime, _ := cmd.Flags().GetUint64("block-time-ms")
			maxTx, _ := cmd.Flags().GetUint64("max-tx")
			minVals, _ := cmd.Flags().GetUint32("min-validators")
			settle, _ := cmd.Flags().GetUint64("settlement-interval")
			vmTypes, _ := cmd.Flags().GetString("vm-types")
			domains, _ := cmd.Flags().GetString("domains")
			msg := &types.MsgRegisterSidechain{
				Creator:                  clientCtx.GetFromAddress().String(),
				LayerID:                  args[0],
				Description:              args[1],
				TargetBlockTimeMs:        blockTime,
				MaxTransactionsPerBlock:  maxTx,
				MinValidators:            minVals,
				SettlementIntervalBlocks: settle,
				SupportedVMTypes:         splitCSV(vmTypes),
				SupportedDomains:         splitCSV(domains),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64("block-time-ms", 2000, "target block time (ms)")
	cmd.Flags().Uint64("max-tx", 1000, "max transactions per block")
	cmd.Flags().Uint32("min-validators", 1, "minimum validator set size")
	cmd.Flags().Uint64("settlement-interval", 100, "settlement interval (blocks)")
	cmd.Flags().String("vm-types", "evm", "comma-separated supported VM types")
	cmd.Flags().String("domains", "defi", "comma-separated supported domains")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdRegisterPaychain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-paychain [layer-id] [description]",
		Short: "Register a new paychain layer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			maxTx, _ := cmd.Flags().GetUint64("max-tx")
			settle, _ := cmd.Flags().GetUint64("settlement-interval")
			mult, _ := cmd.Flags().GetString("base-fee-multiplier")
			msg := &types.MsgRegisterPaychain{
				Creator:                  clientCtx.GetFromAddress().String(),
				LayerID:                  args[0],
				Description:              args[1],
				MaxTransactionsPerBlock:  maxTx,
				SettlementIntervalBlocks: settle,
				BaseFeeMultiplier:        mult,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64("max-tx", 5000, "max transactions per block")
	cmd.Flags().Uint64("settlement-interval", 50, "settlement interval (blocks)")
	cmd.Flags().String("base-fee-multiplier", "0.01", "fee multiplier vs main chain")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdAnchorState() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anchor-state [layer-id] [layer-height] [state-root-hex] [pqc-agg-sig-hex]",
		Short: "Anchor a subsidiary-chain state root",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			h, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			root, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}
			sig, err := hex.DecodeString(args[3])
			if err != nil {
				return err
			}
			msg := &types.MsgAnchorState{
				Relayer:               clientCtx.GetFromAddress().String(),
				LayerID:               args[0],
				LayerHeight:           h,
				StateRoot:             root,
				PQCAggregateSignature: sig,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdRouteTransaction() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "route-transaction [payload-hex]",
		Short: "Request QCAI routing for a transaction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			payload, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			maxLat, _ := cmd.Flags().GetUint64("max-latency-ms")
			maxFee, _ := cmd.Flags().GetString("max-fee")
			pref, _ := cmd.Flags().GetString("preferred-layer")
			msg := &types.MsgRouteTransaction{
				Sender:             clientCtx.GetFromAddress().String(),
				TransactionPayload: payload,
				PreferredLayer:     pref,
				MaxLatencyMs:       maxLat,
				MaxFee:             maxFee,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().Uint64("max-latency-ms", 10000, "max acceptable latency (ms)")
	cmd.Flags().String("max-fee", "1000000", "max fee (uqor)")
	cmd.Flags().String("preferred-layer", "", "preferred layer hint")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdUpdateLayerStatus() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-layer-status [layer-id] [new-status] [reason]",
		Short: "Update a layer's status (proposed|active|suspended|decommissioned)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgUpdateLayerStatus{
				Authority: clientCtx.GetFromAddress().String(),
				LayerID:   args[0],
				NewStatus: types.LayerStatus(args[1]),
				Reason:    args[2],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdChallengeAnchor() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-anchor [layer-id] [anchor-height] [fraud-proof-hex] [reason]",
		Short: "Challenge a state anchor",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			h, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			proof, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}
			msg := &types.MsgChallengeAnchor{
				Challenger:      clientCtx.GetFromAddress().String(),
				LayerID:         args[0],
				AnchorHeight:    h,
				FraudProof:      proof,
				ChallengeReason: args[3],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

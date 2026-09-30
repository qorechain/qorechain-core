package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	multilayertypes "github.com/qorechain/qorechain-core/x/multilayer/types"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// CmdQueryBatches lists a rollup's batches in index order, one page at a time.
func CmdQueryBatches() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batches [rollup-id]",
		Short: "List a rollup's settlement batches in index order (--start, --limit up to 100)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			start, _ := cmd.Flags().GetUint64("start")
			limit, _ := cmd.Flags().GetUint32("limit")
			res, err := types.NewQueryClient(clientCtx).Batches(cmd.Context(), &types.QueryBatchesRequest{
				RollupId: args[0], StartIndex: start, Limit: limit,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	cmd.Flags().Uint64("start", 0, "first batch index")
	cmd.Flags().Uint32("limit", 0, "batches per page (0 = 50, at most 100)")
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// CmdAnchorSignBytes prints, offline, the bytes the rollup creator signs to
// anchor a batch (MsgSubmitBatch.anchor_signature, from v3.2.2).
func CmdAnchorSignBytes() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "anchor-sign-bytes [layer-id] [batch-index] [state-root-hex]",
		Short: "Print (hex) the v2 anchor sign bytes a rollup creator signs for a batch",
		Long: "Prints the x/multilayer v2 anchor sign bytes for a batch. The layer id is the rollup's layer_id " +
			"(the rollup id for rollups created by x/rdk). Sign the bytes with the rollup creator's registered " +
			"Dilithium key and pass the signature to 'tx rdk submit-batch --anchor-signature'. No network access.",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if chainID == "" {
				chainID = clientCtx.ChainID
			}
			if chainID == "" {
				return fmt.Errorf("--%s is required", flags.FlagChainID)
			}
			idx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			root, err := hex.DecodeString(strings.TrimPrefix(args[2], "0x"))
			if err != nil {
				return err
			}
			vsh, err := hexFlag(cmd, "validator-set-hash")
			if err != nil {
				return err
			}
			txCount, _ := cmd.Flags().GetUint64("tx-count")
			bz := multilayertypes.AnchorSignBytesV2(chainID, args[0], idx, root, vsh, txCount, nil)
			_, err = fmt.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(bz))
			return err
		},
	}
	cmd.Flags().String(flags.FlagChainID, "", "chain id the anchor is for")
	cmd.Flags().Uint64("tx-count", 0, "the batch's tx count (must match submit-batch --tx-count)")
	cmd.Flags().String("validator-set-hash", "", "hex validator-set commitment (must match submit-batch --validator-set-hash)")
	return cmd
}

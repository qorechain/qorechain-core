package cli

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// GetTxCmd returns the CLI transaction commands for the rdk module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the rdk module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	cmd.AddCommand(
		cmdCreateRollup(),
		cmdSubmitBatch(),
		cmdChallengeBatch(),
		cmdPauseRollup(),
		cmdResumeRollup(),
		cmdStopRollup(),
		cmdExecuteWithdrawal(),
		cmdSetVerificationKey(),
		cmdPostData(),
		cmdDepositToRollup(),
		cmdBondSubmitter(),
		cmdUnbondSubmitter(),
		cmdSetMembers(),
		cmdPostPrivateData(),
		cmdDecryptPrivateData(),
	)
	return cmd
}

// readVerificationKeyFile reads a verification key file: the binary key as
// written (gnark VerifyingKey.WriteTo for a Groth16 key, the 16 QSTK bytes for
// a STARK key), or the same bytes in hex.
func readVerificationKeyFile(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if text := strings.TrimSpace(string(raw)); text != "" {
		if b, err := hex.DecodeString(strings.TrimPrefix(text, "0x")); err == nil {
			return b, nil
		}
	}
	return raw, nil
}

func cmdSetVerificationKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-verification-key [rollup-id] [key-file]",
		Short: "Register the ZK verification key of a zk rollup you created (chain v3.2.7+), before its first batch",
		Long: `Register or replace the ZK verification key of a zk rollup. Only the rollup's
creator can, and only until the rollup's first batch; after that only governance
can change it (MsgUpdateVerificationKey).

key-file holds the key in binary or hex: for proof system snark, a Groth16
BLS12-381 verifying key written by gnark's VerifyingKey.WriteTo for a circuit
with exactly four public inputs (prev_state_root, state_root, data_hash,
withdrawals_root); for stark, the 16-byte QSTK key. The chain refuses a key that
does not decode.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			vk, err := readVerificationKeyFile(args[1])
			if err != nil {
				return err
			}
			system, _ := cmd.Flags().GetString("proof-system")
			msg := &types.MsgSetVerificationKey{
				Creator:         clientCtx.GetFromAddress().String(),
				RollupID:        args[0],
				VerificationKey: vk,
				ProofSystem:     system,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("proof-system", "", "snark or stark; empty keeps the rollup's proof system")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdExecuteWithdrawal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-withdrawal [rollup-id] [batch-index] [withdrawal-index] [recipient] [denom] [amount]",
		Short: "Finalize an L2->L1 withdrawal against a finalized batch's withdrawals root",
		Long: "Proves a withdrawal leaf is committed in the batch's withdrawals_root and pays the recipient " +
			"from the rdk module escrow. Permissionless: anyone may submit a valid proof. Pass the Merkle " +
			"sibling hashes (leaf->root) as a comma-separated list of hex strings via --proof.",
		Args: cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			batchIdx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			wIdx, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}
			amount, err := strconv.ParseInt(args[5], 10, 64)
			if err != nil {
				return err
			}
			proofCSV, _ := cmd.Flags().GetString("proof")
			var proof [][]byte
			if proofCSV != "" {
				for _, h := range strings.Split(proofCSV, ",") {
					sib, err := hex.DecodeString(strings.TrimSpace(h))
					if err != nil {
						return err
					}
					proof = append(proof, sib)
				}
			}
			msg := &types.MsgExecuteWithdrawal{
				Submitter:       clientCtx.GetFromAddress().String(),
				RollupID:        args[0],
				BatchIndex:      batchIdx,
				WithdrawalIndex: wIdx,
				Recipient:       args[3],
				Denom:           args[4],
				Amount:          amount,
				Proof:           proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("proof", "", "comma-separated hex Merkle sibling hashes (leaf->root)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdCreateRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create-rollup [rollup-id] [profile] [stake-amount]",
		Short: "Create an application-specific rollup (profile: defi|gaming|nft|enterprise|custom)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			stake, err := strconv.ParseInt(args[2], 10, 64)
			if err != nil {
				return err
			}
			// The chain refuses ids outside this grammar from v3.2.2; checking
			// here saves the fee of a transaction that would fail.
			if err := types.ValidateRollupID(args[0]); err != nil {
				return err
			}
			vm, _ := cmd.Flags().GetString("vm")
			msg := &types.MsgCreateRollup{
				Creator:     clientCtx.GetFromAddress().String(),
				RollupID:    args[0],
				Profile:     args[1],
				VmType:      vm,
				StakeAmount: stake,
			}
			msg.Sequencing, _ = cmd.Flags().GetString("sequencing")
			msg.ExecutionVm, _ = cmd.Flags().GetString("execution-vm")
			msg.Privacy, _ = cmd.Flags().GetString("privacy")
			if path, _ := cmd.Flags().GetString("verification-key-file"); path != "" {
				if msg.VerificationKey, err = readVerificationKeyFile(path); err != nil {
					return err
				}
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("vm", "", "rollup VM type (evm|cosmwasm|svm|custom); empty = use the profile's default VM")
	cmd.Flags().String("verification-key-file", "", "zk profiles only (chain v3.2.7+): register this ZK verification key at creation (see set-verification-key)")
	cmd.Flags().String("sequencing", "", "chain v3.2.8+: \"based\" orders the rollup's transactions by their inclusion on this chain (anyone posts data; defi (snark) rollups then accept batches from anyone). defi, gaming and enterprise profiles only")
	cmd.Flags().String("execution-vm", "", "chain v3.2.8+: \"qledger-v1\" makes an optimistic (nft, custom) rollup whose batches a fraud proof re-executes on chain; requires --sequencing based")
	cmd.Flags().String("privacy", "", "chain v3.2.8+: \"encapsulated\" makes a private rollup: posts are ML-KEM-1024 envelopes to its members (defi profile with a Groth16 key only)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSubmitBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submit-batch [rollup-id] [batch-index] [state-root-hex]",
		Short: "Submit a settlement batch for a rollup",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			idx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			stateRoot, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}
			msg := &types.MsgSubmitBatch{
				Sequencer:  clientCtx.GetFromAddress().String(),
				RollupID:   args[0],
				BatchIndex: idx,
				StateRoot:  stateRoot,
			}
			if msg.PrevStateRoot, err = hexFlag(cmd, "prev-state-root"); err != nil {
				return err
			}
			if msg.DataHash, err = hexFlag(cmd, "data-hash"); err != nil {
				return err
			}
			if msg.WithdrawalsRoot, err = hexFlag(cmd, "withdrawals-root"); err != nil {
				return err
			}
			if msg.AnchorSignature, err = hexFlag(cmd, "anchor-signature"); err != nil {
				return err
			}
			if msg.ValidatorSetHash, err = hexFlag(cmd, "validator-set-hash"); err != nil {
				return err
			}
			msg.TxCount, _ = cmd.Flags().GetUint64("tx-count")
			msg.InboxEnd, _ = cmd.Flags().GetUint64("inbox-end")
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("prev-state-root", "", "hex state root the batch starts from (the previous batch's state root)")
	cmd.Flags().String("data-hash", "", "hex hash of the batch data posted as a DA blob")
	cmd.Flags().String("withdrawals-root", "", "hex Merkle root of the batch's L2->L1 withdrawals")
	cmd.Flags().Uint64("tx-count", 0, "number of rollup transactions in the batch")
	cmd.Flags().Uint64("inbox-end", 0, "based rollups (chain v3.2.8+): the last inbox sequence this batch consumes; --data-hash must then be the inbox range hash ('query rdk inbox')")
	cmd.Flags().String("anchor-signature", "", "hex Dilithium signature by the rollup creator over the v2 anchor sign bytes (see 'query rdk anchor-sign-bytes'); the batch is then anchored in x/multilayer")
	cmd.Flags().String("validator-set-hash", "", "hex validator-set commitment carried in the anchor (covered by the anchor signature)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// hexFlag reads an optional hex-encoded flag.
func hexFlag(cmd *cobra.Command, name string) ([]byte, error) {
	v, _ := cmd.Flags().GetString(name)
	if v == "" {
		return nil, nil
	}
	b, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
	if err != nil {
		return nil, fmt.Errorf("--%s: %w", name, err)
	}
	return b, nil
}

func cmdChallengeBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "challenge-batch [rollup-id] [batch-index]",
		Short: "Challenge a settlement batch",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			idx, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			proofHex, _ := cmd.Flags().GetString("proof")
			proof, err := hex.DecodeString(proofHex)
			if err != nil {
				return err
			}
			msg := &types.MsgChallengeBatch{
				Challenger: clientCtx.GetFromAddress().String(),
				RollupID:   args[0],
				BatchIndex: idx,
				Proof:      proof,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("proof", "", "hex-encoded fraud proof")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdPauseRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pause-rollup [rollup-id] [reason]",
		Short: "Pause a rollup",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgPauseRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
				Reason:   args[1],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdResumeRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume-rollup [rollup-id]",
		Short: "Resume a paused rollup",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgResumeRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdStopRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop-rollup [rollup-id]",
		Short: "Stop a rollup and release its stake",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgStopRollup{
				Creator:  clientCtx.GetFromAddress().String(),
				RollupID: args[0],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// cmdPostData sends MsgPostRollupData (chain v3.2.8+).
func cmdPostData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post-data [rollup-id] [data-hex | @file]",
		Short: "Post rollup data on chain (chain v3.2.8+)",
		Long: `Post rollup data on chain. The payload stays in the transaction; the chain
keeps its SHA-256 commitment under the rollup's next inbox sequence. The fee is
charged per billable byte (at least 1 KiB per post); read the current fee and
size caps with 'query rdk params'. Only the rollup's sequencer may post, unless
the rollup is based, where anyone may. Pass the data as hex, or as @path to
read a file's raw bytes.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			var data []byte
			if path, ok := strings.CutPrefix(args[1], "@"); ok {
				if data, err = os.ReadFile(path); err != nil {
					return err
				}
			} else if data, err = hex.DecodeString(strings.TrimPrefix(args[1], "0x")); err != nil {
				return fmt.Errorf("data: %w", err)
			}
			msg := &types.MsgPostRollupData{
				Poster:   clientCtx.GetFromAddress().String(),
				RollupID: args[0],
				Data:     data,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

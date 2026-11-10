package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
	"github.com/qorechain/qorechain-core/x/rdk/privpost"
	"github.com/qorechain/qorechain-core/x/rdk/types"
)

// v3.2.8 settlement-mode commands: qledger-v1 deposits and submitter bonds,
// private-rollup membership, and client-side encryption of private posts.

func cmdDepositToRollup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [rollup-id] [l2-account-hex] [amount-uqor]",
		Short: "Deposit uqor into a qledger-v1 rollup for a 32-byte L2 account id (chain v3.2.8+)",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			acct, err := hex.DecodeString(strings.TrimPrefix(args[1], "0x"))
			if err != nil {
				return fmt.Errorf("l2 account: %w", err)
			}
			amt, err := strconv.ParseUint(args[2], 10, 64)
			if err != nil {
				return err
			}
			msg := &types.MsgDepositToRollup{Depositor: clientCtx.GetFromAddress().String(), RollupID: args[0], Account: acct, Amount: amt}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdBondSubmitter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bond-submitter [rollup-id] [amount-uqor]",
		Short: "Add to your submitter bond on a qledger-v1 rollup (chain v3.2.8+); an upheld fraud proof pays it to the challenger",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			amt, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}
			msg := &types.MsgBondRollupSubmitter{Submitter: clientCtx.GetFromAddress().String(), RollupID: args[0], Amount: amt}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdUnbondSubmitter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unbond-submitter [rollup-id]",
		Short: "Take back your whole submitter bond, a challenge window after your last batch (chain v3.2.8+)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgUnbondRollupSubmitter{Submitter: clientCtx.GetFromAddress().String(), RollupID: args[0]}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdSetMembers() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-members [rollup-id] [address,address,...]",
		Short: "Set a private rollup's member list; each member must publish an ML-KEM-1024 key (chain v3.2.8+)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgSetRollupMembers{Creator: clientCtx.GetFromAddress().String(), RollupID: args[0],
				Members: strings.Split(args[1], ",")}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdPostPrivateData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post-private-data [rollup-id] [data-hex | @file]",
		Short: "Encrypt data to a private rollup's members and post it (chain v3.2.8+)",
		Long: `Encrypt data on this machine to every member of a private rollup and post the
envelope. The members and their ML-KEM-1024 keys are read from the chain; the
plaintext never leaves this machine. The chain checks only the envelope's
structure and stores its SHA-256.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			data, err := readDataArg(args[1])
			if err != nil {
				return err
			}
			rr, err := types.NewQueryClient(clientCtx).Rollup(cmd.Context(), &types.QueryRollupRequest{RollupId: args[0]})
			if err != nil {
				return err
			}
			r := rr.Rollup
			if r.Privacy != types.PrivacyEncapsulated {
				return fmt.Errorf("rollup %s is not private", args[0])
			}
			pq := pqctypes.NewQueryClient(clientCtx)
			var recips []privpost.Recipient
			for _, m := range r.Members {
				kr, err := pq.KEMKey(cmd.Context(), &pqctypes.QueryKEMKeyRequest{Address: m})
				if err != nil {
					return err
				}
				if !kr.Found || kr.Key.AlgorithmId != uint32(pqctypes.AlgorithmMLKEM1024) {
					return fmt.Errorf("member %s has no ML-KEM-1024 key", m)
				}
				recips = append(recips, privpost.Recipient{Member: m, EncapsulationKey: kr.Key.PublicKey})
			}
			env, err := privpost.Seal(rand.Reader, clientCtx.ChainID, args[0], r.MembersEpoch, recips, data)
			if err != nil {
				return err
			}
			msg := &types.MsgPostRollupData{Poster: clientCtx.GetFromAddress().String(), RollupID: args[0], Data: env}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdDecryptPrivateData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt-private-data [rollup-id] [envelope-hex | @file]",
		Short: "Open a private rollup post with a member's ML-KEM-1024 key (offline)",
		Long: `Open an envelope posted to a private rollup, offline. --kem-key-file holds the
member's ML-KEM-1024 private key: the 64-byte seed or the 3168-byte expanded
key, raw or hex. --chain-id must be the chain the post was made on. The
plaintext is written to --out, or to standard output.`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			env, err := readDataArg(args[1])
			if err != nil {
				return err
			}
			member, _ := cmd.Flags().GetString("member")
			keyPath, _ := cmd.Flags().GetString("kem-key-file")
			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			if member == "" || keyPath == "" || chainID == "" {
				return fmt.Errorf("--member, --kem-key-file and --chain-id are required")
			}
			key, err := os.ReadFile(keyPath)
			if err != nil {
				return err
			}
			if k, err := hex.DecodeString(strings.TrimSpace(string(key))); err == nil {
				key = k
			}
			pt, err := privpost.Open(env, chainID, args[0], member, key)
			if err != nil {
				return err
			}
			if out, _ := cmd.Flags().GetString("out"); out != "" {
				return os.WriteFile(out, pt, 0o600)
			}
			_, err = cmd.OutOrStdout().Write(pt)
			return err
		},
	}
	cmd.Flags().String("member", "", "the member address the envelope was encrypted to")
	cmd.Flags().String("kem-key-file", "", "the member's ML-KEM-1024 private key file")
	cmd.Flags().String("out", "", "write the plaintext to this file (mode 0600)")
	cmd.Flags().String(flags.FlagChainID, "", "chain id the post was made on")
	return cmd
}

func readDataArg(a string) ([]byte, error) {
	if path, ok := strings.CutPrefix(a, "@"); ok {
		return os.ReadFile(path)
	}
	b, err := hex.DecodeString(strings.TrimPrefix(a, "0x"))
	if err != nil {
		return nil, fmt.Errorf("data: %w", err)
	}
	return b, nil
}

// CmdQuerySubmitterBond queries a submitter's bond on a qledger-v1 rollup.
func CmdQuerySubmitterBond() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "submitter-bond [rollup-id] [submitter]",
		Short: "Query a submitter's bond on a qledger-v1 rollup and when it can be unbonded",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			res, err := types.NewQueryClient(clientCtx).SubmitterBond(cmd.Context(),
				&types.QuerySubmitterBondRequest{RollupId: args[0], Submitter: args[1]})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}
	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

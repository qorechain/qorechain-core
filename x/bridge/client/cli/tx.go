package cli

import (
	"encoding/hex"
	"strings"

	"github.com/spf13/cobra"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/bridge/types"
)

// GetTxCmd returns the transaction commands for the bridge module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Transaction commands for the bridge module",
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		cmdBridgeDeposit(),
		cmdBridgeWithdraw(),
		cmdRegisterBridgeValidator(),
		cmdBridgeAttestation(),
		cmdUpdateChainConfig(),
		cmdSetVerifierBootstrap(),
	)
	return cmd
}

// cmdUpdateChainConfig activates/updates a chain's bridge config (bridge_admin or
// qcb_bridge license required). Flip --status active + --verifier <name> to bring
// a pending chain online post-deploy, no governance.
func cmdUpdateChainConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-chain-config [chain-id]",
		Short: "Set a chain's bridge config + active verifier (bridge_admin/qcb_bridge only)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			contract, _ := cmd.Flags().GetString("contract")
			confs, _ := cmd.Flags().GetUint32("confirmations")
			arch, _ := cmd.Flags().GetString("architecture")
			status, _ := cmd.Flags().GetString("status")
			verifier, _ := cmd.Flags().GetString("verifier")
			lockSig, _ := cmd.Flags().GetString("lock-event-sig")
			msg := &types.MsgUpdateChainConfig{
				Admin:                 clientCtx.GetFromAddress().String(),
				ChainId:               args[0],
				BridgeContract:        contract,
				ConfirmationsRequired: confs,
				Architecture:          arch,
				Status:                status,
				Verifier:              verifier,
				LockEventSig:          lockSig,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("contract", "", "bridge/lock contract address on the external chain")
	cmd.Flags().Uint32("confirmations", 0, "confirmations required on the source chain")
	cmd.Flags().String("architecture", "", "chain architecture (empty keeps existing)")
	cmd.Flags().String("status", "", "active|paused|pending (empty keeps existing)")
	cmd.Flags().String("verifier", "", "light_client|wormhole|ed25519|bls|starknet|l2_anchored|bitcoin_spv (empty=attestation)")
	cmd.Flags().String("lock-event-sig", "", "hex topic0 of the lock event (EVM/light-client chains)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// cmdSetVerifierBootstrap installs a verifier trust root. KIND is one of
// wormhole|ed25519|bls|bitcoin|starknet. Byte inputs are hex (comma-separated for lists).
func cmdSetVerifierBootstrap() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-verifier-bootstrap [kind] [chain-id]",
		Short: "Install a verifier trust root for a chain (bridge_admin/qcb_bridge only)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			kind, chainID := args[0], args[1]
			msg := &types.MsgSetVerifierBootstrap{Admin: clientCtx.GetFromAddress().String(), ChainId: chainID}
			n, _ := cmd.Flags().GetUint32("threshold")
			switch kind {
			case "wormhole":
				addrs, err := hexList(cmd, "guardians")
				if err != nil {
					return err
				}
				q, _ := cmd.Flags().GetUint32("quorum")
				msg.Wormhole = &types.WormholeGuardianSet{Addresses: addrs, Quorum: q}
			case "ed25519":
				pks, err := hexList(cmd, "pubkeys")
				if err != nil {
					return err
				}
				msg.Ed25519 = &types.ValidatorQuorum{Pubkeys: pks, Threshold: n}
			case "bls":
				pks, err := hexList(cmd, "pubkeys")
				if err != nil {
					return err
				}
				msg.Bls = &types.ValidatorQuorum{Pubkeys: pks, Threshold: n}
			case "bitcoin":
				bh, err := hexFlag(cmd, "block-hash")
				if err != nil {
					return err
				}
				mc, _ := cmd.Flags().GetUint32("min-confs")
				msg.Bitcoin = &types.BitcoinCheckpoint{BlockHash: bh, MinConfs: mc}
			case "starknet":
				root, err := hexFlag(cmd, "state-root")
				if err != nil {
					return err
				}
				msg.StarknetStateRoot = root
			default:
				return types.ErrChainNotSupported.Wrapf("unknown verifier kind %q (want wormhole|ed25519|bls|bitcoin|starknet)", kind)
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("guardians", "", "wormhole: comma-separated hex guardian addresses")
	cmd.Flags().Uint32("quorum", 0, "wormhole: guardian quorum")
	cmd.Flags().String("pubkeys", "", "ed25519/bls: comma-separated hex pubkeys")
	cmd.Flags().Uint32("threshold", 0, "ed25519/bls: signature threshold")
	cmd.Flags().String("block-hash", "", "bitcoin: hex checkpoint block hash")
	cmd.Flags().Uint32("min-confs", 0, "bitcoin: minimum confirmations")
	cmd.Flags().String("state-root", "", "starknet: hex state root")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func hexFlag(cmd *cobra.Command, name string) ([]byte, error) {
	s, _ := cmd.Flags().GetString(name)
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}

func hexList(cmd *cobra.Command, name string) ([][]byte, error) {
	s, _ := cmd.Flags().GetString(name)
	if s == "" {
		return nil, nil
	}
	var out [][]byte
	for _, part := range strings.Split(s, ",") {
		b, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(part), "0x"))
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func cmdBridgeDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deposit [source-chain] [source-tx-hash] [asset] [amount]",
		Short: "Credit assets bridged in from a source chain",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			sigsHex, _ := cmd.Flags().GetString("validator-sigs")
			commitHex, _ := cmd.Flags().GetString("pqc-commitment")
			sigs, err := hex.DecodeString(sigsHex)
			if err != nil {
				return err
			}
			commit, err := hex.DecodeString(commitHex)
			if err != nil {
				return err
			}
			msg := &types.MsgBridgeDeposit{
				Sender:              clientCtx.GetFromAddress().String(),
				SourceChain:         args[0],
				SourceTxHash:        args[1],
				Asset:               args[2],
				Amount:              args[3],
				BridgeValidatorSigs: sigs,
				PQCCommitment:       commit,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("validator-sigs", "", "hex-encoded bridge validator signatures")
	cmd.Flags().String("pqc-commitment", "", "hex-encoded PQC commitment")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdBridgeWithdraw() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw [destination-chain] [destination-address] [asset] [amount]",
		Short: "Initiate a withdrawal to a destination chain",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg := &types.MsgBridgeWithdraw{
				Sender:             clientCtx.GetFromAddress().String(),
				DestinationChain:   args[0],
				DestinationAddress: args[1],
				Asset:              args[2],
				Amount:             args[3],
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdRegisterBridgeValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-validator [pqc-pubkey-hex] [supported-chains-csv]",
		Short: "Register a validator for bridge attestation duty",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pubkey, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}
			msg := &types.MsgRegisterBridgeValidator{
				ValidatorAddress: clientCtx.GetFromAddress().String(),
				PQCPubkey:        pubkey,
				SupportedChains:  strings.Split(args[1], ","),
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func cmdBridgeAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attest [chain] [event-type] [operation-id] [tx-hash] [amount] [asset]",
		Short: "Submit a validator attestation for a bridge operation",
		Args:  cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			amount, ok := sdkmath.NewIntFromString(args[4])
			if !ok {
				return types.ErrInvalidAmount.Wrapf("invalid amount: %s", args[4])
			}
			sigHex, _ := cmd.Flags().GetString("pqc-signature")
			proofHex, _ := cmd.Flags().GetString("proof")
			sig, err := hex.DecodeString(sigHex)
			if err != nil {
				return err
			}
			proof, err := hex.DecodeString(proofHex)
			if err != nil {
				return err
			}
			msg := &types.MsgBridgeAttestation{
				Validator:    clientCtx.GetFromAddress().String(),
				Chain:        args[0],
				EventType:    args[1],
				OperationID:  args[2],
				TxHash:       args[3],
				Amount:       amount,
				Asset:        args[5],
				Proof:        proof,
				PQCSignature: sig,
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}
	cmd.Flags().String("pqc-signature", "", "hex-encoded PQC signature")
	cmd.Flags().String("proof", "", "hex-encoded proof")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

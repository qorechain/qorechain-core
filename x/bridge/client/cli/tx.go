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
	)
	return cmd
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

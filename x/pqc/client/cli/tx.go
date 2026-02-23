package cli

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// GetTxCmd returns the transaction commands for the pqc module.
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "PQC module transaction commands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdRegisterPQCKey(),
		GetCmdRegisterPQCKeyV2(),
		GetCmdMigratePQCKey(),
	)

	return cmd
}

// GetCmdRegisterPQCKey returns the command to register a legacy PQC key (defaults to Dilithium-5).
func GetCmdRegisterPQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-key [pubkey-hex] [key-type]",
		Short: "Register a PQC key for your account (legacy, defaults to Dilithium-5)",
		Long: `Register a post-quantum cryptographic key for your account.
This is the legacy command that defaults to Dilithium-5. Use 'register-key-v2' for explicit algorithm selection.

Key types: hybrid, pqc_only, classical_only
  - hybrid: Both PQC and ECDSA keys (recommended)
  - pqc_only: Only PQC key, no classical fallback
  - classical_only: Only ECDSA key (no PQC protection)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			pubkeyBytes, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("invalid hex-encoded public key: %w", err)
			}

			keyType := args[1]
			if keyType != types.KeyTypeHybrid && keyType != types.KeyTypePQCOnly && keyType != types.KeyTypeClassicalOnly {
				return fmt.Errorf("invalid key type: %s (must be hybrid, pqc_only, or classical_only)", keyType)
			}

			msg := &types.MsgRegisterPQCKey{
				Sender:          clientCtx.GetFromAddress().String(),
				DilithiumPubkey: pubkeyBytes,
				KeyType:         keyType,
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

// GetCmdRegisterPQCKeyV2 returns the command to register a PQC key with explicit algorithm selection.
func GetCmdRegisterPQCKeyV2() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-key-v2 [algorithm] [pubkey-hex] [key-type]",
		Short: "Register a PQC key with explicit algorithm selection (v0.6.0)",
		Long: `Register a post-quantum cryptographic key for your account with explicit algorithm selection.

Supported algorithms:
  - dilithium5 (or 1): NIST FIPS 204 digital signature scheme
  - mlkem1024 (or 2): NIST FIPS 203 key encapsulation mechanism

Key types: hybrid, pqc_only, classical_only`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Parse algorithm ID
			algoID, err := parseAlgorithmID(args[0])
			if err != nil {
				return err
			}

			pubkeyBytes, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid hex-encoded public key: %w", err)
			}

			keyType := args[2]
			if keyType != types.KeyTypeHybrid && keyType != types.KeyTypePQCOnly && keyType != types.KeyTypeClassicalOnly {
				return fmt.Errorf("invalid key type: %s (must be hybrid, pqc_only, or classical_only)", keyType)
			}

			msg := &types.MsgRegisterPQCKeyV2{
				Sender:      clientCtx.GetFromAddress().String(),
				PublicKey:   pubkeyBytes,
				AlgorithmID: algoID,
				KeyType:     keyType,
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

// GetCmdMigratePQCKey returns the command to migrate a PQC key to a new algorithm.
func GetCmdMigratePQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate-key [new-algorithm] [old-pubkey-hex] [new-pubkey-hex] [old-sig-hex] [new-sig-hex]",
		Short: "Migrate your PQC key to a new algorithm (dual-signature required)",
		Long: `Migrate your account's PQC key from one algorithm to another.

This requires dual signatures - one from your old key and one from your new key - proving
ownership of both. A migration must be active for your current algorithm (started via governance).

The message signed by both keys is: "migrate:<your-address>"`,
		Args: cobra.ExactArgs(5),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			newAlgoID, err := parseAlgorithmID(args[0])
			if err != nil {
				return err
			}

			oldPubkey, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("invalid old public key hex: %w", err)
			}

			newPubkey, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("invalid new public key hex: %w", err)
			}

			oldSig, err := hex.DecodeString(args[3])
			if err != nil {
				return fmt.Errorf("invalid old signature hex: %w", err)
			}

			newSig, err := hex.DecodeString(args[4])
			if err != nil {
				return fmt.Errorf("invalid new signature hex: %w", err)
			}

			msg := &types.MsgMigratePQCKey{
				Sender:         clientCtx.GetFromAddress().String(),
				OldPublicKey:   oldPubkey,
				NewPublicKey:   newPubkey,
				NewAlgorithmID: newAlgoID,
				OldSignature:   oldSig,
				NewSignature:   newSig,
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

// parseAlgorithmID parses an algorithm identifier from a string (name or numeric ID).
func parseAlgorithmID(s string) (types.AlgorithmID, error) {
	// Try parsing as a name first
	if id, err := types.AlgorithmIDFromString(s); err == nil {
		return id, nil
	}

	// Try parsing as a number
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return types.AlgorithmUnspecified, fmt.Errorf("invalid algorithm: %s (use name like 'dilithium5' or numeric ID)", s)
	}

	id := types.AlgorithmID(n)
	if id == types.AlgorithmUnspecified {
		return types.AlgorithmUnspecified, fmt.Errorf("algorithm ID 0 is reserved (unspecified)")
	}

	return id, nil
}

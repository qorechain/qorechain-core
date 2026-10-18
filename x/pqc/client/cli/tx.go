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
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdRegisterPQCKey(),
		GetCmdRegisterPQCKeyV2(),
		GetCmdMigratePQCKey(),
		GetCmdOpenEVMWindow(),
		GetCmdCloseEVMWindow(),
		GetCmdAuthorizeEVMDelegation(),
		GetCmdClearEVMDelegation(),
	)
	// Client-side hybrid signer (gen-key, cosign) — real in the full build,
	// placeholders in the community build.
	cmd.AddCommand(pqcSignerCommands()...)

	return cmd
}

// GetCmdRegisterPQCKey returns the command to register a legacy PQC key (defaults to Dilithium-5).
func GetCmdRegisterPQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-key [pubkey-hex] [key-type]",
		Short: "Register a PQC key for your account (legacy, defaults to Dilithium-5)",
		Long: `Register a post-quantum cryptographic key for your account.
This is the legacy command that defaults to Dilithium-5. Use 'register-key-v2' for explicit algorithm selection.

Key types: hybrid, pqc_only
  - hybrid: Both PQC and ECDSA keys (recommended)
  - pqc_only: Only PQC key, no classical fallback

classical_only is refused: it registers no post-quantum key, and on a chain
that requires the hybrid signature the account could never sign again.`,
		Example: `  # Register a hybrid Dilithium-5 key (recommended)
  qorechaind tx pqc register-key <dilithium5-pubkey-hex> hybrid --from mykey

  # Register a PQC-only key (no classical fallback)
  qorechaind tx pqc register-key <dilithium5-pubkey-hex> pqc_only --from mykey`,
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
			if err := checkRegisterKeyType(keyType); err != nil {
				return err
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

Key types: hybrid, pqc_only (classical_only is refused, see register-key)

With --pqc-key <name> (a key from gen-key or recover-key) the command also
attaches a proof of possession: the key's signature over
"qorechain-pqc-pop-v1|chainid|algo|address|hex(pubkey)". The chain verifies it
when present.`,
		Example: `  # Register a Dilithium-5 hybrid key
  qorechaind tx pqc register-key-v2 dilithium5 <pubkey-hex> hybrid --from mykey

  # Register an ML-KEM-1024 PQC-only key
  qorechaind tx pqc register-key-v2 mlkem1024 <pubkey-hex> pqc_only --from mykey`,
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
			if err := checkRegisterKeyType(keyType); err != nil {
				return err
			}

			msg := &types.MsgRegisterPQCKeyV2{
				Sender:      clientCtx.GetFromAddress().String(),
				PublicKey:   pubkeyBytes,
				AlgorithmID: algoID,
				KeyType:     keyType,
			}
			if keyName, _ := cmd.Flags().GetString(flagPQCKey); keyName != "" {
				pop, err := proofOfPossession(clientCtx.HomeDir, keyName, clientCtx.ChainID, algoID, msg.Sender, pubkeyBytes)
				if err != nil {
					return err
				}
				msg.ProofOfPossession = pop
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(flagPQCKey, "", "name of the local PQC key (gen-key / recover-key) to sign a proof of possession with")
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

Both keys sign types.MigrationSignBytes(chain-id, your-address, from-algorithm,
new-algorithm, execution-height, old-pubkey, new-pubkey): the destination key is
part of what the old key vouches for, so the signatures cannot be reused with a
different new key. The execution height is the height the transaction lands at;
sign for the next block and broadcast immediately.`,
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

// flagPQCKey names a local PQC key file under <home>/pqc.
const flagPQCKey = "pqc-key"

// checkRegisterKeyType refuses key types that would lock the account. A
// classical_only registration stores no post-quantum key; under the
// hybrid-required posture the account can then never produce the signature
// every later transaction needs, and before v3.2.3 nothing could repair it
// (QSR-2026-0550, 0400).
func checkRegisterKeyType(keyType string) error {
	switch keyType {
	case types.KeyTypeHybrid, types.KeyTypePQCOnly:
		return nil
	case types.KeyTypeClassicalOnly:
		return fmt.Errorf("refusing key type classical_only: it registers no post-quantum key and " +
			"leaves the account unable to sign under the hybrid-required posture; use hybrid")
	default:
		return fmt.Errorf("invalid key type: %s (must be hybrid or pqc_only)", keyType)
	}
}

// SignProofOfPossession is proofOfPossession for other modules' commands
// (the bridge validator registration uses the same proof format): it signs
// the x/pqc proof of possession for sender and pub with the local key
// keyName, and checks the key matches pub before anything is broadcast.
func SignProofOfPossession(home, keyName, chainID string, sender string, pub []byte) ([]byte, error) {
	return proofOfPossession(home, keyName, chainID, types.AlgorithmDilithium5, sender, pub)
}

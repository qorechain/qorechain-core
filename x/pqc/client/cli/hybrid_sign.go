//go:build full

package cli

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// pqcSignerCommands returns the full-build PQC client-signer commands
// (gen-key, cosign). These require the FFI library and so are only present in
// the full (validator) build; the community build provides stubs.
func pqcSignerCommands() []*cobra.Command {
	return []*cobra.Command{getCmdGenPQCKey(), getCmdCosign(), getCmdRecoverPQCKey(), getCmdRotatePQCKey()}
}

// deriveDilithiumSeed returns the FIPS-204 seed for a derivation name. "adapter"
// (SDK / @qorechain/wallet-adapter, address-bound) is the canonical ecosystem
// derivation; "bridge" (@qorechain/chain-bridge, faucet-api / dashboard backends)
// is the legacy mnemonic-only derivation. See [[chain-pqc-standards-migration]].
func deriveDilithiumSeed(derivation, address, mnemonic string) ([]byte, error) {
	switch derivation {
	case "bridge", "mnemonic-only":
		return qpqc.Shake256([]byte(mnemonic), 32), nil
	case "adapter", "":
		return qpqc.Shake256([]byte("qorechain:pqc:v1|"+address+"|"+mnemonic), 32), nil
	default:
		return nil, fmt.Errorf("unknown derivation %q (use adapter|bridge)", derivation)
	}
}

// getCmdRecoverPQCKey deterministically reconstructs an account's ML-DSA-87 key
// from its BIP-39 mnemonic using the ecosystem-standard derivation, so a key can
// be recovered (or reproduced on a new host) without a random keygen. The
// mnemonic is read from stdin so it never lands in shell history or process args.
func getCmdRecoverPQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover-key [name] [address]",
		Short: "Recover a Dilithium-5 key deterministically from a mnemonic (read from stdin)",
		Long: `Reconstruct the canonical ML-DSA-87 keypair for [address] from its BIP-39
mnemonic, using the ecosystem-standard derivation
shake256("qorechain:pqc:v1|" + address + "|" + mnemonic) as the FIPS-204 seed —
byte-identical to the SDK / wallet-adapter, so the recovered key matches the one
registered on-chain. The mnemonic is read from STDIN. Stores the private key under
<home>/pqc/<name>.dilithium (0600) for use with ` + "`tx pqc cosign`" + `:

  echo "<24 words>" | qorechaind tx pqc recover-key mykey <qor1address>`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			name, address := args[0], args[1]
			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			mnemonic := strings.TrimSpace(string(data))
			if mnemonic == "" {
				return fmt.Errorf("no mnemonic on stdin (pipe the BIP-39 phrase in)")
			}
			// Two derivations exist in the ecosystem — pick the one the key was
			// registered with. "adapter" (SDK/@qorechain/wallet-adapter, address-
			// bound) vs "bridge" (@qorechain/chain-bridge, faucet-api / dashboard
			// backends — mnemonic only).
			derivation, _ := cmd.Flags().GetString("derivation")
			seed, err := deriveDilithiumSeed(derivation, address, mnemonic)
			if err != nil {
				return err
			}
			pk, sk, err := qpqc.MLDSA87.KeygenFromSeed(seed)
			if err != nil {
				return fmt.Errorf("keygen from seed: %w", err)
			}
			dir := pqcKeyDir(clientCtx.HomeDir)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			if err := os.WriteFile(pqcKeyPath(clientCtx.HomeDir, name), []byte(hex.EncodeToString(sk)), 0o600); err != nil {
				return err
			}
			fmt.Printf("recovered Dilithium-5 private key: %s\n", pqcKeyPath(clientCtx.HomeDir, name))
			fmt.Printf("public_key_hex: %s\n", hex.EncodeToString(pk))
			return nil
		},
	}
	cmd.Flags().String("derivation", "adapter", "seed derivation: adapter (shake256(\"qorechain:pqc:v1|addr|mnemonic\"), SDK/wallet-adapter) | bridge (shake256(mnemonic), chain-bridge/faucet-api)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func pqcKeyDir(home string) string  { return filepath.Join(home, "pqc") }
func pqcKeyPath(home, name string) string {
	return filepath.Join(pqcKeyDir(home), name+".dilithium")
}

// getCmdGenPQCKey generates a Dilithium-5 keypair, stores the private key under
// <home>/pqc/<name>.dilithium (0600), and prints the public key hex so it can be
// registered with `tx pqc register-key <pubkey-hex> hybrid`.
func getCmdGenPQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen-key [name]",
		Short: "Generate and store a Dilithium-5 key for client-side hybrid signing",
		Long: `Generate a Dilithium-5 (ML-DSA-87) keypair via the PQC FFI, store the
private key under <home>/pqc/<name>.dilithium (mode 0600), and print the public
key hex. Register it on-chain with:

  qorechaind tx pqc register-key <printed-pubkey-hex> hybrid --from <key>

then sign transactions with `+"`tx pqc cosign`"+`.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			pk, sk, err := qpqc.MLDSA87.Keygen()
			if err != nil {
				return fmt.Errorf("dilithium keygen: %w", err)
			}
			dir := pqcKeyDir(clientCtx.HomeDir)
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}
			path := pqcKeyPath(clientCtx.HomeDir, name)
			if err := os.WriteFile(path, []byte(hex.EncodeToString(sk)), 0o600); err != nil {
				return err
			}
			fmt.Printf("stored Dilithium-5 private key: %s\n", path)
			fmt.Printf("public_key_hex: %s\n", hex.EncodeToString(pk))
			fmt.Printf("\nregister it on-chain:\n  qorechaind tx pqc register-key %s hybrid --from <key>\n", hex.EncodeToString(pk))
			return nil
		},
	}
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// getCmdCosign co-signs a generate-only transaction with a Dilithium-5 hybrid
// extension AND the account's classical secp256k1 signature, then broadcasts it.
// This is the client side of the chain's PQC hybrid verification.
func getCmdCosign() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cosign [unsigned-tx.json]",
		Short: "PQC+classical sign a generate-only tx and broadcast it",
		Long: `Attach a Dilithium-5 hybrid signature (and the account's classical
secp256k1 signature) to a transaction and broadcast it. Produce the input with
any tx command using --generate-only, e.g.:

  qorechaind tx bank send <from> <to> 1000uqor --generate-only > tx.json
  qorechaind tx pqc cosign tx.json --from <from> --pqc-key <name> --chain-id <id>

The --pqc-key name refers to a key created with `+"`tx pqc gen-key`"+`.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pqcKeyName, _ := cmd.Flags().GetString("pqc-key")
			if pqcKeyName == "" {
				return fmt.Errorf("--pqc-key is required (a key created with `tx pqc gen-key`)")
			}

			// 1. Decode the generate-only tx into a raw protobuf Tx so we can read
			//    its messages, memo, timeout, and fee directly.
			bz, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var rawTx txtypes.Tx
			if err := clientCtx.Codec.UnmarshalJSON(bz, &rawTx); err != nil {
				return fmt.Errorf("decode unsigned tx: %w", err)
			}
			if rawTx.Body == nil || rawTx.AuthInfo == nil {
				return fmt.Errorf("unsigned tx is missing body or auth_info")
			}

			// 2. Load the PQC signing key and cosign+broadcast the messages.
			sk, err := loadPQCPrivKey(clientCtx.HomeDir, pqcKeyName)
			if err != nil {
				return err
			}
			return hybridCosignBroadcast(clientCtx, rawTx.Body.Messages, rawTx.Body.Memo, rawTx.Body.TimeoutHeight, rawTx.AuthInfo.Fee, sk)
		},
	}
	cmd.Flags().String("pqc-key", "", "name of the Dilithium key (created with `tx pqc gen-key`)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// hybridCosignBroadcast assembles a TxRaw carrying a Dilithium-5 hybrid extension
// (PQC-signed with pqcSk over frame(B0, authInfo)) plus the from account's
// classical secp256k1 signature, then broadcasts it. This is the shared core of
// `tx pqc cosign` and `tx pqc rotate-key` — anything that must submit a
// PQC-required tx from a key held locally as an ML-DSA-87 secret.
func hybridCosignBroadcast(clientCtx client.Context, msgs []*codectypes.Any, memo string, timeoutHeight uint64, fee *txtypes.Fee, pqcSk []byte) error {
	// Resolve the signing account (classical secp256k1 key).
	fromRec, err := clientCtx.Keyring.Key(clientCtx.FromName)
	if err != nil {
		return err
	}
	fromPub, err := fromRec.GetPubKey()
	if err != nil {
		return err
	}
	fromAddr, err := fromRec.GetAddress()
	if err != nil {
		return err
	}
	accNum, seq, err := clientCtx.AccountRetriever.GetAccountNumberSequence(clientCtx, fromAddr)
	if err != nil {
		return fmt.Errorf("fetch account number/sequence: %w", err)
	}

	// AuthInfo: single DIRECT signer (from account's secp256k1 key) + fee.
	pubAny, err := codectypes.NewAnyWithValue(fromPub)
	if err != nil {
		return err
	}
	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{{
			PublicKey: pubAny,
			ModeInfo: &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{
				Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT},
			}},
			Sequence: seq,
		}},
		Fee: fee,
	}
	authInfoBytes, err := authInfo.Marshal()
	if err != nil {
		return err
	}

	// B0 = canonical body bytes WITHOUT the PQC extension (the chain re-derives
	// the same B0 by stripping the extension).
	bodyNoExt := &txtypes.TxBody{Messages: msgs, Memo: memo, TimeoutHeight: timeoutHeight}
	b0, err := bodyNoExt.Marshal()
	if err != nil {
		return err
	}

	// PQC sign-bytes = BE32(len(B0))||B0||BE32(len(authInfo))||authInfo.
	pqcSig, err := qpqc.MLDSA87.Sign(pqcSk, frame(b0, authInfoBytes))
	if err != nil {
		return fmt.Errorf("dilithium sign: %w", err)
	}

	// Wire body WITH the PQC hybrid extension (proto-encoded so the chain's tx
	// decoder can resolve the registered type URL).
	ext := types.PQCHybridSignature{AlgorithmID: types.AlgorithmDilithium5, PQCSignature: pqcSig}
	extVal, err := ext.Marshal()
	if err != nil {
		return err
	}
	bodyWithExt := &txtypes.TxBody{
		Messages:         msgs,
		Memo:             memo,
		TimeoutHeight:    timeoutHeight,
		ExtensionOptions: []*codectypes.Any{{TypeUrl: types.HybridSigTypeURL, Value: extVal}},
	}
	bodyWithExtBytes, err := bodyWithExt.Marshal()
	if err != nil {
		return err
	}

	// Classical secp256k1 signature over SignDoc{bodyWithExt, authInfo}.
	signDoc := &txtypes.SignDoc{
		BodyBytes:     bodyWithExtBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       clientCtx.ChainID,
		AccountNumber: accNum,
	}
	signDocBytes, err := signDoc.Marshal()
	if err != nil {
		return err
	}
	classicalSig, _, err := clientCtx.Keyring.Sign(clientCtx.FromName, signDocBytes, signing.SignMode_SIGN_MODE_DIRECT)
	if err != nil {
		return fmt.Errorf("classical sign: %w", err)
	}

	txRaw := &txtypes.TxRaw{
		BodyBytes:     bodyWithExtBytes,
		AuthInfoBytes: authInfoBytes,
		Signatures:    [][]byte{classicalSig},
	}
	txBytes, err := txRaw.Marshal()
	if err != nil {
		return err
	}
	res, err := clientCtx.BroadcastTx(txBytes)
	if err != nil {
		return err
	}
	return clientCtx.PrintProto(res)
}

// getCmdRotatePQCKey rotates an account's ML-DSA-87 key to a NEW key of the same
// algorithm in one shot: it recovers the OLD (currently-registered) key from the
// mnemonic, derives or generates the NEW key, dual-signs the domain-separated
// rotation bytes with both, and submits MsgRotatePQCKey cosigned (hybrid) with
// the OLD key — because the account still holds a PQC key, the rotation tx itself
// must pass the PQC hybrid ante under the old key. After success the NEW private
// key is stored locally for future signing.
//
// The canonical use case is moving a legacy-derived key (chain-bridge /
// mnemonic-only, `--old-derivation bridge`) onto the canonical address-bound
// derivation (`--new-derivation adapter`). See [[chain-pqc-standards-migration]].
func getCmdRotatePQCKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate-key",
		Short: "Rotate your ML-DSA-87 key to a new key of the same algorithm (mnemonic on stdin)",
		Long: `Replace your account's PQC key with a NEW key of the SAME algorithm —
to move a legacy-derived key onto the canonical derivation, or to retire a
compromised key. The mnemonic is read from STDIN. Both the OLD and NEW keys sign
the domain-separated bytes "qorechain-pqc-rotate-v1|chainid|algo|address|old|new";
the rotation tx is cosigned (hybrid) with the old key so it passes the PQC ante.

  # move a chain-bridge (legacy) key to the canonical adapter derivation
  echo "<24 words>" | qorechaind tx pqc rotate-key \
    --from mykey --old-derivation bridge --new-derivation adapter \
    --fees 2000uqor --chain-id qorechain-diana

  # rotate a compromised key to a fresh random key stored locally as <name>
  echo "<24 words>" | qorechaind tx pqc rotate-key \
    --from mykey --old-derivation adapter --new-random --new-key-out mykey-new \
    --fees 2000uqor --chain-id qorechain-diana`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := clientCtx.GetFromAddress().String()
			if address == "" {
				return fmt.Errorf("--from is required (the account whose key is rotating)")
			}

			data, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return err
			}
			mnemonic := strings.TrimSpace(string(data))
			if mnemonic == "" {
				return fmt.Errorf("no mnemonic on stdin (pipe the BIP-39 phrase in)")
			}

			// 1. Recover the OLD (currently-registered) key.
			oldDeriv, _ := cmd.Flags().GetString("old-derivation")
			oldSeed, err := deriveDilithiumSeed(oldDeriv, address, mnemonic)
			if err != nil {
				return fmt.Errorf("old key: %w", err)
			}
			oldPub, oldSk, err := qpqc.MLDSA87.KeygenFromSeed(oldSeed)
			if err != nil {
				return fmt.Errorf("recover old key: %w", err)
			}

			// 2. Derive or generate the NEW key.
			var newPub, newSk []byte
			if newRandom, _ := cmd.Flags().GetBool("new-random"); newRandom {
				newPub, newSk, err = qpqc.MLDSA87.Keygen()
				if err != nil {
					return fmt.Errorf("generate new key: %w", err)
				}
			} else {
				newDeriv, _ := cmd.Flags().GetString("new-derivation")
				newSeed, derr := deriveDilithiumSeed(newDeriv, address, mnemonic)
				if derr != nil {
					return fmt.Errorf("new key: %w", derr)
				}
				newPub, newSk, err = qpqc.MLDSA87.KeygenFromSeed(newSeed)
				if err != nil {
					return fmt.Errorf("derive new key: %w", err)
				}
			}
			if hex.EncodeToString(oldPub) == hex.EncodeToString(newPub) {
				return fmt.Errorf("new key equals old key — pick a different --new-derivation or use --new-random")
			}

			// 3. Dual-sign the rotation bytes (algorithm is Dilithium-5 for these
			//    ecosystem-derived keys — the chain re-derives with acct.AlgorithmID).
			signBytes := types.RotationSignBytes(clientCtx.ChainID, uint32(types.AlgorithmDilithium5), address, oldPub, newPub)
			oldSig, err := qpqc.MLDSA87.Sign(oldSk, signBytes)
			if err != nil {
				return fmt.Errorf("old-key rotation sign: %w", err)
			}
			newSig, err := qpqc.MLDSA87.Sign(newSk, signBytes)
			if err != nil {
				return fmt.Errorf("new-key rotation sign: %w", err)
			}

			// 4. Store the NEW private key locally so the account can sign with it
			//    once the rotation lands.
			outName, _ := cmd.Flags().GetString("new-key-out")
			if outName != "" {
				if err := os.MkdirAll(pqcKeyDir(clientCtx.HomeDir), 0o700); err != nil {
					return err
				}
				if err := os.WriteFile(pqcKeyPath(clientCtx.HomeDir, outName), []byte(hex.EncodeToString(newSk)), 0o600); err != nil {
					return err
				}
				fmt.Printf("stored new Dilithium-5 private key: %s\n", pqcKeyPath(clientCtx.HomeDir, outName))
			}
			fmt.Printf("old_public_key: %s\nnew_public_key: %s\n", hex.EncodeToString(oldPub), hex.EncodeToString(newPub))

			// 5. Build MsgRotatePQCKey and cosign+broadcast with the OLD key.
			msg := &types.MsgRotatePQCKey{
				Sender:       address,
				OldPublicKey: oldPub,
				NewPublicKey: newPub,
				OldSignature: oldSig,
				NewSignature: newSig,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			msgAny, err := codectypes.NewAnyWithValue(msg)
			if err != nil {
				return err
			}
			feeStr, _ := cmd.Flags().GetString(flags.FlagFees)
			feeCoins, err := sdk.ParseCoinsNormalized(feeStr)
			if err != nil {
				return fmt.Errorf("invalid --fees %q: %w", feeStr, err)
			}
			gasLimit, _ := cmd.Flags().GetString(flags.FlagGas)
			gas := uint64(500000)
			if gasLimit != "" && gasLimit != flags.GasFlagAuto {
				if g, perr := strconv.ParseUint(gasLimit, 10, 64); perr == nil {
					gas = g
				}
			}
			fee := &txtypes.Fee{Amount: feeCoins, GasLimit: gas}
			return hybridCosignBroadcast(clientCtx, []*codectypes.Any{msgAny}, "", 0, fee, oldSk)
		},
	}
	cmd.Flags().String("old-derivation", "bridge", "derivation of the CURRENT key: bridge (shake256(mnemonic), chain-bridge/faucet-api) | adapter (address-bound, SDK/wallet-adapter)")
	cmd.Flags().String("new-derivation", "adapter", "derivation of the NEW key when not --new-random: adapter | bridge")
	cmd.Flags().Bool("new-random", false, "rotate to a fresh random key instead of a mnemonic-derived one (store it with --new-key-out)")
	cmd.Flags().String("new-key-out", "", "store the new private key under <home>/pqc/<name>.dilithium for future signing")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// frame builds the length-prefixed PQC sign-bytes the chain re-derives in
// ante_hybrid.getSignBytes: BE32(len(b0))||b0||BE32(len(auth))||auth.
func frame(b0, auth []byte) []byte {
	buf := make([]byte, 4+len(b0)+4+len(auth))
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(b0)))
	copy(buf[4:4+len(b0)], b0)
	binary.BigEndian.PutUint32(buf[4+len(b0):8+len(b0)], uint32(len(auth)))
	copy(buf[8+len(b0):], auth)
	return buf
}

func loadPQCPrivKey(home, name string) ([]byte, error) {
	raw, err := os.ReadFile(pqcKeyPath(home, name))
	if err != nil {
		return nil, fmt.Errorf("load PQC key %q: %w (generate one with `tx pqc gen-key`)", name, err)
	}
	return hex.DecodeString(string(raw))
}

//go:build full

package cli

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/qorechain/qorechain-core/x/pqc/ffi"
	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// pqcSignerCommands returns the full-build PQC client-signer commands
// (gen-key, cosign). These require the FFI library and so are only present in
// the full (validator) build; the community build provides stubs.
func pqcSignerCommands() []*cobra.Command {
	return []*cobra.Command{getCmdGenPQCKey(), getCmdCosign()}
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
			pk, sk, err := ffi.NewFFIClient().DilithiumKeygen()
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

			// 2. Resolve the signing account (classical secp256k1 key).
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

			// 3. Build AuthInfo: single DIRECT signer (the from account's secp256k1
			//    key) + the fee from the unsigned tx.
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
				Fee: rawTx.AuthInfo.Fee,
			}
			authInfoBytes, err := authInfo.Marshal()
			if err != nil {
				return err
			}

			// 4. B0 = canonical body bytes WITHOUT the PQC extension (the chain
			//    re-derives the same B0 by stripping the extension).
			bodyNoExt := &txtypes.TxBody{
				Messages:      rawTx.Body.Messages,
				Memo:          rawTx.Body.Memo,
				TimeoutHeight: rawTx.Body.TimeoutHeight,
			}
			b0, err := bodyNoExt.Marshal()
			if err != nil {
				return err
			}

			// 5. PQC sign-bytes = BE32(len(B0))||B0||BE32(len(authInfo))||authInfo.
			pqcSignBytes := frame(b0, authInfoBytes)
			sk, err := loadPQCPrivKey(clientCtx.HomeDir, pqcKeyName)
			if err != nil {
				return err
			}
			pqcSig, err := ffi.NewFFIClient().Sign(types.AlgorithmDilithium5, sk, pqcSignBytes)
			if err != nil {
				return fmt.Errorf("dilithium sign: %w", err)
			}

			// 6. Build the wire body WITH the PQC hybrid extension (proto-encoded
			//    so the chain's tx decoder can resolve the registered type URL).
			ext := types.PQCHybridSignature{
				AlgorithmID:  types.AlgorithmDilithium5,
				PQCSignature: pqcSig,
			}
			extVal, err := ext.Marshal()
			if err != nil {
				return err
			}
			bodyWithExt := &txtypes.TxBody{
				Messages:         rawTx.Body.Messages,
				Memo:             rawTx.Body.Memo,
				TimeoutHeight:    rawTx.Body.TimeoutHeight,
				ExtensionOptions: []*codectypes.Any{{TypeUrl: types.HybridSigTypeURL, Value: extVal}},
			}
			bodyWithExtBytes, err := bodyWithExt.Marshal()
			if err != nil {
				return err
			}

			// 7. Classical secp256k1 signature over SignDoc{bodyWithExt, authInfo}.
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

			// 8. Assemble + broadcast TxRaw.
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
		},
	}
	cmd.Flags().String("pqc-key", "", "name of the Dilithium key (created with `tx pqc gen-key`)")
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

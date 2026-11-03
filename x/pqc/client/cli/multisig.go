//go:build full

package cli

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// The k-of-n post-quantum multisig workflow (v3.2.7, pqc-multisig). Every
// step except the last works offline, and every file passed between members
// holds public data only (public keys and signatures).
//
// Once, to register the member set:
//
//	each PQC member:  tx pqc multisig-member-proof <multisig-addr> --pqc-key <name> --chain-id <id> > proof-i.json
//	anyone:           tx pqc multisig-register <k> proof-1.json ... --from <multisig> --generate-only > reg.json
//	classical members sign reg.json as any multisig transaction (`tx sign --multisig`, `tx multisign`):
//	it carries no extension, so the usual amino-json flow works, and the chain admits it on the
//	classical multisig signature alone (it is the bootstrap).
//
// For every later transaction of the multisig:
//
//	anyone:           tx bank send <multisig> ... --generate-only --gas <n> > tx.json
//	each PQC member:  tx pqc multisig-sign tx.json --multisig <addr> --pqc-key <name> > pqc-i.json
//	anyone:           tx pqc multisig-combine tx.json pqc-1.json pqc-2.json --multisig <addr> > combined.json
//	each classical member (the same agreed signer set):
//	                  tx pqc multisig-classical-sign combined.json --multisig <key> --signers 0,1 --from <member> > cl-i.json
//	anyone:           tx pqc multisig-assemble combined.json cl-1.json cl-2.json --multisig <key> --signers 0,1
//
// The classical half signs SIGN_MODE_DIRECT, not amino-json: amino-json
// refuses a transaction that carries extension options, and the
// post-quantum signatures travel in one. In direct mode every classical
// member signs the final auth info, which names who signs, so the classical
// signer set is agreed before signing (--signers). The post-quantum members
// sign bytes that name neither set, so they sign first and independently.

const (
	flagMultisig       = "multisig"
	flagMembersFile    = "members-file"
	flagClassicalSet   = "signers"
	flagMultisigPQCKey = "pqc-key"
	flagPubKey         = "pubkey"
)

func multisigCommands() []*cobra.Command {
	return []*cobra.Command{
		getCmdMultisigMemberProof(), getCmdMultisigRegister(), getCmdMultisigSign(), getCmdMultisigCombine(),
		getCmdMultisigClassicalSign(), getCmdMultisigAssemble(),
	}
}

// MemberProofFile is what multisig-member-proof prints.
type MemberProofFile struct {
	Multisig          string `json:"multisig"`
	ChainID           string `json:"chain_id"`
	PublicKey         string `json:"public_key"`
	ProofOfPossession string `json:"proof_of_possession"`
}

// MemberSignatureFile is what multisig-sign prints.
type MemberSignatureFile struct {
	Multisig      string `json:"multisig"`
	ChainID       string `json:"chain_id"`
	AccountNumber uint64 `json:"account_number,string"`
	Sequence      uint64 `json:"sequence,string"`
	Signature     string `json:"signature"`
}

// ClassicalSignatureFile is what multisig-classical-sign prints.
type ClassicalSignatureFile struct {
	Multisig    string `json:"multisig"`
	SignerIndex int    `json:"signer_index"`
	Signature   string `json:"signature"`
}

func printJSON(cmd *cobra.Command, v any) error {
	bz, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(bz))
	return err
}

func readJSONFile(path string, v any) error {
	bz, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bz, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func decodeHexField(name, s string) ([]byte, error) {
	bz, err := hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("%s is not hex: %w", name, err)
	}
	return bz, nil
}

// readUnsignedTx reads a transaction JSON file (as --generate-only prints).
func readUnsignedTx(clientCtx client.Context, path string) (*txtypes.Tx, error) {
	bz, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t txtypes.Tx
	if err := clientCtx.Codec.UnmarshalJSON(bz, &t); err != nil {
		return nil, fmt.Errorf("decode transaction %s: %w", path, err)
	}
	if t.Body == nil || t.AuthInfo == nil {
		return nil, fmt.Errorf("transaction %s is missing its body or auth_info", path)
	}
	return &t, nil
}

func multisigAddrFlag(cmd *cobra.Command) (sdk.AccAddress, error) {
	s, _ := cmd.Flags().GetString(flagMultisig)
	if s == "" {
		return nil, fmt.Errorf("--%s is required", flagMultisig)
	}
	return sdk.AccAddressFromBech32(s)
}

// multisigSignBytesFor computes the bytes every PQC member signs for tx.
func multisigSignBytesFor(chainID string, accNum, seq uint64, addr sdk.AccAddress, t *txtypes.Tx) ([]byte, error) {
	b0, err := types.MultisigBodyB0(t.Body)
	if err != nil {
		return nil, err
	}
	a0, err := types.MultisigAuthInfoA0(t.AuthInfo)
	if err != nil {
		return nil, err
	}
	return types.MultisigSignBytes(chainID, accNum, seq, addr, b0, a0), nil
}

func getCmdMultisigMemberProof() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-member-proof [multisig-address]",
		Short: "Prove, as a post-quantum member, that you hold your key and consent to the multisig (offline)",
		Long: `Sign the member proof of possession with a local ML-DSA-87 key (created with
` + "`tx pqc gen-key`" + `) and print it with the public key, for ` + "`tx pqc multisig-register`" + `.
The proof names the chain and the multisig account, so it is valid for nothing else.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			if clientCtx.ChainID == "" {
				return fmt.Errorf("--%s is required", flags.FlagChainID)
			}
			addr, err := sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString(flagMultisigPQCKey)
			sk, err := loadPQCPrivKey(clientCtx.HomeDir, name)
			if err != nil {
				return err
			}
			pubHex, _ := cmd.Flags().GetString(flagPubKey)
			pub, err := decodeHexField("--"+flagPubKey, pubHex)
			if err != nil || len(pub) == 0 {
				return fmt.Errorf("--%s is required: the member's public key in hex, as gen-key printed it", flagPubKey)
			}
			bz := types.MultisigMemberPossessionSignBytes(clientCtx.ChainID, uint32(types.AlgorithmDilithium5), addr.String(), pub)
			sig, err := qpqc.MLDSA87.Sign(sk, bz)
			if err != nil {
				return fmt.Errorf("sign proof: %w", err)
			}
			if !qpqc.MLDSA87.Verify(pub, bz, sig) {
				return fmt.Errorf("the key %q does not match --%s", name, flagPubKey)
			}
			return printJSON(cmd, MemberProofFile{Multisig: addr.String(), ChainID: clientCtx.ChainID,
				PublicKey: hex.EncodeToString(pub), ProofOfPossession: hex.EncodeToString(sig)})
		},
	}
	cmd.Flags().String(flagMultisigPQCKey, "", "name of the member's Dilithium key (created with `tx pqc gen-key`)")
	cmd.Flags().String(flagPubKey, "", "the member's ML-DSA-87 public key in hex (gen-key prints it)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getCmdMultisigRegister() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-register [threshold] [member-proof.json]...",
		Short: "Register (or replace) the post-quantum member set of a multisig account",
		Long: `Build MsgRegisterPQCMultisig from the members' proof files, in member order:
the order of the files is the member order of the bitmap in every later
transaction. Use --from <multisig> --generate-only and have the classical
members sign it: the first registration is admitted on the classical multisig
signature alone; a replacement must carry k-of-n signatures of the CURRENT set
(multisig-sign / multisig-combine / multisig-classical-sign / multisig-assemble).`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			k, err := strconv.ParseUint(args[0], 10, 32)
			if err != nil {
				return fmt.Errorf("threshold: %w", err)
			}
			sender := clientCtx.GetFromAddress().String()
			msg := &types.MsgRegisterPQCMultisig{Sender: sender, AlgorithmID: types.AlgorithmDilithium5, Threshold: uint32(k)}
			for _, path := range args[1:] {
				var p MemberProofFile
				if err := readJSONFile(path, &p); err != nil {
					return err
				}
				if p.Multisig != sender {
					return fmt.Errorf("%s is a proof for %s, not for %s", path, p.Multisig, sender)
				}
				if clientCtx.ChainID != "" && p.ChainID != clientCtx.ChainID {
					return fmt.Errorf("%s is a proof for chain %s, not %s", path, p.ChainID, clientCtx.ChainID)
				}
				pub, err := decodeHexField("public_key", p.PublicKey)
				if err != nil {
					return err
				}
				proof, err := decodeHexField("proof_of_possession", p.ProofOfPossession)
				if err != nil {
					return err
				}
				if !qpqc.MLDSA87.Verify(pub, types.MultisigMemberPossessionSignBytes(p.ChainID,
					uint32(types.AlgorithmDilithium5), sender, pub), proof) {
					return fmt.Errorf("%s: the proof does not verify", path)
				}
				msg.PublicKeys = append(msg.PublicKeys, pub)
				msg.ProofsOfPossession = append(msg.ProofsOfPossession, proof)
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

func getCmdMultisigSign() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-sign [unsigned-tx.json]",
		Short: "Sign a multisig transaction as one post-quantum member (offline with --offline)",
		Long: `Sign the multisig sign-bytes of a --generate-only transaction with a local
ML-DSA-87 key and print the signature file for ` + "`tx pqc multisig-combine`" + `.
The bytes bind the chain id, the multisig's account number and sequence, the
body and the fee, so the transaction must be final (gas limit included) before
anyone signs. With --offline, pass --account-number and --sequence.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			if clientCtx.ChainID == "" {
				return fmt.Errorf("--%s is required", flags.FlagChainID)
			}
			addr, err := multisigAddrFlag(cmd)
			if err != nil {
				return err
			}
			t, err := readUnsignedTx(clientCtx, args[0])
			if err != nil {
				return err
			}
			if len(t.Body.ExtensionOptions) > 0 {
				return fmt.Errorf("sign the transaction as generated, before any extension is added")
			}
			accNum, seq, err := signerAccountNumberSequence(cmd, clientCtx, addr)
			if err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString(flagMultisigPQCKey)
			sk, err := loadPQCPrivKey(clientCtx.HomeDir, name)
			if err != nil {
				return err
			}
			bz, err := multisigSignBytesFor(clientCtx.ChainID, accNum, seq, addr, t)
			if err != nil {
				return err
			}
			sig, err := qpqc.MLDSA87.Sign(sk, bz)
			if err != nil {
				return fmt.Errorf("sign: %w", err)
			}
			return printJSON(cmd, MemberSignatureFile{Multisig: addr.String(), ChainID: clientCtx.ChainID,
				AccountNumber: accNum, Sequence: seq, Signature: hex.EncodeToString(sig)})
		},
	}
	cmd.Flags().String(flagMultisig, "", "bech32 address of the multisig account")
	cmd.Flags().String(flagMultisigPQCKey, "", "name of the member's Dilithium key (created with `tx pqc gen-key`)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// registeredMembers returns the member keys in member order: from
// --members-file (a JSON array of hex keys, offline) or from the chain.
func registeredMembers(cmd *cobra.Command, clientCtx client.Context, addr sdk.AccAddress) ([][]byte, uint32, error) {
	if path, _ := cmd.Flags().GetString(flagMembersFile); path != "" {
		var hexKeys []string
		if err := readJSONFile(path, &hexKeys); err != nil {
			return nil, 0, err
		}
		var keys [][]byte
		for i, h := range hexKeys {
			k, err := decodeHexField(fmt.Sprintf("member %d", i), h)
			if err != nil {
				return nil, 0, err
			}
			keys = append(keys, k)
		}
		return keys, 0, nil
	}
	if clientCtx.Offline {
		return nil, 0, fmt.Errorf("--%s reads the member set from the chain, which --%s forbids; pass --%s",
			flagMultisig, flags.FlagOffline, flagMembersFile)
	}
	res, err := types.NewMultisigQueryClient(clientCtx).Multisig(cmd.Context(), &types.QueryMultisigRequest{Address: addr.String()})
	if err != nil {
		return nil, 0, err
	}
	if !res.Found {
		return nil, 0, fmt.Errorf("%s has no registered post-quantum member set", addr)
	}
	return res.Multisig.PublicKeys, res.Multisig.Threshold, nil
}

// CombineMultisigSignatures checks each member signature against the member
// set and the transaction and returns the extension holding them in member
// order. Exported for tests and other clients.
func CombineMultisigSignatures(chainID string, addr sdk.AccAddress, t *txtypes.Tx, members [][]byte, threshold uint32, files []MemberSignatureFile) (*codectypes.Any, error) {
	bySlot := map[int][]byte{}
	for _, f := range files {
		if f.Multisig != addr.String() || f.ChainID != chainID {
			return nil, fmt.Errorf("a signature for %s on %s, not for %s on %s", f.Multisig, f.ChainID, addr, chainID)
		}
		sig, err := decodeHexField("signature", f.Signature)
		if err != nil {
			return nil, err
		}
		bz, err := multisigSignBytesFor(chainID, f.AccountNumber, f.Sequence, addr, t)
		if err != nil {
			return nil, err
		}
		// The file does not say who signed: the member whose key verifies it.
		slot := -1
		for i, k := range members {
			if qpqc.MLDSA87.Verify(k, bz, sig) {
				slot = i
				break
			}
		}
		if slot < 0 {
			return nil, fmt.Errorf("a signature that no member key verifies for this transaction (account %d, sequence %d)",
				f.AccountNumber, f.Sequence)
		}
		if _, dup := bySlot[slot]; dup {
			return nil, fmt.Errorf("member %d signed twice", slot)
		}
		bySlot[slot] = sig
	}
	if threshold > 0 && len(bySlot) < int(threshold) {
		return nil, fmt.Errorf("%d member signatures, the threshold is %d", len(bySlot), threshold)
	}
	var slots []int
	for i := range members {
		if _, ok := bySlot[i]; ok {
			slots = append(slots, i)
		}
	}
	bm, err := types.MultisigBitmap(len(members), slots)
	if err != nil {
		return nil, err
	}
	ext := types.PQCMultisigSignature{AlgorithmID: types.AlgorithmDilithium5, Bitmap: bm}
	for _, s := range slots {
		ext.Signatures = append(ext.Signatures, bySlot[s])
	}
	if err := ext.ValidateStateless(); err != nil {
		return nil, err
	}
	val, err := ext.Marshal()
	if err != nil {
		return nil, err
	}
	return &codectypes.Any{TypeUrl: types.MultisigSigTypeURL, Value: val}, nil
}

func getCmdMultisigCombine() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-combine [unsigned-tx.json] [member-signature.json]...",
		Short: "Combine post-quantum member signatures into the transaction's multisig extension",
		Long: `Check every member signature against the member set (from the chain, or
--members-file offline) and print the transaction with the PQCMultisigSignature
extension added, ready for the classical members (multisig-classical-sign).`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			addr, err := multisigAddrFlag(cmd)
			if err != nil {
				return err
			}
			t, err := readUnsignedTx(clientCtx, args[0])
			if err != nil {
				return err
			}
			if len(t.Body.ExtensionOptions) > 0 {
				return fmt.Errorf("the transaction already carries an extension option")
			}
			members, threshold, err := registeredMembers(cmd, clientCtx, addr)
			if err != nil {
				return err
			}
			var files []MemberSignatureFile
			chainID := clientCtx.ChainID
			for _, path := range args[1:] {
				var f MemberSignatureFile
				if err := readJSONFile(path, &f); err != nil {
					return err
				}
				if chainID == "" {
					chainID = f.ChainID
				}
				files = append(files, f)
			}
			ext, err := CombineMultisigSignatures(chainID, addr, t, members, threshold, files)
			if err != nil {
				return err
			}
			t.Body.ExtensionOptions = []*codectypes.Any{ext}
			out, err := clientCtx.Codec.MarshalJSON(t)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return err
		},
	}
	cmd.Flags().String(flagMultisig, "", "bech32 address of the multisig account")
	cmd.Flags().String(flagMembersFile, "", "JSON array of the member public keys in hex, in member order (offline)")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// parseSignerSet reads --signers, the agreed classical signer indices.
func parseSignerSet(cmd *cobra.Command, n int) ([]int, error) {
	s, _ := cmd.Flags().GetString(flagClassicalSet)
	if s == "" {
		return nil, fmt.Errorf("--%s is required: the classical member indices that will sign, e.g. 0,2", flagClassicalSet)
	}
	seen := map[int]bool{}
	var out []int
	for _, part := range strings.Split(s, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || i < 0 || i >= n || seen[i] {
			return nil, fmt.Errorf("--%s: bad or repeated index %q for a multisig of %d keys", flagClassicalSet, part, n)
		}
		seen[i] = true
		out = append(out, i)
	}
	// Ascending, the order of the bitarray and of the signatures.
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

// multisigPubKey reads the classical multisig key from the keyring (a record
// made with `keys add --multisig`).
func multisigPubKey(clientCtx client.Context, cmd *cobra.Command) (*kmultisig.LegacyAminoPubKey, error) {
	name, _ := cmd.Flags().GetString(flagMultisig)
	if name == "" {
		return nil, fmt.Errorf("--%s is required: the keyring name of the multisig key", flagMultisig)
	}
	rec, err := clientCtx.Keyring.Key(name)
	if err != nil {
		return nil, fmt.Errorf("multisig key %q: %w", name, err)
	}
	pk, err := rec.GetPubKey()
	if err != nil {
		return nil, err
	}
	ms, ok := pk.(*kmultisig.LegacyAminoPubKey)
	if !ok {
		return nil, fmt.Errorf("key %q is not a multisig key", name)
	}
	return ms, nil
}

// MultisigDirectAuthInfo is the final auth info of a multisig transaction
// signed in SIGN_MODE_DIRECT by the given classical members. Every classical
// member signs it, so every member must build exactly this.
func MultisigDirectAuthInfo(pub *kmultisig.LegacyAminoPubKey, signers []int, seq uint64, base *txtypes.AuthInfo) (*txtypes.AuthInfo, error) {
	pubAny, err := codectypes.NewAnyWithValue(pub)
	if err != nil {
		return nil, err
	}
	bits := cryptotypes.NewCompactBitArray(len(pub.GetPubKeys()))
	modes := make([]*txtypes.ModeInfo, 0, len(signers))
	for _, i := range signers {
		bits.SetIndex(i, true)
		modes = append(modes, &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Single_{
			Single: &txtypes.ModeInfo_Single{Mode: signing.SignMode_SIGN_MODE_DIRECT}}})
	}
	return &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{{
			PublicKey: pubAny,
			ModeInfo:  &txtypes.ModeInfo{Sum: &txtypes.ModeInfo_Multi_{Multi: &txtypes.ModeInfo_Multi{Bitarray: bits, ModeInfos: modes}}},
			Sequence:  seq,
		}},
		Fee: base.Fee,
		Tip: base.Tip, //nolint:staticcheck // carried over unchanged; the PQC members signed it
	}, nil
}

// classicalSignDoc is the SIGN_MODE_DIRECT sign doc of the combined
// transaction with its final auth info.
func classicalSignDoc(cmd *cobra.Command, clientCtx client.Context, t *txtypes.Tx, pub *kmultisig.LegacyAminoPubKey, signers []int) ([]byte, []byte, []byte, error) {
	addr := sdk.AccAddress(pub.Address())
	accNum, seq, err := signerAccountNumberSequence(cmd, clientCtx, addr)
	if err != nil {
		return nil, nil, nil, err
	}
	auth, err := MultisigDirectAuthInfo(pub, signers, seq, t.AuthInfo)
	if err != nil {
		return nil, nil, nil, err
	}
	authBytes, err := auth.Marshal()
	if err != nil {
		return nil, nil, nil, err
	}
	bodyBytes, err := t.Body.Marshal()
	if err != nil {
		return nil, nil, nil, err
	}
	doc, err := (&txtypes.SignDoc{BodyBytes: bodyBytes, AuthInfoBytes: authBytes, ChainId: clientCtx.ChainID, AccountNumber: accNum}).Marshal()
	if err != nil {
		return nil, nil, nil, err
	}
	return doc, bodyBytes, authBytes, nil
}

func getCmdMultisigClassicalSign() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-classical-sign [combined-tx.json]",
		Short: "Sign a combined multisig transaction as one classical member (SIGN_MODE_DIRECT)",
		Long: `Sign the combined transaction (multisig-combine output) with the --from
member's classical key, in SIGN_MODE_DIRECT over the final auth info. Every
classical member passes the same --signers set (ascending indices into the
multisig key, the members who will sign); --from must be one of them.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pub, err := multisigPubKey(clientCtx, cmd)
			if err != nil {
				return err
			}
			signers, err := parseSignerSet(cmd, len(pub.GetPubKeys()))
			if err != nil {
				return err
			}
			t, err := readUnsignedTx(clientCtx, args[0])
			if err != nil {
				return err
			}
			rec, err := clientCtx.Keyring.Key(clientCtx.FromName)
			if err != nil {
				return err
			}
			mine, err := rec.GetPubKey()
			if err != nil {
				return err
			}
			slot := -1
			for i, k := range pub.GetPubKeys() {
				if k.Equals(mine) {
					slot = i
				}
			}
			inSet := false
			for _, i := range signers {
				inSet = inSet || i == slot
			}
			if slot < 0 || !inSet {
				return fmt.Errorf("--from %s is not one of the --%s members of the multisig", clientCtx.FromName, flagClassicalSet)
			}
			doc, _, _, err := classicalSignDoc(cmd, clientCtx, t, pub, signers)
			if err != nil {
				return err
			}
			sig, _, err := clientCtx.Keyring.Sign(rec.Name, doc, signing.SignMode_SIGN_MODE_DIRECT)
			if err != nil {
				return err
			}
			return printJSON(cmd, ClassicalSignatureFile{Multisig: sdk.AccAddress(pub.Address()).String(),
				SignerIndex: slot, Signature: hex.EncodeToString(sig)})
		},
	}
	cmd.Flags().String(flagMultisig, "", "keyring name of the multisig key (`keys add --multisig`)")
	cmd.Flags().String(flagClassicalSet, "", "comma-separated classical member indices that sign, e.g. 0,1")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func getCmdMultisigAssemble() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multisig-assemble [combined-tx.json] [classical-signature.json]...",
		Short: "Assemble the classical member signatures and broadcast (or print) the multisig transaction",
		Long: `Put the classical members' signatures (multisig-classical-sign output) into
the multisig signature and broadcast the transaction. --generate-only or
--offline print it instead; --dry-run simulates it.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			pub, err := multisigPubKey(clientCtx, cmd)
			if err != nil {
				return err
			}
			signers, err := parseSignerSet(cmd, len(pub.GetPubKeys()))
			if err != nil {
				return err
			}
			if len(args)-1 != len(signers) {
				return fmt.Errorf("%d signature files for %d --%s members", len(args)-1, len(signers), flagClassicalSet)
			}
			t, err := readUnsignedTx(clientCtx, args[0])
			if err != nil {
				return err
			}
			doc, bodyBytes, authBytes, err := classicalSignDoc(cmd, clientCtx, t, pub, signers)
			if err != nil {
				return err
			}
			bySlot := map[int][]byte{}
			for _, path := range args[1:] {
				var f ClassicalSignatureFile
				if err := readJSONFile(path, &f); err != nil {
					return err
				}
				sig, err := decodeHexField("signature", f.Signature)
				if err != nil {
					return err
				}
				if f.SignerIndex < 0 || f.SignerIndex >= len(pub.GetPubKeys()) {
					return fmt.Errorf("%s: signer index %d is outside the multisig", path, f.SignerIndex)
				}
				if !pub.GetPubKeys()[f.SignerIndex].VerifySignature(doc, sig) {
					return fmt.Errorf("%s: member %d's signature does not verify for this transaction", path, f.SignerIndex)
				}
				bySlot[f.SignerIndex] = sig
			}
			var sigs [][]byte
			for _, i := range signers {
				s, ok := bySlot[i]
				if !ok {
					return fmt.Errorf("no signature from member %d", i)
				}
				sigs = append(sigs, s)
			}
			multi, err := (&cryptotypes.MultiSignature{Signatures: sigs}).Marshal()
			if err != nil {
				return err
			}
			raw, err := (&txtypes.TxRaw{BodyBytes: bodyBytes, AuthInfoBytes: authBytes, Signatures: [][]byte{multi}}).Marshal()
			if err != nil {
				return err
			}
			return deliverSignedTx(cmd, clientCtx, raw)
		},
	}
	cmd.Flags().String(flagMultisig, "", "keyring name of the multisig key (`keys add --multisig`)")
	cmd.Flags().String(flagClassicalSet, "", "comma-separated classical member indices that sign, e.g. 0,1")
	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

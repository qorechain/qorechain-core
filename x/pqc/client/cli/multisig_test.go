//go:build full

package cli

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	kmultisig "github.com/cosmos/cosmos-sdk/crypto/keys/multisig"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

type msCLI struct {
	*cliFixture
	ms       *kmultisig.LegacyAminoPubKey
	msAddr   sdk.AccAddress
	pqcPubs  [][]byte
	unsigned string
}

// newMSCLI adds to the CLI fixture a 2-of-3 classical multisig key "ms" over
// keyring keys m0..m2, three ML-DSA-87 member keys pq0..pq2 and an unsigned
// send from the multisig.
func newMSCLI(t *testing.T) *msCLI {
	t.Helper()
	fx := newCLIFixture(t)
	f := &msCLI{cliFixture: fx}
	var pubs []cryptotypes.PubKey
	for _, n := range []string{"m0", "m1", "m2"} {
		rec, _, err := fx.ctx.Keyring.NewMnemonic(n, keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
		if err != nil {
			t.Fatal(err)
		}
		pk, _ := rec.GetPubKey()
		pubs = append(pubs, pk)

		pub, sk, err := qpqc.MLDSA87.Keygen()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pqcKeyPath(fx.home, "pq"+n[1:]), []byte(hex.EncodeToString(sk)), 0o600); err != nil {
			t.Fatal(err)
		}
		f.pqcPubs = append(f.pqcPubs, pub)
	}
	f.ms = kmultisig.NewLegacyAminoPubKey(2, pubs)
	if _, err := fx.ctx.Keyring.SaveMultisig("ms", f.ms); err != nil {
		t.Fatal(err)
	}
	f.msAddr = sdk.AccAddress(f.ms.Address())

	b := fx.ctx.TxConfig.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(f.msAddr, fx.from, sdk.NewCoins(sdk.NewInt64Coin("uqor", 5)))); err != nil {
		t.Fatal(err)
	}
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uqor", 2000)))
	b.SetGasLimit(400000)
	b.SetMemo("from the multisig")
	js, err := fx.ctx.TxConfig.TxJSONEncoder()(b.GetTx())
	if err != nil {
		t.Fatal(err)
	}
	f.unsigned = f.write(t, "unsigned.json", js)
	return f
}

func (f *msCLI) write(t *testing.T, name string, bz []byte) string {
	t.Helper()
	p := filepath.Join(f.home, name)
	if err := os.WriteFile(p, bz, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

var offline = []string{"--chain-id", testChainID, "--offline", "--account-number", "7", "--sequence", "3"}

func withOffline(args ...string) []string { return append(args, offline...) }

// The whole client workflow, offline: member proofs, registration, two of
// three post-quantum signatures, combination, two classical signatures in
// direct mode, assembly. The printed transaction carries what the chain
// verifies.
func TestMultisigCLIWorkflow(t *testing.T) {
	f := newMSCLI(t)

	// Registration.
	var proofs []string
	for i := 0; i < 3; i++ {
		out, err := f.run(getCmdMultisigMemberProof(), "", f.msAddr.String(), "--pqc-key", "pq"+string(rune('0'+i)),
			"--pubkey", hex.EncodeToString(f.pqcPubs[i]), "--chain-id", testChainID)
		if err != nil {
			t.Fatal(err)
		}
		proofs = append(proofs, f.write(t, "proof"+string(rune('0'+i))+".json", []byte(out)))
	}
	// A key that is not the one named is caught before anything is printed.
	if _, err := f.run(getCmdMultisigMemberProof(), "", f.msAddr.String(), "--pqc-key", "pq0",
		"--pubkey", hex.EncodeToString(f.pqcPubs[1]), "--chain-id", testChainID); err == nil {
		t.Fatal("a proof for the wrong public key was printed")
	}
	hexKeys := []string{}
	for _, k := range f.pqcPubs {
		hexKeys = append(hexKeys, hex.EncodeToString(k))
	}
	membersJSON, _ := json.Marshal(hexKeys)
	members := f.write(t, "members.json", membersJSON)
	regArgs := []string{"--multisig", "ms", "--members-file", members, "--from", f.msAddr.String(),
		"--generate-only", "--offline", "--account-number", "7", "--sequence", "0"}
	// Every member proven: no proof file needed; member 1's proof may still
	// be attached.
	if _, err := f.run(getCmdMultisigRegister(), "", append([]string{proofs[1]}, regArgs...)...); err != nil {
		t.Fatal(err)
	}
	regTx, err := f.ctx.TxConfig.TxJSONDecoder()(bytes.TrimSpace(f.out.Bytes()))
	if err != nil {
		t.Fatalf("%v\n%s", err, f.out.String())
	}
	reg, ok := regTx.GetMsgs()[0].(*types.MsgRegisterPQCMultisig)
	if !ok || reg.Threshold != 2 || len(reg.PublicKeys) != 3 || !bytes.Equal(reg.PublicKeys[2], f.pqcPubs[2]) ||
		len(reg.ProofsOfPossession[0]) != 0 || len(reg.ProofsOfPossession[1]) == 0 {
		t.Fatalf("registration %+v", regTx.GetMsgs())
	}
	if err := reg.ValidateBasic(); err != nil {
		t.Fatal(err)
	}
	// The threshold and the member order come from the multisig key; the
	// members file must list one key per classical member.
	short, _ := json.Marshal(hexKeys[:2])
	if _, err := f.run(getCmdMultisigRegister(), "", "--multisig", "ms", "--members-file", f.write(t, "short.json", short),
		"--from", f.msAddr.String(), "--generate-only", "--offline", "--account-number", "7", "--sequence", "0"); err == nil {
		t.Fatal("a members file with fewer keys than classical members was accepted")
	}
	// Sent from anyone but the multisig: refused.
	if _, err := f.run(getCmdMultisigRegister(), "", "--multisig", "ms", "--members-file", members,
		"--from", f.from.String(), "--generate-only", "--offline", "--account-number", "7", "--sequence", "0"); err == nil {
		t.Fatal("a registration from another account was built")
	}

	// Post-quantum members 0 and 2 sign.
	var pqcFiles []string
	for _, i := range []int{2, 0} { // order of the files does not matter
		out, err := f.run(getCmdMultisigSign(), "", withOffline(f.unsigned, "--multisig", f.msAddr.String(),
			"--pqc-key", "pq"+string(rune('0'+i)))...)
		if err != nil {
			t.Fatal(err)
		}
		pqcFiles = append(pqcFiles, f.write(t, "pqc"+string(rune('0'+i))+".json", []byte(out)))
	}
	out, err := f.run(getCmdMultisigCombine(), "", append(append([]string{f.unsigned}, pqcFiles...),
		"--multisig", f.msAddr.String(), "--members-file", members, "--chain-id", testChainID, "--offline")...)
	if err != nil {
		t.Fatal(err)
	}
	combined := f.write(t, "combined.json", []byte(out))

	// A signature combined twice is refused.
	if _, err := f.run(getCmdMultisigCombine(), "", f.unsigned, pqcFiles[0], pqcFiles[0],
		"--multisig", f.msAddr.String(), "--members-file", members, "--chain-id", testChainID, "--offline"); err == nil {
		t.Fatal("the same member combined twice")
	}

	// Classical members 0 and 1 sign in direct mode.
	var clFiles []string
	for _, n := range []string{"m1", "m0"} {
		out, err := f.run(getCmdMultisigClassicalSign(), "", withOffline(combined, "--multisig", "ms", "--signers", "1,0", "--from", n)...)
		if err != nil {
			t.Fatal(err)
		}
		clFiles = append(clFiles, f.write(t, "cl-"+n+".json", []byte(out)))
	}
	// m2 is not in the agreed set.
	if _, err := f.run(getCmdMultisigClassicalSign(), "", withOffline(combined, "--multisig", "ms", "--signers", "0,1", "--from", "m2")...); err == nil {
		t.Fatal("a member outside --signers signed")
	}

	if _, err := f.run(getCmdMultisigAssemble(), "", append(append([]string{combined}, clFiles...),
		withOffline("--multisig", "ms", "--signers", "0,1")...)...); err != nil {
		t.Fatal(err)
	}
	raw := f.decodePrinted(t)
	f.verifyMultisigTx(t, raw, []int{0, 2}, []int{0, 1})
}

// verifyMultisigTx checks a printed transaction the way the chain does: the
// post-quantum members over the multisig sign-bytes, the classical members
// over the direct sign doc of the final auth info.
func (f *msCLI) verifyMultisigTx(t *testing.T, raw *txtypes.TxRaw, pqcMembers, classical []int) {
	t.Helper()
	var body txtypes.TxBody
	if err := body.Unmarshal(raw.BodyBytes); err != nil {
		t.Fatal(err)
	}
	var auth txtypes.AuthInfo
	if err := auth.Unmarshal(raw.AuthInfoBytes); err != nil {
		t.Fatal(err)
	}
	if body.Memo != "from the multisig" || len(body.ExtensionOptions) != 1 || body.ExtensionOptions[0].TypeUrl != types.MultisigSigTypeURL {
		t.Fatalf("body %+v", body)
	}
	var ext types.PQCMultisigSignature
	if err := ext.Unmarshal(body.ExtensionOptions[0].Value); err != nil {
		t.Fatal(err)
	}
	if err := ext.ValidateStateless(); err != nil {
		t.Fatal(err)
	}
	got, err := ext.Signers(3)
	if err != nil || len(got) != len(pqcMembers) {
		t.Fatalf("signers %v %v", got, err)
	}
	b0, _ := types.MultisigBodyB0(&body)
	a0, _ := types.MultisigAuthInfoA0(&auth)
	bz := types.MultisigSignBytes(testChainID, 7, 3, f.msAddr, b0, a0)
	for j, i := range got {
		if i != pqcMembers[j] || !qpqc.MLDSA87.Verify(f.pqcPubs[i], bz, ext.Signatures[j]) {
			t.Fatalf("post-quantum member %d does not verify", i)
		}
	}

	if len(auth.SignerInfos) != 1 || auth.SignerInfos[0].Sequence != 3 {
		t.Fatalf("signer infos %+v", auth.SignerInfos)
	}
	multi := auth.SignerInfos[0].ModeInfo.GetMulti()
	if multi == nil || multi.Bitarray.Count() != 3 || len(multi.ModeInfos) != len(classical) {
		t.Fatalf("mode info %+v", auth.SignerInfos[0].ModeInfo)
	}
	var ms cryptotypes.MultiSignature
	if err := ms.Unmarshal(raw.Signatures[0]); err != nil {
		t.Fatal(err)
	}
	doc, _ := (&txtypes.SignDoc{BodyBytes: raw.BodyBytes, AuthInfoBytes: raw.AuthInfoBytes, ChainId: testChainID, AccountNumber: 7}).Marshal()
	for j, i := range classical {
		if !multi.Bitarray.GetIndex(i) || !f.ms.GetPubKeys()[i].VerifySignature(doc, ms.Signatures[j]) {
			t.Fatalf("classical member %d does not verify", i)
		}
	}
}

// Below the threshold the combiner refuses when it knows the threshold (from
// the chain); offline it builds the extension and the chain refuses it.
func TestCombineMultisigSignaturesThreshold(t *testing.T) {
	f := newMSCLI(t)
	tx, err := readUnsignedTx(f.ctx, f.unsigned)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.run(getCmdMultisigSign(), "", withOffline(f.unsigned, "--multisig", f.msAddr.String(), "--pqc-key", "pq1")...)
	if err != nil {
		t.Fatal(err)
	}
	var file MemberSignatureFile
	if err := json.Unmarshal([]byte(out), &file); err != nil {
		t.Fatal(err)
	}
	if _, err := CombineMultisigSignatures(testChainID, f.msAddr, tx, f.pqcPubs, 2, []MemberSignatureFile{file}); err == nil {
		t.Fatal("1 of a 2-of-3 combined")
	}
	if _, err := CombineMultisigSignatures("qorechain-other", f.msAddr, tx, f.pqcPubs, 2, []MemberSignatureFile{file}); err == nil {
		t.Fatal("a signature for another chain combined")
	}
	file.Sequence = 4
	if _, err := CombineMultisigSignatures(testChainID, f.msAddr, tx, f.pqcPubs, 0, []MemberSignatureFile{file}); err == nil {
		t.Fatal("a signature for another sequence combined")
	}
}

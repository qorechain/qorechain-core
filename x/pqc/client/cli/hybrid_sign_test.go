//go:build full

package cli

import (
	"bytes"
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

const testChainID = "qorechain-test"

type cliFixture struct {
	ctx     client.Context
	out     *bytes.Buffer
	from    sdk.AccAddress
	pqcPub  []byte
	home    string
	txFile  string
	keyName string
}

func newCLIFixture(t *testing.T) *cliFixture {
	t.Helper()
	reg := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(reg)
	banktypes.RegisterInterfaces(reg)
	types.RegisterInterfaces(reg)
	cdc := codec.NewProtoCodec(reg)
	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	kr := keyring.NewInMemory(cdc)
	rec, _, err := kr.NewMnemonic("alice", keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	if err != nil {
		t.Fatal(err)
	}
	from, _ := rec.GetAddress()

	home := t.TempDir()
	pk, sk, err := qpqc.MLDSA87.Keygen()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(pqcKeyDir(home), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pqcKeyPath(home, "pq"), []byte(hex.EncodeToString(sk)), 0o600); err != nil {
		t.Fatal(err)
	}

	// An unsigned generate-only tx, as `tx bank send --generate-only` writes it.
	b := txCfg.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(from, from, sdk.NewCoins(sdk.NewInt64Coin("uqor", 5)))); err != nil {
		t.Fatal(err)
	}
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uqor", 2000)))
	b.SetGasLimit(200000)
	b.SetMemo("hello")
	js, err := txCfg.TxJSONEncoder()(b.GetTx())
	if err != nil {
		t.Fatal(err)
	}
	txFile := filepath.Join(home, "tx.json")
	if err := os.WriteFile(txFile, js, 0o600); err != nil {
		t.Fatal(err)
	}

	out := &bytes.Buffer{}
	ctx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(reg).
		WithTxConfig(txCfg).
		WithKeyring(kr).
		WithHomeDir(home).
		WithOutput(out).
		WithAccountRetriever(failingRetriever{t})
	return &cliFixture{ctx: ctx, out: out, from: from, pqcPub: pk, home: home, txFile: txFile, keyName: "alice"}
}

// failingRetriever fails the test if a command that must stay offline asks the
// node for the account.
type failingRetriever struct{ t *testing.T }

func (f failingRetriever) GetAccount(client.Context, sdk.AccAddress) (client.Account, error) {
	f.t.Fatal("queried the node for the account")
	return nil, nil
}
func (f failingRetriever) GetAccountWithHeight(client.Context, sdk.AccAddress) (client.Account, int64, error) {
	f.t.Fatal("queried the node for the account")
	return nil, 0, nil
}
func (f failingRetriever) EnsureExists(client.Context, sdk.AccAddress) error { return nil }
func (f failingRetriever) GetAccountNumberSequence(client.Context, sdk.AccAddress) (uint64, uint64, error) {
	f.t.Fatal("queried the node for the account number")
	return 0, 0, nil
}

func (fx *cliFixture) run(cmd *cobra.Command, stdin string, args ...string) (string, error) {
	fx.out.Reset()
	c := context.WithValue(context.Background(), client.ClientContextKey, &fx.ctx)
	cmd.SetContext(c)
	cmd.SetIn(strings.NewReader(stdin))
	var errOut bytes.Buffer
	cmd.SetErr(&errOut)
	cmd.SetOut(&errOut)
	cmd.SetArgs(args)
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	err := cmd.Execute()
	return errOut.String(), err
}

// decodePrinted parses the JSON tx a command printed and returns its wire form.
func (fx *cliFixture) decodePrinted(t *testing.T) *txtypes.TxRaw {
	t.Helper()
	tx, err := fx.ctx.TxConfig.TxJSONDecoder()(bytes.TrimSpace(fx.out.Bytes()))
	if err != nil {
		t.Fatalf("printed output is not a tx: %v\n%s", err, fx.out.String())
	}
	bz, err := fx.ctx.TxConfig.TxEncoder()(tx)
	if err != nil {
		t.Fatal(err)
	}
	var raw txtypes.TxRaw
	if err := raw.Unmarshal(bz); err != nil {
		t.Fatal(err)
	}
	return &raw
}

// verifyHybrid checks both signatures of a printed tx: the classical one over
// the SignDoc with the given account number, and the ML-DSA-87 one over the v2
// hybrid sign-bytes. A tx that passes both is what the ante accepts.
func (fx *cliFixture) verifyHybrid(t *testing.T, raw *txtypes.TxRaw, pqcPub []byte, accNum, seq uint64) *txtypes.TxBody {
	t.Helper()
	var body txtypes.TxBody
	if err := body.Unmarshal(raw.BodyBytes); err != nil {
		t.Fatal(err)
	}
	var auth txtypes.AuthInfo
	if err := auth.Unmarshal(raw.AuthInfoBytes); err != nil {
		t.Fatal(err)
	}
	if len(auth.SignerInfos) != 1 || auth.SignerInfos[0].Sequence != seq {
		t.Fatalf("signer infos %+v, want one with sequence %d", auth.SignerInfos, seq)
	}
	rec, _ := fx.ctx.Keyring.Key(fx.keyName)
	pub, _ := rec.GetPubKey()
	doc := txtypes.SignDoc{BodyBytes: raw.BodyBytes, AuthInfoBytes: raw.AuthInfoBytes, ChainId: testChainID, AccountNumber: accNum}
	docBz, _ := doc.Marshal()
	if len(raw.Signatures) != 1 || !pub.VerifySignature(docBz, raw.Signatures[0]) {
		t.Fatal("classical signature does not verify")
	}
	if len(body.ExtensionOptions) != 1 || body.ExtensionOptions[0].TypeUrl != types.HybridSigTypeURL {
		t.Fatalf("missing hybrid extension: %+v", body.ExtensionOptions)
	}
	var ext types.PQCHybridSignature
	if err := ext.Unmarshal(body.ExtensionOptions[0].Value); err != nil {
		t.Fatal(err)
	}
	b0, _ := (&txtypes.TxBody{Messages: body.Messages, Memo: body.Memo, TimeoutHeight: body.TimeoutHeight}).Marshal()
	if !qpqc.MLDSA87.Verify(pqcPub, types.HybridSignBytes(testChainID, b0, raw.AuthInfoBytes), ext.PQCSignature) {
		t.Fatal("PQC signature does not verify")
	}
	return &body
}

// TestCosignOfflinePrints pins QSR-2026-0273: --offline with account number,
// sequence and an explicit sign-bytes form prints a fully signed tx and talks
// to no node.
func TestCosignOfflinePrints(t *testing.T) {
	fx := newCLIFixture(t)
	_, err := fx.run(getCmdCosign(), "", fx.txFile, "--from", fx.keyName, "--pqc-key", "pq",
		"--chain-id", testChainID, "--offline", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2")
	if err != nil {
		t.Fatal(err)
	}
	body := fx.verifyHybrid(t, fx.decodePrinted(t), fx.pqcPub, 5, 7)
	if body.Memo != "hello" || len(body.Messages) != 1 {
		t.Fatalf("body not carried over: %+v", body)
	}
}

// TestCosignGenerateOnlyPrints: --generate-only prints instead of
// broadcasting (the fixture has no node, so a broadcast would fail).
func TestCosignGenerateOnlyPrints(t *testing.T) {
	fx := newCLIFixture(t)
	_, err := fx.run(getCmdCosign(), "", fx.txFile, "--from", fx.keyName, "--pqc-key", "pq",
		"--chain-id", testChainID, "--generate-only", "--account-number", "1", "--sequence", "0", "--sign-bytes", "v2")
	if err != nil {
		t.Fatal(err)
	}
	fx.verifyHybrid(t, fx.decodePrinted(t), fx.pqcPub, 1, 0)
}

func TestCosignOfflineRefusals(t *testing.T) {
	fx := newCLIFixture(t)
	base := []string{fx.txFile, "--from", fx.keyName, "--pqc-key", "pq", "--chain-id", testChainID, "--offline"}
	for name, extra := range map[string][]string{
		"no account number": {"--sign-bytes", "v2"},
		"auto sign-bytes":   {"--account-number", "5", "--sequence", "7"},
		"dry-run offline":   {"--account-number", "5", "--sequence", "7", "--sign-bytes", "v2", "--dry-run"},
	} {
		if _, err := fx.run(getCmdCosign(), "", append(append([]string{}, base...), extra...)...); err == nil {
			t.Errorf("%s: accepted", name)
		}
		if fx.out.Len() != 0 {
			t.Errorf("%s: printed a tx", name)
		}
	}
}

// TestRotateKeyRefusesUnsavedRandomKey pins the stranded-key half of
// QSR-2026-0273: a random new key without --new-key-out is refused before
// anything is signed.
func TestRotateKeyRefusesUnsavedRandomKey(t *testing.T) {
	fx := newCLIFixture(t)
	_, err := fx.run(getCmdRotatePQCKey(), "some mnemonic words", "--from", fx.keyName, "--chain-id", testChainID,
		"--new-random", "--offline", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2")
	if err == nil || !strings.Contains(err.Error(), "would not be saved") {
		t.Fatalf("got %v", err)
	}
	if fx.out.Len() != 0 {
		t.Fatal("printed a tx")
	}
}

func TestRotateKeyOfflineSavesKeyThenPrints(t *testing.T) {
	fx := newCLIFixture(t)
	args := []string{"--from", fx.keyName, "--chain-id", testChainID, "--old-derivation", "bridge",
		"--new-random", "--new-key-out", "next", "--offline", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2"}
	if _, err := fx.run(getCmdRotatePQCKey(), "some mnemonic words", args...); err != nil {
		t.Fatal(err)
	}
	path := pqcKeyPath(fx.home, "next")
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("new key not stored: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("new key mode %04o", fi.Mode().Perm())
	}

	oldPub, _, err := qpqc.MLDSA87.KeygenFromSeed(qpqc.Shake256([]byte("some mnemonic words"), 32))
	if err != nil {
		t.Fatal(err)
	}
	body := fx.verifyHybrid(t, fx.decodePrinted(t), oldPub, 5, 7)
	var msg types.MsgRotatePQCKey
	if err := msg.Unmarshal(body.Messages[0].Value); err != nil {
		t.Fatal(err)
	}
	newSk, err := loadPQCPrivKey(fx.home, "next")
	if err != nil {
		t.Fatal(err)
	}
	probe := []byte("probe")
	sig, err := qpqc.MLDSA87.Sign(newSk, probe)
	if err != nil || !qpqc.MLDSA87.Verify(msg.NewPublicKey, probe, sig) {
		t.Fatal("stored key does not match the rotation's new public key")
	}

	// A second run must not replace the stored key.
	before, _ := os.ReadFile(path)
	if _, err := fx.run(getCmdRotatePQCKey(), "some mnemonic words", args...); err == nil {
		t.Fatal("overwrote an existing new-key file")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("existing key file changed")
	}
}

//go:build full

package cli

import (
	"context"
	"strings"
	"testing"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"

	"cosmossdk.io/x/tx/signing/textual"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	qpqc "github.com/qorechain/qorechain-pqc/go"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// withTextual gives the fixture a TxConfig that, like the node's client
// (cmd/qorechaind/cmd/root.go), has SIGN_MODE_TEXTUAL enabled, and stands in
// for the node's denom metadata (none: amounts render in base units on both
// sides).
func (fx *cliFixture) withTextual(t *testing.T) {
	t.Helper()
	noMetadata := func(context.Context, string) (*bankv1beta1.Metadata, error) { return nil, nil }
	cfg, err := authtx.NewTxConfigWithOptions(fx.ctx.Codec.(codec.Codec), authtx.ConfigOptions{
		EnabledSignModes:           append(append([]signing.SignMode{}, authtx.DefaultSignModes...), signing.SignMode_SIGN_MODE_TEXTUAL),
		TextualCoinMetadataQueryFn: noMetadata,
	})
	if err != nil {
		t.Fatal(err)
	}
	fx.ctx = fx.ctx.WithTxConfig(cfg)
	prev := textualCoinMetadata
	textualCoinMetadata = func(client.Context) textual.CoinMetadataQueryFn { return noMetadata }
	t.Cleanup(func() { textualCoinMetadata = prev })
}

// `tx pqc cosign --sign-mode textual` (the only sign mode a Ledger can use
// that also commits to the PQC extension): the auth info names TEXTUAL, the
// PQC signature covers that auth info, and the classical signature covers the
// textual rendering of the final body, extension included.
func TestCosignTextualSignsTheRenderedFinalBody(t *testing.T) {
	fx := newCLIFixture(t)
	fx.withTextual(t)
	_, err := fx.run(getCmdCosign(), "", fx.txFile, "--from", fx.keyName, "--pqc-key", "pq",
		"--chain-id", testChainID, "--generate-only", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2",
		"--sign-mode", "textual")
	if err != nil {
		t.Fatal(err)
	}
	raw := fx.decodePrinted(t)
	var auth txtypes.AuthInfo
	if err := auth.Unmarshal(raw.AuthInfoBytes); err != nil {
		t.Fatal(err)
	}
	if m := auth.SignerInfos[0].ModeInfo.GetSingle().GetMode(); m != signing.SignMode_SIGN_MODE_TEXTUAL {
		t.Fatalf("signer mode %v, want TEXTUAL", m)
	}
	var body txtypes.TxBody
	if err := body.Unmarshal(raw.BodyBytes); err != nil {
		t.Fatal(err)
	}
	if len(body.ExtensionOptions) != 1 || body.ExtensionOptions[0].TypeUrl != types.HybridSigTypeURL {
		t.Fatalf("missing hybrid extension: %+v", body.ExtensionOptions)
	}
	var ext types.PQCHybridSignature
	if err := ext.Unmarshal(body.ExtensionOptions[0].Value); err != nil {
		t.Fatal(err)
	}
	b0, _ := (&txtypes.TxBody{Messages: body.Messages, Memo: body.Memo, TimeoutHeight: body.TimeoutHeight}).Marshal()
	if !qpqc.MLDSA87.Verify(fx.pqcPub, types.HybridSignBytes(testChainID, b0, raw.AuthInfoBytes), ext.PQCSignature) {
		t.Fatal("PQC signature does not verify")
	}

	rec, _ := fx.ctx.Keyring.Key(fx.keyName)
	pub, _ := rec.GetPubKey()
	bz, _ := raw.Marshal()
	tx, err := fx.ctx.TxConfig.TxDecoder()(bz)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := authsigning.GetSignBytesAdapter(context.Background(), fx.ctx.TxConfig.SignModeHandler(), signing.SignMode_SIGN_MODE_TEXTUAL,
		authsigning.SignerData{Address: fx.from.String(), ChainID: testChainID, AccountNumber: 5, Sequence: 7, PubKey: pub}, tx)
	if err != nil {
		t.Fatal(err)
	}
	if !pub.VerifySignature(sb, raw.Signatures[0]) {
		t.Fatal("classical signature does not verify over the textual sign bytes of the final tx")
	}
	doc, _ := (&txtypes.SignDoc{BodyBytes: raw.BodyBytes, AuthInfoBytes: raw.AuthInfoBytes, ChainId: testChainID, AccountNumber: 5}).Marshal()
	if pub.VerifySignature(doc, raw.Signatures[0]) {
		t.Fatal("the classical signature is a DIRECT one")
	}
}

// Amino JSON cannot carry the extension at all (the SDK refuses to build its
// sign doc for a body with extension options), and DIRECT_AUX has no place
// for it either: cosign refuses both with a reason instead of signing.
func TestCosignRefusesSignModesThatCannotCoverTheExtension(t *testing.T) {
	fx := newCLIFixture(t)
	fx.withTextual(t)
	for _, mode := range []string{"amino-json", "direct-aux"} {
		_, err := fx.run(getCmdCosign(), "", fx.txFile, "--from", fx.keyName, "--pqc-key", "pq",
			"--chain-id", testChainID, "--offline", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2",
			"--sign-mode", mode)
		if err == nil || !strings.Contains(err.Error(), "--sign-mode") {
			t.Fatalf("%s: got %v", mode, err)
		}
		if fx.out.Len() != 0 {
			t.Fatalf("%s: printed a tx", mode)
		}
	}
}

// Textual cannot be signed offline: the screens need the chain's denom
// metadata, and rendering without it would not match what the chain verifies.
func TestCosignTextualRefusesOffline(t *testing.T) {
	fx := newCLIFixture(t)
	fx.withTextual(t)
	_, err := fx.run(getCmdCosign(), "", fx.txFile, "--from", fx.keyName, "--pqc-key", "pq",
		"--chain-id", testChainID, "--offline", "--account-number", "5", "--sequence", "7", "--sign-bytes", "v2",
		"--sign-mode", "textual")
	if err == nil || !strings.Contains(err.Error(), "denom metadata") {
		t.Fatalf("got %v", err)
	}
	if fx.out.Len() != 0 {
		t.Fatal("printed a tx")
	}
}

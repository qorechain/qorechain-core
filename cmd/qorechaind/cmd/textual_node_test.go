package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	txsigning "cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/tx/signing/textual"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/qorechain/qorechain-core/app"
)

// The SIGN_MODE_TEXTUAL screens render amounts with the chain's denom
// metadata, read from a node. That node must be the one the command was
// pointed at (--node, or the client.toml of the --home in use), not the node
// of the default home's client config, which is what the client TxConfig
// built at start-up queried. A signature rendered with another node's
// metadata (or none, when the default node is down) does not verify on the
// chain the transaction is meant for.

const textualTestChainID = "qorechain-textual-test"

// fakeNode is a JSON-RPC endpoint that answers abci_query for the bank
// DenomMetadata query with uqor/qor metadata and records every query path.
type fakeNode struct {
	srv   *httptest.Server
	mu    sync.Mutex
	paths []string
}

var fakeMetadata = &bankv1beta1.Metadata{
	Base: "uqor", Display: "qor", Name: "QoreChain", Symbol: "QOR",
	DenomUnits: []*bankv1beta1.DenomUnit{{Denom: "uqor", Exponent: 0}, {Denom: "qor", Exponent: 6}},
}

func newFakeNode(t *testing.T) *fakeNode {
	t.Helper()
	n := &fakeNode{}
	n.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params struct {
				Path string `json:"path"`
				Data string `json:"data"`
			} `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		n.mu.Lock()
		n.paths = append(n.paths, req.Method+" "+req.Params.Path)
		n.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		reply := func(val []byte) {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{"response":{"code":0,"log":"","info":"","index":"0","value":"` +
				base64.StdEncoding.EncodeToString(val) + `","height":"1","codespace":""}}}`))
		}
		if req.Method == "abci_query" && req.Params.Path == "/cosmos.auth.v1beta1.Query/Account" {
			// The signer's account, as tx sign checks it exists.
			data, _ := hex.DecodeString(req.Params.Data)
			var q authtypes.QueryAccountRequest
			_ = q.Unmarshal(data)
			acc, err := codectypes.NewAnyWithValue(&authtypes.BaseAccount{Address: q.Address, AccountNumber: 5, Sequence: 3})
			if err != nil {
				t.Error(err)
			}
			val, _ := (&authtypes.QueryAccountResponse{Account: acc}).Marshal()
			reply(val)
			return
		}
		if req.Method != "abci_query" || req.Params.Path != "/cosmos.bank.v1beta1.Query/DenomMetadata" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"not served"}}`))
			return
		}
		val, err := (&banktypes.QueryDenomMetadataResponse{Metadata: banktypes.Metadata{
			Base: "uqor", Display: "qor", Name: "QoreChain", Symbol: "QOR",
			DenomUnits: []*banktypes.DenomUnit{{Denom: "uqor", Exponent: 0}, {Denom: "qor", Exponent: 6}},
		}}).Marshal()
		if err != nil {
			t.Error(err)
		}
		reply(val)
	}))
	t.Cleanup(n.srv.Close)
	return n
}

func (n *fakeNode) url() string { return n.srv.URL }

func (n *fakeNode) metadataQueries() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	c := 0
	for _, p := range n.paths {
		if p == "abci_query /cosmos.bank.v1beta1.Query/DenomMetadata" {
			c++
		}
	}
	return c
}

// runRoot runs one qorechaind command line in home and returns stdout.
func runRoot(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	root.PersistentFlags().String(flags.FlagHome, home, "node home")
	var errOut bytes.Buffer
	root.SetOut(&errOut)
	root.SetErr(&errOut)
	root.SetArgs(append(args, "--home", home))
	var err error
	out := captureStdout(t, func() { err = root.Execute() })
	if err != nil {
		return out + errOut.String(), err
	}
	return out + errOut.String(), nil
}

type textualFixture struct {
	home   string
	txFile string
}

func newTextualFixture(t *testing.T) textualFixture {
	t.Helper()
	home := t.TempDir()
	if out, err := runRoot(t, home, "keys", "add", "a", "--keyring-backend", "test", "--output", "json"); err != nil {
		t.Fatalf("keys add: %v\n%s", err, out)
	}
	addr, err := runRoot(t, home, "keys", "show", "a", "-a", "--keyring-backend", "test")
	if err != nil {
		t.Fatalf("keys show: %v\n%s", err, addr)
	}
	addr = strings.TrimSpace(addr)
	unsigned, err := runRoot(t, home, "tx", "bank", "send", addr, addr, "1500000uqor", "--generate-only",
		"--chain-id", textualTestChainID, "--fees", "2000uqor", "--keyring-backend", "test")
	if err != nil {
		t.Fatalf("generate-only: %v\n%s", err, unsigned)
	}
	txFile := filepath.Join(home, "unsigned.json")
	if err := os.WriteFile(txFile, []byte(strings.TrimSpace(unsigned)), 0o600); err != nil {
		t.Fatal(err)
	}
	return textualFixture{home: home, txFile: txFile}
}

func (fx textualFixture) sign(t *testing.T, mode string, extra ...string) (*txtypes.Tx, error) {
	t.Helper()
	args := append([]string{"tx", "sign", fx.txFile, "--from", "a", "--sign-mode", mode,
		"--chain-id", textualTestChainID, "--account-number", "5", "--sequence", "3",
		"--keyring-backend", "test", "--output", "json"}, extra...)
	out, err := runRoot(t, fx.home, args...)
	if err != nil {
		return nil, err
	}
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no tx printed:\n%s", out)
	}
	dec := json.NewDecoder(strings.NewReader(out[start:]))
	var raw json.RawMessage
	if err := dec.Decode(&raw); err != nil {
		t.Fatalf("printed tx: %v\n%s", err, out)
	}
	clientCtx := NewRootCmdClientContextForTest(t)
	var tx txtypes.Tx
	if err := clientCtx.Codec.UnmarshalJSON(raw, &tx); err != nil {
		t.Fatalf("decode printed tx: %v", err)
	}
	return &tx, nil
}

// NewRootCmdClientContextForTest returns a client context wired like the
// root command's (depinject over the app config), for decoding what it printed.
func NewRootCmdClientContextForTest(t *testing.T) client.Context {
	t.Helper()
	var clientCtx client.Context
	if err := depinject.Inject(
		depinject.Configs(app.AppConfig, depinject.Supply(log.NewNopLogger()), depinject.Provide(ProvideClientContext)),
		&clientCtx,
	); err != nil {
		t.Fatal(err)
	}
	return clientCtx
}

// verifyTextual checks the printed signature against textual sign bytes
// rendered with the fake node's metadata, i.e. what that chain verifies.
func verifyTextual(t *testing.T, tx *txtypes.Tx) bool {
	t.Helper()
	clientCtx := NewRootCmdClientContextForTest(t)
	h, err := textual.NewSignModeHandler(textual.SignModeOptions{
		CoinMetadataQuerier: func(context.Context, string) (*bankv1beta1.Metadata, error) { return fakeMetadata, nil },
		FileResolver:        clientCtx.InterfaceRegistry,
	})
	if err != nil {
		t.Fatal(err)
	}
	bz, err := clientCtx.Codec.Marshal(tx)
	if err != nil {
		t.Fatal(err)
	}
	sdkTx, err := clientCtx.TxConfig.TxDecoder()(bz)
	if err != nil {
		t.Fatal(err)
	}
	var pub cryptotypes.PubKey
	if err := clientCtx.InterfaceRegistry.UnpackAny(tx.AuthInfo.SignerInfos[0].PublicKey, &pub); err != nil {
		t.Fatal(err)
	}
	sb, err := authsigning.GetSignBytesAdapter(context.Background(), txsigning.NewHandlerMap(h), signing.SignMode_SIGN_MODE_TEXTUAL,
		authsigning.SignerData{Address: sdk.AccAddress(pub.Address()).String(), ChainID: textualTestChainID,
			AccountNumber: 5, Sequence: 3, PubKey: pub}, sdkTx)
	if err != nil {
		t.Fatal(err)
	}
	return pub.VerifySignature(sb, tx.Signatures[0])
}

func TestTxSignTextualReadsMetadataFromTheNodeFlag(t *testing.T) {
	fx := newTextualFixture(t)
	node := newFakeNode(t)
	tx, err := fx.sign(t, "textual", "--node", node.url())
	if err != nil {
		t.Fatalf("tx sign --sign-mode textual --node %s: %v", node.url(), err)
	}
	if node.metadataQueries() == 0 {
		t.Fatal("the denom metadata was not read from the --node given")
	}
	if !verifyTextual(t, tx) {
		t.Fatal("the textual signature does not verify against screens rendered with that node's metadata")
	}
}

// Without --node, the node of the --home's client.toml is used.
func TestTxSignTextualReadsMetadataFromTheHomeClientConfig(t *testing.T) {
	fx := newTextualFixture(t)
	node := newFakeNode(t)
	cfgPath := filepath.Join(fx.home, "config", "client.toml")
	cfg, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(cfg), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "node = ") {
			lines[i] = `node = "` + node.url() + `"`
		}
	}
	if err := os.WriteFile(cfgPath, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	tx, err := fx.sign(t, "textual")
	if err != nil {
		t.Fatalf("tx sign --sign-mode textual: %v", err)
	}
	if node.metadataQueries() == 0 || !verifyTextual(t, tx) {
		t.Fatal("textual did not use the node of the home's client config")
	}
}

// DIRECT is untouched: same signature as before, and no metadata query.
func TestTxSignDirectIsUnchanged(t *testing.T) {
	fx := newTextualFixture(t)
	node := newFakeNode(t)
	tx, err := fx.sign(t, "direct", "--node", node.url())
	if err != nil {
		t.Fatalf("%v (node saw %v)", err, node.paths)
	}
	if n := node.metadataQueries(); n != 0 {
		t.Fatalf("DIRECT signing queried denom metadata %d times", n)
	}
	clientCtx := NewRootCmdClientContextForTest(t)
	body, _ := clientCtx.Codec.Marshal(tx.Body)
	auth, _ := clientCtx.Codec.Marshal(tx.AuthInfo)
	doc, _ := (&txtypes.SignDoc{BodyBytes: body, AuthInfoBytes: auth, ChainId: textualTestChainID, AccountNumber: 5}).Marshal()
	var pub cryptotypes.PubKey
	if err := clientCtx.InterfaceRegistry.UnpackAny(tx.AuthInfo.SignerInfos[0].PublicKey, &pub); err != nil {
		t.Fatal(err)
	}
	if !pub.VerifySignature(doc, tx.Signatures[0]) {
		t.Fatal("DIRECT signature does not verify")
	}
	if m := tx.AuthInfo.SignerInfos[0].ModeInfo.GetSingle().GetMode(); m != signing.SignMode_SIGN_MODE_DIRECT {
		t.Fatalf("mode %v", m)
	}
}

// The rebuilt TxConfig differs from the first one only in where textual reads
// metadata: same sign modes, same DIRECT and amino JSON bytes. Offline keeps
// the first TxConfig.
func TestTextualRebuildKeepsEveryOtherSignMode(t *testing.T) {
	var (
		clientCtx client.Context
		opts      authtx.ConfigOptions
	)
	if err := depinject.Inject(
		depinject.Configs(app.AppConfig, depinject.Supply(log.NewNopLogger()), depinject.Provide(ProvideClientContext)),
		&clientCtx, &opts,
	); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	flags.AddTxFlagsToCmd(cmd)

	off, err := withTextualMetadataFromNode(clientCtx.WithOffline(true), cmd, opts)
	if err != nil {
		t.Fatal(err)
	}
	if off.TxConfig != clientCtx.TxConfig {
		t.Fatal("an offline command got a new TxConfig")
	}

	if err := cmd.Flags().Set(flags.FlagNode, "http://127.0.0.1:1"); err != nil {
		t.Fatal(err)
	}
	re, err := withTextualMetadataFromNode(clientCtx, cmd, opts)
	if err != nil {
		t.Fatal(err)
	}
	if re.TxConfig == clientCtx.TxConfig {
		t.Fatal("the TxConfig was not rebuilt")
	}
	if re.NodeURI != clientCtx.NodeURI {
		t.Fatalf("the context handed on changed its node: %q -> %q", clientCtx.NodeURI, re.NodeURI)
	}
	before, after := clientCtx.TxConfig.SignModeHandler().SupportedModes(), re.TxConfig.SignModeHandler().SupportedModes()
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("sign modes changed: %v -> %v", before, after)
	}

	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	b := clientCtx.TxConfig.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewInt64Coin("uqor", 7)))); err != nil {
		t.Fatal(err)
	}
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uqor", 2000)))
	b.SetGasLimit(200000)
	b.SetMemo("same bytes")
	for _, mode := range []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT, signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON} {
		if err := b.SetSignatures(signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{SignMode: mode}, Sequence: 3}); err != nil {
			t.Fatal(err)
		}
		sd := authsigning.SignerData{Address: addr.String(), ChainID: textualTestChainID, AccountNumber: 5, Sequence: 3, PubKey: priv.PubKey()}
		x, err := authsigning.GetSignBytesAdapter(context.Background(), clientCtx.TxConfig.SignModeHandler(), mode, sd, b.GetTx())
		if err != nil {
			t.Fatal(err)
		}
		y, err := authsigning.GetSignBytesAdapter(context.Background(), re.TxConfig.SignModeHandler(), mode, sd, b.GetTx())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(x, y) {
			t.Fatalf("%v sign bytes changed", mode)
		}
	}
}

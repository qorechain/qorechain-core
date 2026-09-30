package app

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var testEncoding = moduletestutil.MakeTestEncodingConfig(bank.AppModuleBasic{})

// buildSignedTx builds and round-trips through the real encoder and decoder a
// MsgSend whose only signer is an ORDINARY secp256k1 key, with sigData as its
// signature. The signature data comes from mode_info, not from the pubkey, which
// is exactly how the attack carries a deep multisig on a plain funded account.
func buildSignedTx(t *testing.T, sigData signing.SignatureData) sdk.Tx {
	t.Helper()
	bz := buildSignedTxBytes(t, sigData)
	tx, err := testEncoding.TxConfig.TxDecoder()(bz)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return tx
}

func buildSignedTxBytes(t *testing.T, sigData signing.SignatureData) []byte {
	t.Helper()
	priv := secp256k1.GenPrivKey()
	from := sdk.AccAddress(priv.PubKey().Address())
	to := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	b := testEncoding.TxConfig.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("uqor", 1)))); err != nil {
		t.Fatal(err)
	}
	b.SetGasLimit(200_000)
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uqor", 1)))
	if err := b.SetSignatures(signing.SignatureV2{PubKey: priv.PubKey(), Data: sigData}); err != nil {
		t.Fatal(err)
	}
	bz, err := testEncoding.TxConfig.TxEncoder()(b.GetTx())
	if err != nil {
		t.Fatal(err)
	}
	return bz
}

func failIfReached(t *testing.T) sdk.AnteHandler {
	return func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		t.Helper()
		t.Fatal("the decorator that expands signatures was reached")
		return ctx, nil
	}
}

// sentinel stands where SetPubKeyDecorator sits in the real chain: it counts how
// often a transaction got that far.
type sentinel struct{ hits atomic.Int64 }

func (s *sentinel) AnteHandle(ctx sdk.Context, tx sdk.Tx, sim bool, next sdk.AnteHandler) (sdk.Context, error) {
	s.hits.Add(1)
	return next(ctx, tx, sim)
}

type okBank struct {
	banktypes.UnimplementedMsgServer
}

func (*okBank) Send(context.Context, *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	return &banktypes.MsgSendResponse{}, nil
}

// newGuardedBaseApp is a bare BaseApp wired the way ours is in the parts that
// matter here: a real mempool (so the default ProcessProposal handler really
// re-runs the ante chain), the default proposal handler, and an ante chain of the
// multisig guard followed by a sentinel standing in for SetPubKey.
func newGuardedBaseApp(t *testing.T, ante sdk.AnteHandler) *baseapp.BaseApp {
	t.Helper()
	key := storetypes.NewKVStoreKey("test")
	bapp := baseapp.NewBaseApp("guard-test", log.NewNopLogger(), dbm.NewMemDB(),
		testEncoding.TxConfig.TxDecoder(), baseapp.SetChainID("guard-test"))
	bapp.MountStores(key)
	bapp.SetInterfaceRegistry(testEncoding.InterfaceRegistry)

	router := baseapp.NewMsgServiceRouter()
	router.SetInterfaceRegistry(testEncoding.InterfaceRegistry)
	banktypes.RegisterMsgServer(router, &okBank{})
	bapp.SetMsgServiceRouter(router)

	bapp.SetAnteHandler(ante)
	mp := sdkmempool.NewSenderNonceMempool(sdkmempool.SenderNonceMaxTxOpt(1000))
	bapp.SetMempool(mp)
	h := baseapp.NewDefaultProposalHandler(mp, bapp)
	bapp.SetPrepareProposal(h.PrepareProposalHandler())
	bapp.SetProcessProposal(h.ProcessProposalHandler())

	if err := bapp.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	if _, err := bapp.InitChain(&abci.RequestInitChain{
		ChainId:       "guard-test",
		InitialHeight: 1,
	}); err != nil {
		t.Fatal(err)
	}
	return bapp
}

func guardedChain(s *sentinel) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(NewMultisigDepthGuardDecorator(nil), s)
}

// THE ATTACK, end to end through baseapp. A depth-100 multisig on a plain key is
// refused in simulate, CheckTx and ProcessProposal before anything that would
// expand it, and quickly. FinalizeBlock is unchanged while
// FeatureMultisigSignatureBound is off: that leg needs a coordinated height and is
// covered in ante_multisig_block_test.go.
func TestDeepMultisigRefusedInEveryNodeLocalMode(t *testing.T) {
	s := &sentinel{}
	bapp := newGuardedBaseApp(t, guardedChain(s))
	deep := buildSignedTxBytes(t, chain(100))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	_, _, err := bapp.Simulate(deep)
	if !errors.Is(err, errortypes.ErrTooManySignatures) {
		t.Fatalf("simulate: got %v, want ErrTooManySignatures", err)
	}

	res, err := bapp.CheckTx(&abci.RequestCheckTx{Tx: deep, Type: abci.CheckTxType_New})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != errortypes.ErrTooManySignatures.ABCICode() {
		t.Fatalf("CheckTx: code %d (%s), want ErrTooManySignatures", res.Code, res.Log)
	}
	if len(res.Events) != 0 {
		t.Fatalf("CheckTx emitted events for a refused tx: %v", res.Events)
	}

	pp, err := bapp.ProcessProposal(&abci.RequestProcessProposal{Height: 1, Txs: [][]byte{deep}})
	if err != nil {
		t.Fatal(err)
	}
	if pp.Status != abci.ResponseProcessProposal_REJECT {
		t.Fatalf("ProcessProposal: got %v, want REJECT for a proposal carrying the tx", pp.Status)
	}

	prep, err := bapp.PrepareProposal(&abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 1 << 20, Txs: [][]byte{deep}})
	if err != nil {
		t.Fatal(err)
	}
	if len(prep.Txs) != 0 {
		t.Fatalf("PrepareProposal kept the tx in the block it builds")
	}

	runtime.ReadMemStats(&after)
	if delta := after.TotalAlloc - before.TotalAlloc; delta > 1<<20 {
		t.Fatalf("refusing the tx in four modes allocated %d bytes", delta)
	}
	if n := s.hits.Load(); n != 0 {
		t.Fatalf("the SetPubKey stand-in was reached %d times", n)
	}

	// Block execution is untouched while the v3.2.2 switch is off: the tx gets
	// past the guard.
	if _, err := bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{deep}}); err != nil {
		t.Fatal(err)
	}
	if n := s.hits.Load(); n != 1 {
		t.Fatalf("FinalizeBlock: SetPubKey stand-in reached %d times, want 1 (unchanged behaviour)", n)
	}
}

// Legitimate multisigs keep working in every mode: flat k-of-n and one level of
// nesting, up to 32 members per node.
func TestLegitimateMultisigStillSimulatesAndIsAdmitted(t *testing.T) {
	for name, data := range map[string]signing.SignatureData{
		"single key":         single(),
		"flat 2-of-3":        nOf(3, single),
		"flat 32":            nOf(32, single),
		"two-level multisig": nOf(3, func() signing.SignatureData { return nOf(3, single) }),
	} {
		t.Run(name, func(t *testing.T) {
			s := &sentinel{}
			bapp := newGuardedBaseApp(t, guardedChain(s))
			bz := buildSignedTxBytes(t, data)

			// The bare app meters nothing, so success is the estimate here.
			if _, _, err := bapp.Simulate(bz); err != nil {
				t.Fatalf("simulate: %v", err)
			}
			res, err := bapp.CheckTx(&abci.RequestCheckTx{Tx: bz, Type: abci.CheckTxType_New})
			if err != nil || res.Code != 0 {
				t.Fatalf("CheckTx: %v code %d %s", err, res.Code, res.Log)
			}
			pp, err := bapp.ProcessProposal(&abci.RequestProcessProposal{Height: 1, Txs: [][]byte{bz}})
			if err != nil || pp.Status != abci.ResponseProcessProposal_ACCEPT {
				t.Fatalf("ProcessProposal: %v %v", err, pp.Status)
			}
			if s.hits.Load() != 3 {
				t.Fatalf("SetPubKey stand-in reached %d times, want 3", s.hits.Load())
			}
		})
	}
}

// Depth 3 and breadth 33 are refused in the node-local modes; they were the
// cases the old rule (depth 3, no breadth limit) let through.
func TestTighterBoundInNodeLocalModes(t *testing.T) {
	for name, data := range map[string]signing.SignatureData{
		"depth 3":    chain(3),
		"breadth 33": nOf(33, single),
	} {
		t.Run(name, func(t *testing.T) {
			s := &sentinel{}
			bapp := newGuardedBaseApp(t, guardedChain(s))
			bz := buildSignedTxBytes(t, data)
			if _, _, err := bapp.Simulate(bz); !errors.Is(err, errortypes.ErrTooManySignatures) {
				t.Fatalf("simulate: %v", err)
			}
			pp, _ := bapp.ProcessProposal(&abci.RequestProcessProposal{Height: 1, Txs: [][]byte{bz}})
			if pp.Status != abci.ResponseProcessProposal_REJECT {
				t.Fatalf("ProcessProposal: %v", pp.Status)
			}
		})
	}
}

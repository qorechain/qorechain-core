package burn

import (
	"context"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/qorechain/qorechain-core/x/burn/types"
)

// These tests drive the failed_tx channel through a real BaseApp, because the
// channel's correctness rests on baseapp's commit rules (ante writes kept on
// message failure, post-handler writes kept only on success) and on the ante
// value reaching the post handler. The keeper is a stand-in that stores the
// accumulator the way the real one does: off the gas meter, gated by a switch.

var failedFeeEncoding = moduletestutil.MakeTestEncodingConfig(bank.AppModuleBasic{})

// accumulatorKeeper is the part of BurnKeeper the decorators use.
type accumulatorKeeper struct {
	BurnKeeper // nil: any other method panics, which the test would notice
	key        *storetypes.KVStoreKey
	on         bool
}

func (k *accumulatorKeeper) get(ctx sdk.Context) math.Int {
	bz := ctx.KVStore(k.key).Get(types.BlockFailedFeesKey)
	if bz == nil {
		return math.ZeroInt()
	}
	v, _ := math.NewIntFromString(string(bz))
	return v
}

func (k *accumulatorKeeper) set(ctx sdk.Context, v math.Int) {
	if !v.IsPositive() {
		ctx.KVStore(k.key).Delete(types.BlockFailedFeesKey)
		return
	}
	ctx.KVStore(k.key).Set(types.BlockFailedFeesKey, []byte(v.String()))
}

func (k *accumulatorKeeper) RecordTxFee(ctx sdk.Context, fee math.Int) bool {
	if !k.on {
		return false
	}
	free := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	k.set(free, k.get(free).Add(fee))
	return true
}

func (k *accumulatorKeeper) ReleaseTxFee(ctx sdk.Context, fee math.Int) {
	free := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	k.set(free, k.get(free).Sub(fee))
}

// scriptedBank succeeds, fails or runs out of gas depending on the amount sent,
// and writes to the store so that gas_used includes real store costs.
type scriptedBank struct {
	banktypes.UnimplementedMsgServer
	key *storetypes.KVStoreKey
}

func (b *scriptedBank) Send(goCtx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ctx.KVStore(b.key).Set([]byte("last-send"), []byte(msg.Amount.String()))
	switch msg.Amount.AmountOf("uqor").Int64() {
	case 13:
		return nil, fmt.Errorf("scripted failure")
	case 99:
		ctx.GasMeter().ConsumeGas(10_000_000, "scripted out of gas")
	}
	return &banktypes.MsgSendResponse{}, nil
}

type failedFeeApp struct {
	bapp   *baseapp.BaseApp
	keeper *accumulatorKeeper
}

// newFailedFeeApp builds a BaseApp whose ante chain sets up the gas meter and,
// when withChannel is set, runs the failed-fee decorator and post handler.
func newFailedFeeApp(t *testing.T, withChannel, on bool) failedFeeApp {
	t.Helper()
	return newFailedFeeAppWithChecker(t, withChannel, on, nil)
}

func newFailedFeeAppWithChecker(t *testing.T, withChannel, on bool, checker ante.TxFeeChecker) failedFeeApp {
	t.Helper()
	key := storetypes.NewKVStoreKey("burn")
	bapp := baseapp.NewBaseApp("failed-fee", log.NewNopLogger(), dbm.NewMemDB(),
		failedFeeEncoding.TxConfig.TxDecoder(), baseapp.SetChainID("failed-fee"))
	bapp.MountStores(key)
	bapp.SetInterfaceRegistry(failedFeeEncoding.InterfaceRegistry)
	router := baseapp.NewMsgServiceRouter()
	router.SetInterfaceRegistry(failedFeeEncoding.InterfaceRegistry)
	banktypes.RegisterMsgServer(router, &scriptedBank{key: key})
	bapp.SetMsgServiceRouter(router)

	k := &accumulatorKeeper{key: key, on: on}
	decorators := []sdk.AnteDecorator{ante.NewSetUpContextDecorator()}
	if withChannel {
		decorators = append(decorators, NewBurnFailedFeeDecorator(k, checker))
		bapp.SetPostHandler(sdk.ChainPostDecorators(NewBurnFailedFeePostDecorator(k)))
	}
	bapp.SetAnteHandler(sdk.ChainAnteDecorators(decorators...))
	if err := bapp.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	if _, err := bapp.InitChain(&abci.RequestInitChain{ChainId: "failed-fee", InitialHeight: 1}); err != nil {
		t.Fatal(err)
	}
	return failedFeeApp{bapp: bapp, keeper: k}
}

func (a failedFeeApp) accumulated(t *testing.T) math.Int {
	t.Helper()
	ctx := a.bapp.NewUncachedContext(false, cmtproto.Header{})
	return a.keeper.get(ctx)
}

// sendTx is a MsgSend of `amount` uqor paying `fee` uqor, as bytes.
func sendTx(t *testing.T, amount, fee int64) []byte {
	t.Helper()
	priv := secp256k1.GenPrivKey()
	from := sdk.AccAddress(priv.PubKey().Address())
	to := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	b := failedFeeEncoding.TxConfig.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(from, to, sdk.NewCoins(sdk.NewInt64Coin("uqor", amount)))); err != nil {
		t.Fatal(err)
	}
	b.SetGasLimit(200_000)
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uqor", fee)))
	if err := b.SetSignatures(signing.SignatureV2{PubKey: priv.PubKey(), Data: &signing.SingleSignatureData{SignMode: signing.SignMode_SIGN_MODE_DIRECT, Signature: []byte{1}}}); err != nil {
		t.Fatal(err)
	}
	bz, err := failedFeeEncoding.TxConfig.TxEncoder()(b.GetTx())
	if err != nil {
		t.Fatal(err)
	}
	return bz
}

// block is: success (fee 100), message failure (fee 7), out of gas (fee 11),
// success (fee 1000).
func block(t *testing.T) [][]byte {
	return [][]byte{sendTx(t, 1, 100), sendTx(t, 13, 7), sendTx(t, 99, 11), sendTx(t, 2, 1000)}
}

func finalize(t *testing.T, a failedFeeApp, txs [][]byte) *abci.ResponseFinalizeBlock {
	t.Helper()
	res, err := a.bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: txs})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.TxResults) != len(txs) {
		t.Fatalf("%d results for %d txs", len(res.TxResults), len(txs))
	}
	return res
}

// With the channel on, what is left in the accumulator after the block is the
// fees of exactly the failed transactions (message failure and out of gas), and
// every transaction's code, gas_used and events are those of an app without the
// channel at all.
func TestFailedFeeChannelCountsExactlyTheFailedTxs(t *testing.T) {
	txs := block(t)

	plain := newFailedFeeApp(t, false, false)
	base := finalize(t, plain, txs)

	on := newFailedFeeApp(t, true, true)
	got := finalize(t, on, txs)

	wantCodes := []bool{true, false, false, true}
	for i := range txs {
		if (got.TxResults[i].Code == 0) != wantCodes[i] {
			t.Fatalf("tx %d: code %d (%s)", i, got.TxResults[i].Code, got.TxResults[i].Log)
		}
		b, g := base.TxResults[i], got.TxResults[i]
		if b.Code != g.Code || b.GasUsed != g.GasUsed || b.GasWanted != g.GasWanted || len(b.Events) != len(g.Events) {
			t.Fatalf("tx %d differs from the app without the channel: code %d/%d gas_used %d/%d events %d/%d",
				i, b.Code, g.Code, b.GasUsed, g.GasUsed, len(b.Events), len(g.Events))
		}
	}
	if _, err := on.bapp.Commit(); err != nil {
		t.Fatal(err)
	}
	if acc := on.accumulated(t); !acc.Equal(math.NewInt(7 + 11)) {
		t.Fatalf("accumulator holds %s, want 18 (the two failed txs' fees)", acc)
	}
}

// With the channel's switch off, the decorators change nothing at all: same
// results and the same app hash as an app that does not have them.
func TestFailedFeeChannelOffIsByteIdentical(t *testing.T) {
	txs := block(t)
	plain := newFailedFeeApp(t, false, false)
	off := newFailedFeeApp(t, true, false)
	a, b := finalize(t, plain, txs), finalize(t, off, txs)
	for i := range txs {
		x, y := a.TxResults[i], b.TxResults[i]
		if x.Code != y.Code || x.GasUsed != y.GasUsed || x.Log != y.Log || len(x.Events) != len(y.Events) {
			t.Fatalf("tx %d differs with the switch off", i)
		}
	}
	if string(a.AppHash) != string(b.AppHash) {
		t.Fatalf("app hash differs with the switch off: %X vs %X", a.AppHash, b.AppHash)
	}
	if _, err := off.bapp.Commit(); err != nil {
		t.Fatal(err)
	}
	if acc := off.accumulated(t); !acc.IsZero() {
		t.Fatalf("accumulator written with the switch off: %s", acc)
	}
}

// CheckTx and simulation never touch the accumulator; only block execution
// counts.
func TestFailedFeeChannelIgnoresCheckTxAndSimulate(t *testing.T) {
	on := newFailedFeeApp(t, true, true)
	failing := sendTx(t, 13, 7)
	if _, _, err := on.bapp.Simulate(failing); err == nil {
		t.Fatal("simulate of the scripted failure succeeded")
	}
	if _, err := on.bapp.CheckTx(&abci.RequestCheckTx{Tx: failing, Type: abci.CheckTxType_New}); err != nil {
		t.Fatal(err)
	}
	finalize(t, on, nil)
	if _, err := on.bapp.Commit(); err != nil {
		t.Fatal(err)
	}
	if acc := on.accumulated(t); !acc.IsZero() {
		t.Fatalf("accumulator written outside block execution: %s", acc)
	}
}

// The accumulator records what the fee checker charged, not what the
// transaction declares. With a dynamic-fee extension the charged fee can be far
// below the declared one; counting the declared amount would let one cheap
// failing transaction with an inflated declared fee make EndBlock burn a share
// of everybody else's fees.
func TestFailedFeeChannelRecordsTheChargedFee(t *testing.T) {
	// A checker that charges a tenth of the declared fee, as a dynamic-fee
	// transaction with a large fee cap and no tip is charged base fee * gas.
	tenth := func(_ sdk.Context, tx sdk.Tx) (sdk.Coins, int64, error) {
		declared := tx.(sdk.FeeTx).GetFee().AmountOf("uqor")
		return sdk.NewCoins(sdk.NewCoin("uqor", declared.QuoRaw(10))), 0, nil
	}
	on := newFailedFeeAppWithChecker(t, true, true, tenth)
	// success (declared 100), failure with an inflated declared fee (1e15),
	// out of gas (declared 110).
	txs := [][]byte{sendTx(t, 1, 100), sendTx(t, 13, 1_000_000_000_000_000), sendTx(t, 99, 110)}
	finalize(t, on, txs)
	if _, err := on.bapp.Commit(); err != nil {
		t.Fatal(err)
	}
	want := math.NewInt(100_000_000_000_000 + 11)
	if acc := on.accumulated(t); !acc.Equal(want) {
		t.Fatalf("accumulator holds %s, want %s (the charged fees of the failed txs)", acc, want)
	}

	// A checker that refuses: nothing is recorded (DeductFee would have refused
	// the transaction before this decorator in the real chain).
	refuse := func(sdk.Context, sdk.Tx) (sdk.Coins, int64, error) { return nil, 0, fmt.Errorf("refused") }
	r := newFailedFeeAppWithChecker(t, true, true, refuse)
	finalize(t, r, [][]byte{sendTx(t, 13, 7)})
	if _, err := r.bapp.Commit(); err != nil {
		t.Fatal(err)
	}
	if acc := r.accumulated(t); !acc.IsZero() {
		t.Fatalf("accumulator holds %s after a refused fee check", acc)
	}
}

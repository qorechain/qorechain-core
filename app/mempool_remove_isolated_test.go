package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// panickingPool is a pool whose removal check runs out of the gas of its own
// view of the state, as the EVM pool's did when its shared meter was spent.
type panickingPool struct {
	*sdkmempool.PriorityNonceMempool[math.Int]
	panicOnRemove bool
	removeErr     error
	removes       int
}

func (p *panickingPool) Remove(tx sdk.Tx) error {
	p.removes++
	if p.panicOnRemove {
		panic(storetypes.ErrorOutOfGas{Descriptor: "ReadFlat"})
	}
	// A calm pool that does not hold the transaction (a node that never
	// saw it before the block).
	if p.removeErr == nil {
		return sdkmempool.ErrTxNotFound
	}
	return p.removeErr
}

// A panic in the pool's removal never leaves the wrapper (it is the node's
// own business); an error is passed on unchanged.
func TestEvictingMempoolRemoveContainsPoolPanics(t *testing.T) {
	for _, tx := range []sdk.Tx{
		&memTx{id: 1, signer: addr(1), msgs: []sdk.Msg{&evmtypes.MsgEthereumTx{}}},
		&memTx{id: 2, signer: addr(2), msgs: []sdk.Msg{&banktypes.MsgSend{}}},
	} {
		inner := &panickingPool{PriorityNonceMempool: newMemPool(10), panicOnRemove: true}
		m := newEvictingMempool(inner, memTxPriority(), log.NewNopLogger())
		m.signers = memTxSigners{}
		var err error
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("the pool's panic left Remove: %v", r)
				}
			}()
			err = m.Remove(tx)
		}()
		if err != nil || inner.removes != 1 {
			t.Fatalf("Remove after a pool panic: err %v, pool asked %d times", err, inner.removes)
		}

		want := errors.New("pool error")
		inner.panicOnRemove, inner.removeErr = false, want
		if err := m.Remove(tx); !errors.Is(err, want) {
			t.Fatalf("an error of the pool was not passed on: %v", err)
		}
	}
}

// End to end through baseapp, which removes every transaction of a block from
// the pool after its ante handler and fails the transaction on any error but
// not-found: a node whose pool panics there executes the block exactly as a
// node whose pool does not.
func TestBlockResultsDoNotDependOnThePool(t *testing.T) {
	run := func(panicOnRemove bool) ([]*abci.ExecTxResult, []byte) {
		inner := &panickingPool{PriorityNonceMempool: newMemPool(100), panicOnRemove: panicOnRemove}
		m := newEvictingMempool(inner, memTxPriority(), log.NewNopLogger())

		key := storetypes.NewKVStoreKey("test")
		bapp := baseapp.NewBaseApp("pool-test", log.NewNopLogger(), dbm.NewMemDB(),
			testEncoding.TxConfig.TxDecoder(), baseapp.SetChainID("pool-test"))
		bapp.MountStores(key)
		bapp.SetInterfaceRegistry(testEncoding.InterfaceRegistry)
		router := baseapp.NewMsgServiceRouter()
		router.SetInterfaceRegistry(testEncoding.InterfaceRegistry)
		banktypes.RegisterMsgServer(router, &okBank{})
		bapp.SetMsgServiceRouter(router)
		bapp.SetAnteHandler(func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			ctx.KVStore(key).Set([]byte("ante"), []byte{1})
			return ctx.WithGasMeter(storetypes.NewGasMeter(200_000)), nil
		})
		bapp.SetMempool(m)
		if err := bapp.LoadLatestVersion(); err != nil {
			t.Fatal(err)
		}
		if _, err := bapp.InitChain(&abci.RequestInitChain{ChainId: "pool-test", InitialHeight: 1}); err != nil {
			t.Fatal(err)
		}
		res, err := bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{
			buildSignedTxBytes(t, single()), buildSignedTxBytes(t, single()),
		}})
		if err != nil {
			t.Fatal(err)
		}
		if inner.removes != 2 {
			t.Fatalf("the pool was asked %d times, want once per transaction", inner.removes)
		}
		return res.TxResults, res.AppHash
	}
	calm, calmHash := run(false)
	loaded, loadedHash := run(true)
	for i := range calm {
		if calm[i].Code != 0 || loaded[i].Code != calm[i].Code || loaded[i].GasUsed != calm[i].GasUsed {
			t.Fatalf("tx %d: code %d gas %d with a calm pool, code %d gas %d (%s) with a panicking one",
				i, calm[i].Code, calm[i].GasUsed, loaded[i].Code, loaded[i].GasUsed, loaded[i].Log)
		}
	}
	if string(calmHash) != string(loadedHash) {
		t.Fatalf("app hash %X with a calm pool, %X with a panicking one", calmHash, loadedHash)
	}
}

// A panic in the CheckTx handler (the EVM pool's nonce-gap insert runs
// outside baseapp's recovery) is a refusal, not a stopped node; an answer
// passes through unchanged.
func TestCheckTxHandlerPanicIsARefusal(t *testing.T) {
	panicking := recoveringCheckTxHandler(func(sdk.RunTx, *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
		panic(storetypes.ErrorOutOfGas{Descriptor: "ReadFlat"})
	}, log.NewNopLogger())
	var res *abci.ResponseCheckTx
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("the handler's panic left CheckTx: %v", r)
			}
		}()
		res, err = panicking(nil, &abci.RequestCheckTx{})
	}()
	if err != nil || res == nil || res.Code == 0 {
		t.Fatalf("a panicking handler answered %+v, %v; want a refusal", res, err)
	}

	// The recovered panic is logged at error level with the transaction's
	// hash and size, the check type, the panic and the stack.
	rec := &errorRecorder{}
	txBytes := []byte("a transaction")
	logged := recoveringCheckTxHandler(func(sdk.RunTx, *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
		panic(storetypes.ErrorOutOfGas{Descriptor: "ReadFlat"})
	}, rec)
	if res, err := logged(nil, &abci.RequestCheckTx{Tx: txBytes, Type: abci.CheckTxType_Recheck}); err != nil || res == nil || res.Code == 0 {
		t.Fatalf("a panicking handler answered %+v, %v; want a refusal", res, err)
	}
	if len(rec.errors) != 1 {
		t.Fatalf("want one error-level line, got %d", len(rec.errors))
	}
	kv := rec.errors[0]
	if got, want := kv["tx_hash"], fmt.Sprintf("%X", cmttypes.Tx(txBytes).Hash()); got != want {
		t.Fatalf("tx_hash %v, want %v", got, want)
	}
	if kv["tx_bytes"] != len(txBytes) || kv["check_type"] != "RECHECK" || kv["panic_type"] != "types.ErrorOutOfGas" {
		t.Fatalf("logged %v", kv)
	}
	if p, _ := kv["panic"].(string); !strings.Contains(p, "ReadFlat") {
		t.Fatalf("panic value not logged: %v", kv["panic"])
	}
	if st, _ := kv["stack"].(string); !strings.Contains(st, "recoveringCheckTxHandler") {
		t.Fatalf("stack not logged: %q", st)
	}
	// A request the handler cannot read (nil) is still refused and logged.
	if res, err := logged(nil, nil); err != nil || res == nil || res.Code == 0 {
		t.Fatalf("a panicking handler on a nil request answered %+v, %v; want a refusal", res, err)
	}

	want := &abci.ResponseCheckTx{GasWanted: 7}
	passing := recoveringCheckTxHandler(func(sdk.RunTx, *abci.RequestCheckTx) (*abci.ResponseCheckTx, error) {
		return want, nil
	}, log.NewNopLogger())
	if got, err := passing(nil, &abci.RequestCheckTx{}); got != want || err != nil {
		t.Fatalf("answer changed: %+v, %v", got, err)
	}
}

// errorRecorder keeps the key/value pairs of every error-level line.
type errorRecorder struct{ errors []map[string]any }

func (r *errorRecorder) Error(_ string, keyVals ...any) {
	kv := map[string]any{}
	for i := 0; i+1 < len(keyVals); i += 2 {
		kv[fmt.Sprint(keyVals[i])] = keyVals[i+1]
	}
	r.errors = append(r.errors, kv)
}
func (r *errorRecorder) Info(string, ...any)    {}
func (r *errorRecorder) Warn(string, ...any)    {}
func (r *errorRecorder) Debug(string, ...any)   {}
func (r *errorRecorder) With(...any) log.Logger { return r }
func (r *errorRecorder) Impl() any              { return r }

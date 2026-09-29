package app

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// feelessTxBytes is a MsgSend with a body and NO auth info, the reporter's shape.
func feelessTxBytes(t *testing.T) []byte {
	t.Helper()
	from := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	b := testEncoding.TxConfig.NewTxBuilder()
	if err := b.SetMsgs(banktypes.NewMsgSend(from, from, sdk.NewCoins(sdk.NewInt64Coin("uqor", 1)))); err != nil {
		t.Fatal(err)
	}
	full, err := testEncoding.TxConfig.TxEncoder()(b.GetTx())
	if err != nil {
		t.Fatal(err)
	}
	var raw txtypes.TxRaw
	if err := raw.Unmarshal(full); err != nil {
		t.Fatal(err)
	}
	raw.AuthInfoBytes = nil
	raw.Signatures = nil
	bz, err := raw.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	return bz
}

// The fee-less tx used to panic in SetUpContextDecorator and return a stack trace
// from simulate. On the public paths it is now a plain error and no panic.
func TestFeelessTxIsRefusedCleanlyOnPublicPaths(t *testing.T) {
	setup := sdk.ChainAnteDecorators(ante.NewSetUpContextDecorator())
	var panicked atomic.Bool
	guarded := func(ctx sdk.Context, tx sdk.Tx, sim bool) (sdk.Context, error) {
		if err := refuseFeelessTx(ctx, tx, sim); err != nil {
			return ctx, err
		}
		defer func() {
			if r := recover(); r != nil {
				panicked.Store(true)
				panic(r)
			}
		}()
		return setup(ctx, tx, sim)
	}
	bapp := newGuardedBaseApp(t, guarded)
	bz := feelessTxBytes(t)

	_, _, err := bapp.Simulate(bz)
	if !errors.Is(err, errortypes.ErrInvalidRequest) || !strings.Contains(err.Error(), "tx has no fee") {
		t.Fatalf("simulate: got %v, want a plain 'tx has no fee'", err)
	}
	res, err := bapp.CheckTx(&abci.RequestCheckTx{Tx: bz, Type: abci.CheckTxType_New})
	if err != nil {
		t.Fatal(err)
	}
	if res.Code != errortypes.ErrInvalidRequest.ABCICode() || strings.Contains(res.Log, "stack") {
		t.Fatalf("CheckTx: code %d log %q", res.Code, res.Log)
	}
	if panicked.Load() {
		t.Fatal("the public paths still panicked")
	}

	// Block execution is unchanged: the tx still reaches SetUpContext and fails
	// there with the same code as before.
	fb, err := bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{bz}})
	if err != nil {
		t.Fatal(err)
	}
	if !panicked.Load() {
		t.Fatal("FinalizeBlock no longer reaches SetUpContext; block behaviour changed")
	}
	if got := fb.TxResults[0].Code; got != errortypes.ErrPanic.ABCICode() {
		t.Fatalf("FinalizeBlock code %d, want ErrPanic as before", got)
	}
}

func panickingAnte(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
	var m map[string]int
	m["boom"] = 1 // nil map write: a genuine runtime panic
	return ctx, nil
}

// The recovery handler keeps the code and gas, and drops the stack from the
// response. The control case proves the default really did leak it.
func TestQuietRecoveryHandlerDropsTheStack(t *testing.T) {
	bz := buildSignedTxBytes(t, single())

	control := newGuardedBaseApp(t, panickingAnte)
	_, _, err := control.Simulate(bz)
	if err == nil || !strings.Contains(err.Error(), "stack:") {
		t.Fatalf("control: the SDK default no longer leaks a stack (%v); revisit this handler", err)
	}
	ctrlFB, err := control.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{bz}})
	if err != nil {
		t.Fatal(err)
	}

	// Added after construction, as app.go does after Build: NewBaseApp installs
	// the default middleware after running its options, so an option would be
	// overwritten.
	quiet := newGuardedBaseApp(t, panickingAnte)
	quiet.AddRunTxRecoveryHandler(newQuietPanicRecoveryHandler(log.NewNopLogger()))
	_, _, err = quiet.Simulate(bz)
	if !errors.Is(err, errortypes.ErrPanic) {
		t.Fatalf("got %v, want ErrPanic", err)
	}
	if strings.Contains(err.Error(), "stack") || strings.Contains(err.Error(), ".go:") {
		t.Fatalf("the response still carries a stack: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "assignment to entry in nil map") {
		t.Fatalf("the panic value was lost: %q", err.Error())
	}

	// Deterministic result fields in block execution are identical to the default.
	fb, err := quiet.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{bz}})
	if err != nil {
		t.Fatal(err)
	}
	a, b := ctrlFB.TxResults[0], fb.TxResults[0]
	if a.Code != b.Code || a.Codespace != b.Codespace || a.GasWanted != b.GasWanted ||
		a.GasUsed != b.GasUsed || string(a.Data) != string(b.Data) || len(a.Events) != len(b.Events) {
		t.Fatalf("deterministic fields differ: default %+v, quiet %+v", a, b)
	}
	if string(ctrlFB.AppHash) != string(fb.AppHash) {
		t.Fatal("app hash differs from the default handler's")
	}
}

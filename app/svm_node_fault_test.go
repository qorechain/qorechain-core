package app

import (
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	svmtypes "github.com/qorechain/qorechain-core/x/svm/types"
)

// svm-execution-thread-node-fault: with the recovery handlers in app.go's
// order, an SVM NodeFault raised inside a transaction leaves FinalizeBlock (the
// node halts before committing the block) instead of becoming a failed
// transaction; any other panic is still a failed transaction (ErrPanic).
func TestSVMNodeFaultLeavesTheBlock(t *testing.T) {
	newApp := func(value interface{}) func() {
		bapp := newGuardedBaseApp(t, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
			panic(value)
		})
		bapp.AddRunTxRecoveryHandler(newQuietPanicRecoveryHandler(log.NewNopLogger()))
		bapp.AddRunTxRecoveryHandler(svmtypes.NodeFaultRecoveryHandler)
		bz := buildSignedTxBytes(t, single())
		return func() {
			res, err := bapp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Txs: [][]byte{bz}})
			if err != nil {
				t.Fatal(err)
			}
			if got := res.TxResults[0].Code; got != errortypes.ErrPanic.ABCICode() {
				t.Fatalf("an ordinary panic gave code %d, want ErrPanic", got)
			}
		}
	}

	var escaped interface{}
	func() {
		defer func() { escaped = recover() }()
		newApp(svmtypes.NodeFault{Reason: "test"})()
	}()
	nf, ok := escaped.(svmtypes.NodeFault)
	if !ok {
		t.Fatalf("NodeFault became a transaction result (recovered %v)", escaped)
	}
	if nf.Reason != "test" {
		t.Fatalf("NodeFault lost its reason: %v", nf)
	}

	// Control: any other panic is a failed transaction, as before.
	newApp("ordinary panic")()
}

// A NodeFault raised during the block fails the block even when something in
// between recovered the panic; a block without one is untouched, and a fault
// left over from before the block does not count.
func TestSVMNodeFaultLatchFailsTheBlock(t *testing.T) {
	t.Cleanup(svmtypes.ClearNodeFault)
	ok := &abci.ResponseFinalizeBlock{AppHash: []byte{1}}
	res, err := finalizeWithNodeFaultLatch(log.NewNopLogger(), 5, func() (*abci.ResponseFinalizeBlock, error) {
		func() {
			defer func() { _ = recover() }() // a caller that swallows every panic
			svmtypes.RaiseNodeFault(svmtypes.NodeFault{Reason: "swallowed"})
		}()
		return ok, nil
	})
	if err == nil || res != nil {
		t.Fatalf("a swallowed NodeFault let the block finish: %v %v", res, err)
	}
	if _, isFault := err.(svmtypes.NodeFault); !isFault {
		t.Fatalf("error %T", err)
	}
	// A stale fault is cleared at the start of the block.
	func() {
		defer func() { _ = recover() }()
		svmtypes.RaiseNodeFault(svmtypes.NodeFault{Reason: "stale"})
	}()
	res, err = finalizeWithNodeFaultLatch(log.NewNopLogger(), 6, func() (*abci.ResponseFinalizeBlock, error) { return ok, nil })
	if err != nil || res != ok {
		t.Fatalf("clean block failed: %v", err)
	}
}

package app

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	protov2 "google.golang.org/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type msgsTx struct{ msgs []sdk.Msg }

func (t msgsTx) GetMsgs() []sdk.Msg                    { return t.msgs }
func (t msgsTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

func ethMsg(t *testing.T, signer ethtypes.Signer, inner ethtypes.TxData) *evmtypes.MsgEthereumTx {
	t.Helper()
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	tx, err := ethtypes.SignNewTx(key, signer, inner)
	if err != nil {
		t.Fatal(err)
	}
	msg := &evmtypes.MsgEthereumTx{}
	msg.FromEthereumTx(tx)
	return msg
}

func TestEVMReplayGuard(t *testing.T) {
	chainID := big.NewInt(9801)
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	legacy := &ethtypes.LegacyTx{Nonce: 0, To: &to, Gas: 21000, GasPrice: big.NewInt(1)}

	unprotected := ethMsg(t, ethtypes.HomesteadSigner{}, legacy)
	protected := ethMsg(t, ethtypes.NewEIP155Signer(chainID), legacy)
	dynamic := ethMsg(t, ethtypes.NewLondonSigner(chainID), &ethtypes.DynamicFeeTx{
		ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(1),
	})

	passed := false
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		passed = true
		return ctx, nil
	}
	run := func(ctx sdk.Context, msg sdk.Msg, simulate bool) (bool, error) {
		passed = false
		_, err := NewEVMReplayGuardDecorator().AnteHandle(ctx, msgsTx{[]sdk.Msg{msg}}, simulate, next)
		return passed, err
	}
	checkTx := sdk.Context{}.WithIsCheckTx(true)
	recheckTx := sdk.Context{}.WithIsReCheckTx(true)
	deliverTx := sdk.Context{}

	// The mempool refuses an unprotected transaction, on check and recheck.
	for _, ctx := range []sdk.Context{checkTx, recheckTx} {
		ok, err := run(ctx, unprotected, false)
		if ok || !errortypes.ErrInvalidChainID.Is(err) {
			t.Fatalf("unprotected legacy tx admitted to the mempool (err %v)", err)
		}
	}

	// Block execution is untouched: the guard passes straight through.
	if ok, err := run(deliverTx, unprotected, false); !ok || err != nil {
		t.Fatalf("the guard must not act in block execution (err %v)", err)
	}
	// Simulation is left alone.
	if ok, err := run(checkTx, unprotected, true); !ok || err != nil {
		t.Fatalf("the guard must not act in simulation (err %v)", err)
	}

	// Replay-protected and typed transactions, and non-EVM messages, pass.
	for _, msg := range []sdk.Msg{protected, dynamic, &banktypes.MsgSend{}} {
		if ok, err := run(checkTx, msg, false); !ok || err != nil {
			t.Fatalf("%T admitted wrongly refused: %v", msg, err)
		}
	}
}

package app

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	protov2 "google.golang.org/protobuf/proto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
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
		_, err := NewEVMReplayGuardDecorator(nil).AnteHandle(ctx, msgsTx{[]sdk.Msg{msg}}, simulate, next)
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

// featureSet answers ProtocolFeatureEnabled from a fixed set.
type featureSet map[string]bool

func (f featureSet) ProtocolFeatureEnabled(_ sdk.Context, name string) bool { return f[name] }

// QSR-2026-0052, consensus phase. With FeatureEVMUnprotectedTxRefusal off the
// guard behaves exactly as the mempool-only v3.2.1 guard (history replays
// unchanged: an unprotected transaction included in a block still executes).
// With it on, an unprotected transaction is refused in block execution and
// simulation too, and every replay-protected shape still passes everywhere.
func TestEVMReplayGuardFeatureGate(t *testing.T) {
	chainID := big.NewInt(9801)
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	legacy := &ethtypes.LegacyTx{Nonce: 0, To: &to, Gas: 21000, GasPrice: big.NewInt(1)}

	unprotected := ethMsg(t, ethtypes.HomesteadSigner{}, legacy)
	protected := ethMsg(t, ethtypes.NewEIP155Signer(chainID), legacy)
	accessList := ethMsg(t, ethtypes.NewPragueSigner(chainID), &ethtypes.AccessListTx{
		ChainID: chainID, To: &to, Gas: 21000, GasPrice: big.NewInt(1),
	})
	dynamic := ethMsg(t, ethtypes.NewPragueSigner(chainID), &ethtypes.DynamicFeeTx{
		ChainID: chainID, To: &to, Gas: 21000, GasFeeCap: big.NewInt(1), GasTipCap: big.NewInt(1),
	})
	setCode := ethMsg(t, ethtypes.NewPragueSigner(chainID), &ethtypes.SetCodeTx{
		ChainID: uint256.MustFromBig(chainID), To: to, Gas: 21000,
		GasFeeCap: uint256.NewInt(1), GasTipCap: uint256.NewInt(1),
		AuthList: []ethtypes.SetCodeAuthorization{{ChainID: *uint256.MustFromBig(chainID), Address: to}},
	})

	passed := false
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		passed = true
		return ctx, nil
	}
	run := func(features protocolFeatureReader, ctx sdk.Context, msg sdk.Msg, simulate bool) (bool, error) {
		passed = false
		_, err := NewEVMReplayGuardDecorator(features).AnteHandle(ctx, msgsTx{[]sdk.Msg{msg}}, simulate, next)
		return passed, err
	}
	off := featureSet{}
	on := featureSet{pqctypes.FeatureEVMUnprotectedTxRefusal: true}
	unrelated := featureSet{pqctypes.FeatureEVMPQCAuthorisationWindow: true}

	type mode struct {
		name     string
		ctx      sdk.Context
		simulate bool
	}
	modes := []mode{
		{"checktx", sdk.Context{}.WithIsCheckTx(true), false},
		{"rechecktx", sdk.Context{}.WithIsReCheckTx(true), false},
		{"delivertx", sdk.Context{}, false},
		{"simulate", sdk.Context{}.WithIsCheckTx(true), true},
		{"simulate-deliver", sdk.Context{}, true},
	}

	for _, m := range modes {
		// Off, or only unrelated features on: the v3.2.1 behaviour.
		for _, f := range []protocolFeatureReader{nil, off, unrelated} {
			ok, err := run(f, m.ctx, unprotected, m.simulate)
			mempoolOnly := (m.ctx.IsCheckTx() || m.ctx.IsReCheckTx()) && !m.simulate
			if mempoolOnly {
				if ok || !errortypes.ErrInvalidChainID.Is(err) {
					t.Fatalf("%s off: unprotected tx must be refused by the mempool (err %v)", m.name, err)
				}
				if err.Error() != "only replay-protected (EIP-155) Ethereum transactions are accepted into the mempool: invalid chain-id" {
					t.Fatalf("%s off: mempool error text changed: %q", m.name, err.Error())
				}
			} else if !ok || err != nil {
				t.Fatalf("%s off: the guard must pass through (err %v)", m.name, err)
			}
		}

		// On: refused in every mode.
		if ok, err := run(on, m.ctx, unprotected, m.simulate); ok || !errortypes.ErrInvalidChainID.Is(err) {
			t.Fatalf("%s on: unprotected tx admitted (err %v)", m.name, err)
		}

		// Replay-protected and typed transactions pass in every mode, either way.
		for _, f := range []protocolFeatureReader{nil, off, on} {
			for _, msg := range []sdk.Msg{protected, accessList, dynamic, setCode, &banktypes.MsgSend{}} {
				if ok, err := run(f, m.ctx, msg, m.simulate); !ok || err != nil {
					t.Fatalf("%s: %T wrongly refused: %v", m.name, msg, err)
				}
			}
		}
	}

	// A batch is refused if any of its messages is unprotected.
	passed = false
	_, err := NewEVMReplayGuardDecorator(on).AnteHandle(sdk.Context{}, msgsTx{[]sdk.Msg{protected, unprotected}}, false, next)
	if passed || !errortypes.ErrInvalidChainID.Is(err) {
		t.Fatalf("batch with an unprotected tx admitted in block execution (err %v)", err)
	}
}

package app

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	evmencoding "github.com/cosmos/evm/encoding"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// fakeComet records broadcasts; release gates them so a test can hold the
// worker busy.
type fakeComet struct {
	client.CometRPC
	mu      sync.Mutex
	got     []common.Hash
	release chan struct{}
}

func (f *fakeComet) BroadcastTxSync(_ context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	if f.release != nil {
		<-f.release
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, common.BytesToHash(tx))
	return &coretypes.ResultBroadcastTx{}, nil
}

func (f *fakeComet) sent() []common.Hash {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]common.Hash(nil), f.got...)
}

func newTestBroadcaster() *evmTxBroadcaster {
	b := newEVMTxBroadcaster(log.NewNopLogger(), nil)
	b.encode = func(tx *ethtypes.Transaction) ([]byte, error) { return tx.Hash().Bytes(), nil }
	return b
}

func legacyTx(nonce uint64) *ethtypes.Transaction {
	return ethtypes.NewTx(&ethtypes.LegacyTx{Nonce: nonce})
}

// QSR-2026-0028: promoted transactions are broadcast through the client
// context the node registers after it starts, not the one captured when the
// mempool was built.
func TestEVMBroadcasterUsesTheLateClient(t *testing.T) {
	b := newTestBroadcaster()

	// Before the node registers its client there is nowhere to send: the
	// transaction is skipped (it stays in the pool) without an error.
	b.broadcast(legacyTx(0))

	comet := &fakeComet{}
	b.setClientCtx(client.Context{}.WithClient(comet))
	txs := []*ethtypes.Transaction{legacyTx(1), legacyTx(2)}
	if err := b.enqueue(txs); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(comet.sent()) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	got := comet.sent()
	if len(got) != 2 || got[0] != txs[0].Hash() || got[1] != txs[1].Hash() {
		t.Fatalf("broadcast %v, want the two promoted transactions in order", got)
	}

	// A context without a client does not replace the registered one.
	b.setClientCtx(client.Context{})
	if b.clientCtx.Load().Client != comet {
		t.Fatal("a client-less context replaced the node's client")
	}
}

// The mempool calls the broadcaster from its reorganisation goroutine, and a
// broadcast re-enters the mempool through CheckTx, so the call must return at
// once however slow the broadcast is.
func TestEVMBroadcasterNeverBlocksThePool(t *testing.T) {
	b := newTestBroadcaster()
	comet := &fakeComet{release: make(chan struct{})}
	defer close(comet.release)
	b.setClientCtx(client.Context{}.WithClient(comet))

	txs := make([]*ethtypes.Transaction, evmBroadcastQueueSize+10)
	for i := range txs {
		txs[i] = legacyTx(uint64(i))
	}
	done := make(chan error, 1)
	go func() { done <- b.enqueue(txs) }()
	select {
	case err := <-done:
		// The worker holds one and the queue the next evmBroadcastQueueSize;
		// the rest are reported, not waited for.
		if err == nil {
			t.Fatal("an overflowing batch must be reported")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("enqueue blocked on a busy broadcaster")
	}
}

// The bytes are an Ethereum transaction as eth_sendRawTransaction builds it:
// the ante handler routes on the extension option and refuses a bare
// MsgEthereumTx.
func TestEncodeEVMTxCarriesTheEthereumExtension(t *testing.T) {
	_ = evmtypes.NewEVMConfigurator().WithEVMCoinInfo(evmtypes.EvmCoinInfo{
		Denom: BaseDenom, ExtendedDenom: ExtendedDenom, DisplayDenom: DisplayDenom, Decimals: uint32(evmtypes.SixDecimals),
	}).Configure() // may already be configured in this test binary
	encCfg := evmencoding.MakeConfig(EVMChainIDTestnet)
	evmtypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	chainID := new(big.Int).SetUint64(EVMChainIDTestnet)
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	tx, err := ethtypes.SignNewTx(key, ethtypes.LatestSignerForChainID(chainID), &ethtypes.DynamicFeeTx{
		ChainID: chainID, Nonce: 3, To: &to, Gas: 21000, GasFeeCap: big.NewInt(10), GasTipCap: big.NewInt(1),
	})
	if err != nil {
		t.Fatal(err)
	}

	bz, err := encodeEVMTx(encCfg.TxConfig, tx)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := encCfg.TxConfig.TxDecoder()(bz)
	if err != nil {
		t.Fatal(err)
	}
	opts := decoded.(authante.HasExtensionOptionsTx).GetExtensionOptions()
	if len(opts) != 1 || opts[0].GetTypeUrl() != "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx" {
		t.Fatalf("extension options %v", opts)
	}
	msg := decoded.GetMsgs()[0].(*evmtypes.MsgEthereumTx)
	if msg.AsTransaction().Hash() != tx.Hash() {
		t.Fatal("the wrapped transaction differs")
	}
	if common.BytesToAddress(msg.From) != crypto.PubkeyToAddress(key.PublicKey) {
		t.Fatal("the sender is not set")
	}
}

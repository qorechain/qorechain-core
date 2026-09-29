package app

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// evmBroadcastQueueSize bounds the promoted transactions waiting to be
// broadcast. The pool promotes at most its pending capacity at once.
const evmBroadcastQueueSize = 4096

// evmTxBroadcaster puts the EVM transactions the mempool promotes (a queued
// transaction whose nonce gap was filled, or a same-nonce replacement) into
// the node's consensus mempool, which gossips them. Until it existed they
// never left the node that received them (QSR-2026-0028).
//
// The mempool is built with the application, before the node exists. Its
// default broadcaster captured the client context of that moment, which has
// no node client, so every promotion broadcast failed. The context with the
// in-process client reaches the application later, through
// RegisterTxService; the broadcaster reads it at the time of each broadcast.
//
// The mempool calls the broadcaster from its reorganisation goroutine, and a
// broadcast goes through CheckTx, which inserts into that same pool. So the
// call only queues the transactions and a worker broadcasts them.
type evmTxBroadcaster struct {
	logger log.Logger
	encode func(*ethtypes.Transaction) ([]byte, error)

	clientCtx atomic.Pointer[client.Context]
	queue     chan *ethtypes.Transaction
	start     sync.Once
}

func newEVMTxBroadcaster(logger log.Logger, txConfig client.TxConfig) *evmTxBroadcaster {
	return &evmTxBroadcaster{
		logger: logger.With("module", "evm-broadcast"),
		encode: func(tx *ethtypes.Transaction) ([]byte, error) { return encodeEVMTx(txConfig, tx) },
		queue:  make(chan *ethtypes.Transaction, evmBroadcastQueueSize),
	}
}

// setClientCtx records the client context that carries the in-process node
// client.
func (b *evmTxBroadcaster) setClientCtx(clientCtx client.Context) {
	if clientCtx.Client == nil {
		return
	}
	b.clientCtx.Store(&clientCtx)
}

// enqueue is the mempool's broadcast function. It never blocks: when the
// queue is full the rest are dropped and reported, and they stay in the pool,
// where this node can still include them in a block it proposes.
func (b *evmTxBroadcaster) enqueue(txs []*ethtypes.Transaction) error {
	b.start.Do(func() { go b.run() })
	dropped := 0
	for _, tx := range txs {
		select {
		case b.queue <- tx:
		default:
			dropped++
		}
	}
	if dropped > 0 {
		return fmt.Errorf("evm broadcast queue full: %d promoted transactions not broadcast", dropped)
	}
	return nil
}

func (b *evmTxBroadcaster) run() {
	for tx := range b.queue {
		b.broadcast(tx)
	}
}

func (b *evmTxBroadcaster) broadcast(tx *ethtypes.Transaction) {
	clientCtx := b.clientCtx.Load()
	if clientCtx == nil {
		b.logger.Debug("no local node client yet; promoted transaction not broadcast", "hash", tx.Hash().Hex())
		return
	}
	txBytes, err := b.encode(tx)
	if err != nil {
		b.logger.Error("failed to encode promoted transaction", "hash", tx.Hash().Hex(), "error", err)
		return
	}
	res, err := clientCtx.BroadcastTxSync(txBytes)
	switch {
	case err != nil:
		if !alreadyKnown(err.Error()) {
			b.logger.Error("failed to broadcast promoted transaction", "hash", tx.Hash().Hex(), "error", err)
		}
	case res.Code != 0 && !alreadyKnownResponse(res):
		b.logger.Debug("promoted transaction refused by CheckTx", "hash", tx.Hash().Hex(), "code", res.Code, "log", res.RawLog)
	}
}

// encodeEVMTx wraps a signed Ethereum transaction the way
// eth_sendRawTransaction does: sender set, fee and the Ethereum extension
// option attached. A bare MsgEthereumTx is refused by the ante handler.
func encodeEVMTx(txConfig client.TxConfig, tx *ethtypes.Transaction) ([]byte, error) {
	msg := &evmtypes.MsgEthereumTx{}
	if err := msg.FromSignedEthereumTx(tx, ethtypes.LatestSignerForChainID(tx.ChainId())); err != nil {
		return nil, fmt.Errorf("recover sender: %w", err)
	}
	cosmosTx, err := msg.BuildTx(txConfig.NewTxBuilder(), evmtypes.GetEVMCoinDenom())
	if err != nil {
		return nil, fmt.Errorf("build cosmos tx: %w", err)
	}
	return txConfig.TxEncoder()(cosmosTx)
}

// alreadyKnown reports whether a broadcast failed only because the node holds
// the transaction already, in the consensus mempool cache or in the EVM pool.
func alreadyKnown(msg string) bool {
	return strings.Contains(msg, "tx already exists in cache") || strings.Contains(msg, "already known")
}

// alreadyKnownResponse is alreadyKnown for a CheckTx answer. The client maps
// a mempool cache hit to ErrTxInMempoolCache with an empty log.
func alreadyKnownResponse(res *sdk.TxResponse) bool {
	if res.Codespace == sdkerrors.ErrTxInMempoolCache.Codespace() && res.Code == sdkerrors.ErrTxInMempoolCache.ABCICode() {
		return true
	}
	return alreadyKnown(res.RawLog)
}

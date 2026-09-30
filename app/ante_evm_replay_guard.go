package app

import (
	errorsmod "cosmossdk.io/errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// EVMReplayGuardDecorator refuses to admit into this node's mempool a legacy
// Ethereum transaction signed without a chain ID (pre-EIP-155, v = 27/28).
//
// Such a signature is valid on every EVM chain, so a transaction signed for
// another chain, or for the testnet, can be replayed here once the nonce
// matches. eth_sendRawTransaction already refuses it (json-rpc
// allow-unprotected-txs is off), but the same bytes wrapped as a Cosmos
// transaction reached the EVM lane of the ante handler, where nothing checked
// replay protection (QSR-2026-0052).
//
// It has two phases.
//
// Below the v3.2.2 plan height it runs in CheckTx only, like
// WasmMempoolGuardDecorator: mempool policy, holding for blocks proposed by
// nodes that run it, while block execution and simulation are untouched so
// earlier blocks replay unchanged.
//
// From the plan height (FeatureEVMUnprotectedTxRefusal) it refuses such a
// transaction everywhere, in block execution and simulation too, so a proposer
// can no longer include one.
type EVMReplayGuardDecorator struct {
	features protocolFeatureReader
}

// protocolFeatureReader is the part of the PQC keeper the guard needs: a
// gas-free read of the protocol-feature record.
type protocolFeatureReader interface {
	ProtocolFeatureEnabled(ctx sdk.Context, name string) bool
}

// NewEVMReplayGuardDecorator returns the guard. features may be nil, which
// keeps the mempool-only phase.
func NewEVMReplayGuardDecorator(features protocolFeatureReader) EVMReplayGuardDecorator {
	return EVMReplayGuardDecorator{features: features}
}

func (d EVMReplayGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	if d.features != nil && d.features.ProtocolFeatureEnabled(ctx, pqctypes.FeatureEVMUnprotectedTxRefusal) {
		if err := rejectUnprotectedEthTxs(tx.GetMsgs(), "accepted"); err != nil {
			return ctx, err
		}
		return next(ctx, tx, simulate)
	}
	// Only guard the mempool. Simulation is left alone, as for the wasm guard.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	if err := rejectUnprotectedEthTxs(tx.GetMsgs(), "accepted into the mempool"); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// rejectUnprotectedEthTxs returns an error for the first legacy Ethereum
// transaction without replay protection. Typed transactions (EIP-2930, 1559,
// 7702) always carry a chain ID.
func rejectUnprotectedEthTxs(msgs []sdk.Msg, where string) error {
	for _, msg := range msgs {
		ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			continue
		}
		ethTx := ethMsg.AsTransaction()
		if ethTx != nil && ethTx.Type() == ethtypes.LegacyTxType && !ethTx.Protected() {
			return errorsmod.Wrap(errortypes.ErrInvalidChainID,
				"only replay-protected (EIP-155) Ethereum transactions are "+where)
		}
	}
	return nil
}

package app

import (
	errorsmod "cosmossdk.io/errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	evmtypes "github.com/cosmos/evm/x/vm/types"
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
// Like WasmMempoolGuardDecorator it runs in CheckTx only, so block validity is
// unchanged and it rolls out one node at a time. It is mempool policy: it holds
// for blocks proposed by nodes that run it. Refusing such a transaction inside
// a block needs a protocol feature switched on at an upgrade height, so that
// earlier blocks replay unchanged.
type EVMReplayGuardDecorator struct{}

func NewEVMReplayGuardDecorator() EVMReplayGuardDecorator {
	return EVMReplayGuardDecorator{}
}

func (d EVMReplayGuardDecorator) AnteHandle(
	ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler,
) (sdk.Context, error) {
	// Only guard the mempool. Simulation is left alone, as for the wasm guard.
	if !ctx.IsCheckTx() || simulate {
		return next(ctx, tx, simulate)
	}
	if err := rejectUnprotectedEthTxs(tx.GetMsgs()); err != nil {
		return ctx, err
	}
	return next(ctx, tx, simulate)
}

// rejectUnprotectedEthTxs returns an error for the first legacy Ethereum
// transaction without replay protection. Typed transactions (EIP-2930, 1559,
// 7702) always carry a chain ID.
func rejectUnprotectedEthTxs(msgs []sdk.Msg) error {
	for _, msg := range msgs {
		ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			continue
		}
		ethTx := ethMsg.AsTransaction()
		if ethTx != nil && ethTx.Type() == ethtypes.LegacyTxType && !ethTx.Protected() {
			return errorsmod.Wrap(errortypes.ErrInvalidChainID,
				"only replay-protected (EIP-155) Ethereum transactions are accepted into the mempool")
		}
	}
	return nil
}

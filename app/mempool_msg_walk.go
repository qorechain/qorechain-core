package app

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/cosmos/cosmos-sdk/x/group"
)

// mempoolMsgWalkMaxDepth bounds how deep the mempool guards look into
// wrappers. Nesting beyond a few levels is not legitimate traffic, and a
// transaction that cannot be inspected is not admitted.
const mempoolMsgWalkMaxDepth = 4

// walkMempoolMsgs calls visit on every message of a transaction and on every
// message carried inside a wrapper that a non-governance account can use to
// have messages executed on its behalf:
//
//   - authz MsgExec (executed with the granter as signer);
//   - x/group MsgSubmitProposal (executed with the group policy as signer,
//     at once with Exec TRY or later through MsgExec of the proposal).
//
// Governance proposals are deliberately not entered: what they carry runs
// only after a vote, under the governance authority, and storing CosmWasm
// code is meant to go exactly that way.
//
// It is the one walker for the CheckTx-only mempool guards (wasm upload,
// cross-VM payload size), so a wrapper added here is covered by all of them.
// intoGroupProposals selects whether group proposals are entered: a guard
// that refuses a message as policy (wasm upload) enters them; a guard that
// only spares a fee for a certain failure (cross-VM payload size) does not,
// because a group proposal that fails to execute does not fail the
// transaction that submitted it.
// A payload that will not unpack, or nesting beyond mempoolMsgWalkMaxDepth,
// is refused: it cannot be shown to be safe. what names the guard in that
// error.
//
// Mempool policy only: never call it from block execution, where looking into
// more wrappers would change which transactions a block accepts.
func walkMempoolMsgs(msgs []sdk.Msg, what string, intoGroupProposals bool, visit func(sdk.Msg) error) error {
	return walkMempoolMsgsAt(msgs, 0, what, intoGroupProposals, visit)
}

func walkMempoolMsgsAt(msgs []sdk.Msg, depth int, what string, intoGroupProposals bool, visit func(sdk.Msg) error) error {
	if depth > mempoolMsgWalkMaxDepth {
		return errorsmod.Wrapf(errortypes.ErrUnauthorized,
			"message nesting too deep to inspect for %s", what)
	}
	for _, msg := range msgs {
		if err := visit(msg); err != nil {
			return err
		}
		var inner []sdk.Msg
		var err error
		switch m := msg.(type) {
		case *authz.MsgExec:
			inner, err = m.GetMessages()
			if err != nil {
				return errorsmod.Wrap(errortypes.ErrUnauthorized,
					"authz payload could not be unpacked for inspection")
			}
		case *group.MsgSubmitProposal:
			if !intoGroupProposals {
				continue
			}
			inner, err = m.GetMsgs()
			if err != nil {
				return errorsmod.Wrap(errortypes.ErrUnauthorized,
					"group proposal payload could not be unpacked for inspection")
			}
		default:
			continue
		}
		if err := walkMempoolMsgsAt(inner, depth+1, what, intoGroupProposals, visit); err != nil {
			return err
		}
	}
	return nil
}

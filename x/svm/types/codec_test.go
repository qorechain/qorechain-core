package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/svm/types"
)

// Every message the module handles must resolve through the interface registry, or
// the transaction decoder refuses it with "unable to resolve type URL" before any
// handler runs. Handler tests construct messages in Go and never exercise the
// decoder, so an unregistered message passes all of them and fails on the first real
// transaction. MsgUpdateParams shipped exactly that way and was caught in review.
func TestEveryHandledMessageIsRegistered(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	for _, msg := range []sdk.Msg{
		&types.MsgDeployProgram{},
		&types.MsgCreateAccount{},
		&types.MsgExecuteProgram{},
		&types.MsgUpdateParams{},
	} {
		url := sdk.MsgTypeURL(msg)
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		if err != nil {
			t.Fatalf("%s: could not pack: %v", url, err)
		}
		var out sdk.Msg
		if err := registry.UnpackAny(anyMsg, &out); err != nil {
			t.Fatalf("%s is handled but not registered; a transaction carrying it is "+
				"refused at decode: %v", url, err)
		}
	}
}

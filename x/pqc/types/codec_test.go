package types_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/pqc/types"
)

// Every message the module handles must be resolvable through the interface
// registry, or the transaction decoder rejects it with "unable to resolve type URL"
// before any handler runs.
//
// This is the class of bug unit tests do not catch on their own: the handler tests
// construct the message in Go and call the handler directly, so a message that was
// never registered passes every one of them and then fails on the first real
// transaction. The EVM window messages shipped exactly that way and were caught in
// review, not by a test. This test is what would have caught them.
func TestEveryHandledMessageIsRegistered(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)

	handled := []sdk.Msg{
		&types.MsgRegisterPQCKey{},
		&types.MsgRegisterPQCKeyV2{},
		&types.MsgMigratePQCKey{},
		&types.MsgRotatePQCKey{},
		&types.MsgAddAlgorithm{},
		&types.MsgDeprecateAlgorithm{},
		&types.MsgDisableAlgorithm{},
		&types.MsgOpenEVMWindow{},
		&types.MsgCloseEVMWindow{},
	}

	for _, msg := range handled {
		url := sdk.MsgTypeURL(msg)
		// Resolving the URL to a concrete type is exactly what the decoder does
		// when it unpacks a transaction body.
		var out sdk.Msg
		anyMsg, err := codectypes.NewAnyWithValue(msg)
		if err != nil {
			t.Fatalf("%s: could not pack: %v", url, err)
		}
		if err := registry.UnpackAny(anyMsg, &out); err != nil {
			t.Fatalf("%s is handled by the module but not registered in the interface "+
				"registry; a transaction carrying it would be refused at decode with "+
				"\"unable to resolve type URL\": %v", url, err)
		}
	}
}

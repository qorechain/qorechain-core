package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// A genesis call whose sender is not a bech32 account (pending or not) is
// refused, so no imported call (and no escrow it may hold) can be indexed
// or refunded under a sender that does not parse.
func TestGenesisRefusesCallWithoutValidSender(t *testing.T) {
	ok := CrossVMMessage{ID: "a1", SourceVM: VMTypeEVM, TargetVM: VMTypeSVM,
		Sender: sdk.AccAddress([]byte("an-ordinary-account0")).String(), TargetContract: "program",
		Payload: []byte{1}, Status: StatusPending}
	gs := DefaultGenesisState()
	gs.Messages = []CrossVMMessage{ok}
	if err := gs.Validate(); err != nil {
		t.Fatalf("valid call refused: %v", err)
	}
	for _, sender := range []string{"", "not-bech32", "cosmos1/x", "0x00000000000000000000000000000000c0a7e101"} {
		bad := ok
		bad.Sender = sender
		gs.Messages = []CrossVMMessage{bad}
		if err := gs.Validate(); err == nil {
			t.Fatalf("sender %q accepted", sender)
		}
	}
}

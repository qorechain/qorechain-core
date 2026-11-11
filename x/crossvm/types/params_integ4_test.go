package types

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The queue may not be larger than one block's scan of the queue index
// (MaxQueueScanPerBlock): Params.Validate holds it for genesis, for
// SetParams and for governance updates alike.
func TestMaxQueueSizeWithinOneBlockScan(t *testing.T) {
	p := DefaultParams()
	p.MaxQueueSize = uint32(MaxQueueScanPerBlock)
	if err := p.Validate(); err != nil {
		t.Fatalf("a queue of %d: %v", MaxQueueScanPerBlock, err)
	}
	if err := p.ValidateUpdate(); err != nil {
		t.Fatalf("update to a queue of %d: %v", MaxQueueScanPerBlock, err)
	}
	p.MaxQueueSize++
	for name, err := range map[string]error{
		"Validate":       p.Validate(),
		"ValidateUpdate": p.ValidateUpdate(),
		"genesis":        GenesisState{Params: p}.Validate(),
	} {
		if err == nil || !strings.Contains(err.Error(), "one block's queue walk scans") {
			t.Fatalf("%s admitted a queue of %d: %v", name, p.MaxQueueSize, err)
		}
	}
}

// A genesis listing one message ID twice is refused.
func TestGenesisRefusesDuplicateMessageIDs(t *testing.T) {
	m := CrossVMMessage{ID: "a", SourceVM: VMTypeEVM, TargetVM: VMTypeSVM,
		Sender: sdk.AccAddress([]byte("integ4-genesis-sender")).String(), TargetContract: "p", Payload: []byte{1}, Status: StatusPending}
	gs := GenesisState{Params: DefaultParams(), Messages: []CrossVMMessage{m}}
	if err := gs.Validate(); err != nil {
		t.Fatal(err)
	}
	done := m
	done.Status = StatusExecuted
	gs.Messages = append(gs.Messages, done)
	if err := gs.Validate(); err == nil || !strings.Contains(err.Error(), "listed twice") {
		t.Fatalf("duplicate ID admitted: %v", err)
	}
}

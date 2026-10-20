package types_test

import (
	"encoding/json"
	"testing"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// QSR-2026-0379: the CLI builds MsgCrossVMCall without a source lane, because the
// chain derives it. The placeholder it now fills in must pass ValidateBasic for
// every target lane, so the documented flow is no longer refused client side and
// in ante.
func TestPlaceholderSourceVMPassesValidateBasic(t *testing.T) {
	sender := validQorAddress(t)
	for _, target := range []types.VMType{types.VMTypeEVM, types.VMTypeCosmWasm, types.VMTypeSVM} {
		msg := types.MsgCrossVMCall{
			Sender:         sender,
			SourceVM:       types.PlaceholderSourceVM(target),
			TargetVM:       target,
			TargetContract: "target",
			Payload:        []byte{1},
		}
		if err := msg.ValidateBasic(); err != nil {
			t.Fatalf("target %s: a CLI-built message is still refused: %v", target, err)
		}
		// Without the placeholder the old rule still applies: ValidateBasic is
		// unchanged, so no earlier block's verdict changes.
		msg.SourceVM = ""
		if err := msg.ValidateBasic(); err == nil {
			t.Fatalf("target %s: ValidateBasic was relaxed, which changes historical acceptance", target)
		}
	}
}

// QSR-2026-0395: the bridge allowlist matches exact "<vm>:<contract>" entries,
// and an empty list allows nothing.
func TestBridgeTargetAllowed(t *testing.T) {
	p := types.DefaultParams()
	if p.BridgeTargetAllowed(types.VMTypeCosmWasm, "qor1abc") {
		t.Fatal("the default params allow a bridge target")
	}
	p.BridgeAllowedTargets = []string{"cosmwasm:qor1abc"}
	if err := p.Validate(); err != nil {
		t.Fatalf("valid allowlist refused: %v", err)
	}
	if !p.BridgeTargetAllowed(types.VMTypeCosmWasm, "qor1abc") {
		t.Fatal("an allowlisted target was refused")
	}
	if p.BridgeTargetAllowed(types.VMTypeEVM, "qor1abc") || p.BridgeTargetAllowed(types.VMTypeCosmWasm, "qor1abd") {
		t.Fatal("a target outside the allowlist was allowed")
	}
	if p.BridgeTargetAllowed(types.VMTypeCosmWasm, "") {
		t.Fatal("an empty contract was allowed")
	}

	for _, bad := range []string{"qor1abc", "native:qor1abc", "cosmwasm:", "bogus:x"} {
		p.BridgeAllowedTargets = []string{bad}
		if err := p.Validate(); err == nil {
			t.Fatalf("malformed allowlist entry %q accepted", bad)
		}
	}
}

// Existing chains store and export params without the new field: the JSON of the
// default params must not change.
func TestParamsJSONUnchangedWithoutAllowlist(t *testing.T) {
	bz, err := json.Marshal(types.DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"max_message_size":65536,"max_queue_size":1000,"queue_timeout_blocks":100,"enabled":true}`
	if string(bz) != want {
		t.Fatalf("params JSON changed:\n got %s\nwant %s", bz, want)
	}
}

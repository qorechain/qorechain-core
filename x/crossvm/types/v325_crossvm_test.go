package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

func qorContract(t *testing.T, seed byte) string {
	t.Helper()
	initBech32()
	b := make([]byte, 32)
	b[0] = seed
	addr, err := sdk.Bech32ifyAddressBytes("qor", b)
	if err != nil {
		t.Fatal(err)
	}
	return addr
}

// MsgUpdateParams installs every field it carries, and a governance update is
// held to bounds that genesis and stored params are not.
func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	auth := validQorAddress(t)
	good := func() *types.MsgUpdateParams {
		p := types.DefaultParams()
		return &types.MsgUpdateParams{
			Authority: auth, MaxMessageSize: p.MaxMessageSize, MaxQueueSize: p.MaxQueueSize,
			QueueTimeoutBlocks: p.QueueTimeoutBlocks, Enabled: true,
			BridgeAllowedTargets: []string{"cosmwasm:" + qorContract(t, 1)},
		}
	}
	if err := good().ValidateBasic(); err != nil {
		t.Fatalf("a valid update refused: %v", err)
	}
	if p := good().Params(); !p.Enabled || len(p.BridgeAllowedTargets) != 1 || p.MaxQueueSize != types.DefaultMaxQueueSize {
		t.Fatalf("Params() does not carry the message: %+v", p)
	}

	many := make([]string, types.MaxBridgeAllowedTargets+1)
	for i := range many {
		many[i] = "cosmwasm:" + qorContract(t, byte(i+1))
	}
	for name, mutate := range map[string]func(*types.MsgUpdateParams){
		"bad authority":   func(m *types.MsgUpdateParams) { m.Authority = "nope" },
		"zero size":       func(m *types.MsgUpdateParams) { m.MaxMessageSize = 0 },
		"size ceiling":    func(m *types.MsgUpdateParams) { m.MaxMessageSize = types.MaxUpdateMessageSize + 1 },
		"queue ceiling":   func(m *types.MsgUpdateParams) { m.MaxQueueSize = types.MaxUpdateQueueSize + 1 },
		"timeout ceiling": func(m *types.MsgUpdateParams) { m.QueueTimeoutBlocks = types.MaxUpdateQueueTimeout + 1 },
		"evm target":      func(m *types.MsgUpdateParams) { m.BridgeAllowedTargets = []string{"evm:0xabc"} },
		"svm target":      func(m *types.MsgUpdateParams) { m.BridgeAllowedTargets = []string{"svm:program"} },
		"bad address":     func(m *types.MsgUpdateParams) { m.BridgeAllowedTargets = []string{"cosmwasm:qor1abc"} },
		"duplicate": func(m *types.MsgUpdateParams) {
			m.BridgeAllowedTargets = append(m.BridgeAllowedTargets, m.BridgeAllowedTargets[0])
		},
		"too many": func(m *types.MsgUpdateParams) { m.BridgeAllowedTargets = many },
		"too long": func(m *types.MsgUpdateParams) {
			m.BridgeAllowedTargets = []string{"cosmwasm:" + strings.Repeat("q", 300)}
		},
	} {
		m := good()
		mutate(m)
		if err := m.ValidateBasic(); err == nil {
			t.Fatalf("%s: accepted", name)
		}
	}

	// Validate, which genesis and SetParams use, is unchanged: an entry for
	// another lane still passes it.
	p := types.DefaultParams()
	p.BridgeAllowedTargets = []string{"evm:0xabc"}
	if err := p.Validate(); err != nil {
		t.Fatalf("Validate was tightened: %v", err)
	}
}

package types_test

import (
	"sync"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

var setBech32Once sync.Once

func initBech32() {
	setBech32Once.Do(func() {
		config := sdk.GetConfig()
		config.SetBech32PrefixForAccount("qor", "qorpub")
		config.SetBech32PrefixForValidator("qorvaloper", "qorvaloperpub")
		config.SetBech32PrefixForConsensusNode("qorvalcons", "qorvalconspub")
	})
}

func validQorAddress(t *testing.T) string {
	t.Helper()
	initBech32()
	addr := make([]byte, 20) // zero-filled 20 bytes
	bech32Addr, err := sdk.Bech32ifyAddressBytes("qor", addr)
	if err != nil {
		t.Fatalf("failed to create test address: %v", err)
	}
	return bech32Addr
}

func TestMsgCrossVMCallValidateBasic(t *testing.T) {
	validAddr := validQorAddress(t)

	tests := []struct {
		name    string
		msg     types.MsgCrossVMCall
		wantErr bool
	}{
		{
			name: "valid message",
			msg: types.MsgCrossVMCall{
				Sender:         validAddr,
				SourceVM:       types.VMTypeCosmWasm,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0xdeadbeef",
				Payload:        []byte("test"),
			},
			wantErr: false,
		},
		{
			name: "invalid sender",
			msg: types.MsgCrossVMCall{
				Sender:         "invalid",
				SourceVM:       types.VMTypeCosmWasm,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0xdeadbeef",
				Payload:        []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "same VM",
			msg: types.MsgCrossVMCall{
				Sender:         validAddr,
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0xdeadbeef",
				Payload:        []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "empty payload",
			msg: types.MsgCrossVMCall{
				Sender:         validAddr,
				SourceVM:       types.VMTypeCosmWasm,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0xdeadbeef",
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMsgProcessQueueValidateBasic(t *testing.T) {
	validAddr := validQorAddress(t)

	tests := []struct {
		name    string
		msg     types.MsgProcessQueue
		wantErr bool
	}{
		{
			name:    "valid authority",
			msg:     types.MsgProcessQueue{Authority: validAddr},
			wantErr: false,
		},
		{
			name:    "invalid authority",
			msg:     types.MsgProcessQueue{Authority: "invalid"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

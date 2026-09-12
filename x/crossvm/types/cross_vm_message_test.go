package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

// A real, checksum-valid account address. The sender is the identity the target VM
// executes the call as, so these cases have to carry an address that actually parses;
// the placeholders they used before ("0x1234", "qor1sender") were the very confusion
// the sender check exists to catch, and one of them was a lane-native EVM string in
// the field every target parses with the canonical codec.
var testSender = sdk.AccAddress([]byte("crossvm-test-sender0")).String()

func TestCrossVMMessageValidation(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.CrossVMMessage
		wantErr bool
	}{
		{
			name: "valid EVM to CosmWasm message",
			msg: types.CrossVMMessage{
				ID:             "test-id-1",
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeCosmWasm,
				Sender:         testSender,
				TargetContract: "qor1abc123",
				Payload:        []byte(`{"execute":{}}`),
			},
			wantErr: false,
		},
		{
			name: "valid CosmWasm to EVM message",
			msg: types.CrossVMMessage{
				ID:             "test-id-2",
				SourceVM:       types.VMTypeCosmWasm,
				TargetVM:       types.VMTypeEVM,
				Sender:         testSender,
				TargetContract: "0xdeadbeef",
				Payload:        []byte{0x01, 0x02},
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			msg: types.CrossVMMessage{
				Sender:         testSender,
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
				Payload:        []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "same source and target VM",
			msg: types.CrossVMMessage{
				ID:             "test-id-3",
				Sender:         testSender,
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0x1234",
				Payload:        []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "empty target contract",
			msg: types.CrossVMMessage{
				ID:       "test-id-4",
				Sender:   testSender,
				SourceVM: types.VMTypeEVM,
				TargetVM: types.VMTypeCosmWasm,
				Payload:  []byte("test"),
			},
			wantErr: true,
		},
		{
			name: "empty payload",
			msg: types.CrossVMMessage{
				ID:             "test-id-5",
				Sender:         testSender,
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
			},
			wantErr: true,
		},
		{
			name: "invalid source VM",
			msg: types.CrossVMMessage{
				ID:             "test-id-6",
				Sender:         testSender,
				SourceVM:       "invalid",
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
				Payload:        []byte("test"),
			},
			wantErr: true,
		},
		{
			// native is the honest origin for an account calling out directly from a
			// transaction, and it has to be accepted as a source.
			name: "valid native to EVM message",
			msg: types.CrossVMMessage{
				ID:             "test-id-native",
				Sender:         testSender,
				SourceVM:       types.VMTypeNative,
				TargetVM:       types.VMTypeEVM,
				TargetContract: "0xdeadbeef",
				Payload:        []byte{0x01},
			},
			wantErr: false,
		},
		{
			// There is no native runtime to route a call INTO, so it is a source only.
			name: "native as target is refused",
			msg: types.CrossVMMessage{
				ID:             "test-id-native-target",
				Sender:         testSender,
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeNative,
				TargetContract: "qor1abc",
				Payload:        []byte{0x01},
			},
			wantErr: true,
		},
		{
			name: "empty sender",
			msg: types.CrossVMMessage{
				ID:             "test-id-nosender",
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
				Payload:        []byte{0x01},
			},
			wantErr: true,
		},
		{
			// A lane-native address in the sender field: every target parses this with
			// the canonical codec, so it must be refused at the boundary rather than
			// failing later inside the target lane.
			name: "lane-native sender is refused",
			msg: types.CrossVMMessage{
				ID:             "test-id-hexsender",
				Sender:         "0x71C7656EC7ab88b098defB751B7401B5f6d8976F",
				SourceVM:       types.VMTypeEVM,
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
				Payload:        []byte{0x01},
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestCrossVMMessageMarshalUnmarshal(t *testing.T) {
	msg := types.CrossVMMessage{
		ID:             "test-marshal-1",
		SourceVM:       types.VMTypeEVM,
		TargetVM:       types.VMTypeCosmWasm,
		Sender:         testSender,
		SourceContract: "0xsource",
		TargetContract: "qor1target",
		Payload:        []byte(`{"method":"transfer"}`),
		Status:         types.StatusPending,
		CreatedHeight:  100,
	}

	bz, err := types.MarshalCrossVMMessage(&msg)
	if err != nil {
		t.Fatalf("MarshalCrossVMMessage() error = %v", err)
	}

	restored, err := types.UnmarshalCrossVMMessage(bz)
	if err != nil {
		t.Fatalf("UnmarshalCrossVMMessage() error = %v", err)
	}

	if restored.ID != msg.ID {
		t.Errorf("ID mismatch: got %s, want %s", restored.ID, msg.ID)
	}
	if restored.SourceVM != msg.SourceVM {
		t.Errorf("SourceVM mismatch: got %s, want %s", restored.SourceVM, msg.SourceVM)
	}
	if restored.Status != msg.Status {
		t.Errorf("Status mismatch: got %s, want %s", restored.Status, msg.Status)
	}
	if restored.CreatedHeight != msg.CreatedHeight {
		t.Errorf("CreatedHeight mismatch: got %d, want %d", restored.CreatedHeight, msg.CreatedHeight)
	}
}

func TestGenesisStateValidation(t *testing.T) {
	// Default genesis should be valid
	gs := types.DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Errorf("default genesis should be valid: %v", err)
	}

	// Genesis with valid message
	gs.Messages = []types.CrossVMMessage{
		{
			ID:             "gen-msg-1",
			SourceVM:       types.VMTypeEVM,
			TargetVM:       types.VMTypeCosmWasm,
			Sender:         testSender,
			TargetContract: "qor1target",
			Payload:        []byte("test"),
			Status:         types.StatusExecuted,
		},
	}
	if err := gs.Validate(); err != nil {
		t.Errorf("genesis with valid message should be valid: %v", err)
	}

	// Genesis with invalid message
	gs.Messages = []types.CrossVMMessage{
		{
			ID:       "",
			SourceVM: types.VMTypeEVM,
			TargetVM: types.VMTypeCosmWasm,
		},
	}
	if err := gs.Validate(); err == nil {
		t.Error("genesis with invalid message should be invalid")
	}
}

package types_test

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

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
				Sender:         "0x1234",
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
				Sender:         "qor1sender",
				TargetContract: "0xdeadbeef",
				Payload:        []byte{0x01, 0x02},
			},
			wantErr: false,
		},
		{
			name: "empty ID",
			msg: types.CrossVMMessage{
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
				SourceVM:       "invalid",
				TargetVM:       types.VMTypeCosmWasm,
				TargetContract: "qor1abc",
				Payload:        []byte("test"),
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
		Sender:         "0xsender",
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
			Sender:         "0xsender",
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

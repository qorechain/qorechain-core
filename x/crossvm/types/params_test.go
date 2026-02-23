package types_test

import (
	"testing"

	"github.com/qorechain/qorechain-core/x/crossvm/types"
)

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()
	if params.MaxMessageSize != 65536 {
		t.Errorf("expected MaxMessageSize 65536, got %d", params.MaxMessageSize)
	}
	if params.MaxQueueSize != 1000 {
		t.Errorf("expected MaxQueueSize 1000, got %d", params.MaxQueueSize)
	}
	if params.QueueTimeoutBlocks != 100 {
		t.Errorf("expected QueueTimeoutBlocks 100, got %d", params.QueueTimeoutBlocks)
	}
	if !params.Enabled {
		t.Error("expected Enabled to be true")
	}
	if err := params.Validate(); err != nil {
		t.Errorf("default params should be valid: %v", err)
	}
}

func TestParamsValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{
			name:    "valid default params",
			params:  types.DefaultParams(),
			wantErr: false,
		},
		{
			name: "zero max message size",
			params: types.Params{
				MaxMessageSize:     0,
				MaxQueueSize:       1000,
				QueueTimeoutBlocks: 100,
				Enabled:            true,
			},
			wantErr: true,
		},
		{
			name: "zero max queue size",
			params: types.Params{
				MaxMessageSize:     65536,
				MaxQueueSize:       0,
				QueueTimeoutBlocks: 100,
				Enabled:            true,
			},
			wantErr: true,
		},
		{
			name: "negative queue timeout",
			params: types.Params{
				MaxMessageSize:     65536,
				MaxQueueSize:       1000,
				QueueTimeoutBlocks: -1,
				Enabled:            true,
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

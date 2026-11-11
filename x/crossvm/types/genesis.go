package types

import (
	"encoding/json"
	"fmt"
)

// GenesisState defines the crossvm module's genesis state.
type GenesisState struct {
	Params   Params           `json:"params"`
	Messages []CrossVMMessage `json:"messages"`
	// MsgCounter is the message-ID counter, exported so that a chain started
	// from an export never generates the ID of an exported record again.
	// omitempty keeps genesis files that predate it valid and unchanged.
	MsgCounter uint64 `json:"msg_counter,omitempty"`
	// WasmCongestionFactor and WasmCongestionExhaustedHeight carry the
	// CosmWasm congestion state (wasm-congestion-pricing) across an export.
	WasmCongestionFactor          uint64 `json:"wasm_congestion_factor,omitempty"`
	WasmCongestionExhaustedHeight int64  `json:"wasm_congestion_exhausted_height,omitempty"`
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params:   DefaultParams(),
		Messages: []CrossVMMessage{},
	}
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if f := gs.WasmCongestionFactor; f != 0 &&
		(f < CongestionFactorOne || f > uint64(MaxWasmCongestionMaxFactor)*CongestionFactorOne) {
		return fmt.Errorf("wasm congestion factor %d is outside %d..%d", f, CongestionFactorOne, uint64(MaxWasmCongestionMaxFactor)*CongestionFactorOne)
	}
	for _, msg := range gs.Messages {
		if err := msg.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Marshal returns the JSON encoding of the genesis state.
func (gs GenesisState) Marshal() ([]byte, error) {
	return json.Marshal(gs)
}

// Unmarshal parses the JSON-encoded genesis state.
func UnmarshalGenesisState(bz []byte) (*GenesisState, error) {
	var gs GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return nil, err
	}
	return &gs, nil
}

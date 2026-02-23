package types

import "encoding/json"

// GenesisState defines the crossvm module's genesis state.
type GenesisState struct {
	Params   Params           `json:"params"`
	Messages []CrossVMMessage `json:"messages"`
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

package types

import (
	"encoding/json"
	"fmt"
)

// GenesisState defines the SVM module's genesis state.
type GenesisState struct {
	Params      Params        `json:"params"`
	Accounts    []SVMAccount  `json:"accounts"`
	Programs    []ProgramMeta `json:"programs"`
	CurrentSlot uint64        `json:"current_slot"`
}

// DefaultGenesis returns a genesis state with default params and system program accounts.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:      DefaultParams(),
		Accounts:    defaultSystemAccounts(),
		Programs:    []ProgramMeta{},
		CurrentSlot: 0,
	}
}

// defaultSystemAccounts returns the built-in system program accounts (all executable).
func defaultSystemAccounts() []SVMAccount {
	programs := [][32]byte{
		SystemProgramAddress,
		SPLTokenAddress,
		ATAAddress,
		MemoAddress,
		QorPQCAddress,
		QorAIAddress,
	}

	accounts := make([]SVMAccount, len(programs))
	for i, addr := range programs {
		accounts[i] = SVMAccount{
			Address:    addr,
			Lamports:   1, // minimum balance
			DataLen:    0,
			Data:       []byte{},
			Owner:      SystemProgramAddress, // owned by system program
			Executable: true,
			RentEpoch:  0,
		}
	}
	return accounts
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[[32]byte]bool)
	for _, acc := range gs.Accounts {
		// Built-in system programs use SystemProgramAddress (the zero address)
		// as their owner. Skip SVMAccount.Validate() for these native program
		// entries because the standard validator rejects zero owner on
		// executable accounts — which is correct for user-deployed programs
		// but not for the pre-deployed native programs in genesis.
		if !(acc.Executable && acc.Owner == SystemProgramAddress) {
			if err := acc.Validate(); err != nil {
				return err
			}
		}
		if seen[acc.Address] {
			return fmt.Errorf("duplicate account address in genesis")
		}
		seen[acc.Address] = true
	}
	for _, prog := range gs.Programs {
		var zeroAddr [32]byte
		if prog.ProgramAddress == zeroAddr {
			return fmt.Errorf("program address cannot be zero")
		}
	}
	return nil
}

// Marshal returns the JSON encoding of the genesis state.
func (gs GenesisState) Marshal() ([]byte, error) {
	return json.Marshal(gs)
}

// UnmarshalGenesisState parses the JSON-encoded genesis state.
func UnmarshalGenesisState(bz []byte) (*GenesisState, error) {
	var gs GenesisState
	if err := json.Unmarshal(bz, &gs); err != nil {
		return nil, err
	}
	return &gs, nil
}

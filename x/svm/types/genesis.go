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
	// Dust is every wallet's sub-uqor lamport remainder, so an export and
	// import keeps it. Absent in older exports.
	Dust []DustEntry `json:"dust,omitempty"`
	// CustodyLiability (lamports, decimal), CustodyWindowStart and
	// CustodyWindowNet (lamports, decimal, may be negative) carry the custody
	// accounting of svm-custody-liability-window. Absent in older exports.
	CustodyLiability   string `json:"custody_liability,omitempty"`
	CustodyWindowStart int64  `json:"custody_window_start,omitempty"`
	CustodyWindowNet   string `json:"custody_window_net,omitempty"`
	// PQCKeys are the keys registered with MsgRegisterSVMPQCKey. Absent in
	// older exports.
	PQCKeys []PQCKeyEntry `json:"pqc_keys,omitempty"`
}

// PQCKeyEntry is one SVM address's registered PQC public key.
type PQCKeyEntry struct {
	Address [32]byte `json:"address"`
	Key     []byte   `json:"key"`
}

// DustEntry is one address's dust in lamports (1..LamportsPerUqor-1).
type DustEntry struct {
	Address  [32]byte `json:"address"`
	Lamports uint64   `json:"lamports"`
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
	dust := make(map[[32]byte]bool, len(gs.Dust))
	for _, d := range gs.Dust {
		if d.Lamports == 0 || d.Lamports >= LamportsPerUqor || !IsCanonicalWalletAddress(d.Address) || dust[d.Address] {
			return fmt.Errorf("invalid dust entry for %s", Base58Encode(d.Address))
		}
		dust[d.Address] = true
	}
	keys := make(map[[32]byte]bool, len(gs.PQCKeys))
	for _, k := range gs.PQCKeys {
		if len(k.Key) != SVMPQCPublicKeyLen || keys[k.Address] {
			return fmt.Errorf("invalid pqc key entry for %s", Base58Encode(k.Address))
		}
		keys[k.Address] = true
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

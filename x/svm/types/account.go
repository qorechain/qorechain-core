package types

import (
	"encoding/json"
	"fmt"
)

// SVMAccount represents an account in the SVM runtime, mirroring the
// account model used by SVM-compatible virtual machines.
type SVMAccount struct {
	Address    [32]byte `json:"address"`
	Lamports   uint64   `json:"lamports"`
	DataLen    uint64   `json:"data_len"`
	Data       []byte   `json:"data"`
	Owner      [32]byte `json:"owner"`
	Executable bool     `json:"executable"`
	RentEpoch  uint64   `json:"rent_epoch"`
}

// Validate checks the SVMAccount for internal consistency.
func (a *SVMAccount) Validate() error {
	var zeroAddr [32]byte
	if a.Address == zeroAddr {
		return fmt.Errorf("SVM account address cannot be zero")
	}
	if uint64(len(a.Data)) != a.DataLen {
		return fmt.Errorf("data length mismatch: data has %d bytes but data_len is %d", len(a.Data), a.DataLen)
	}
	if a.Executable && a.Owner == zeroAddr {
		return fmt.Errorf("executable account must have a non-zero owner")
	}
	return nil
}

// Marshal serializes the SVMAccount to JSON bytes.
func (a *SVMAccount) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

// Unmarshal deserializes the SVMAccount from JSON bytes.
func (a *SVMAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, a)
}

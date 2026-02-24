package types

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
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
		return errorsmod.Wrap(ErrInvalidAddress, "SVM account address cannot be zero")
	}
	if uint64(len(a.Data)) != a.DataLen {
		return errorsmod.Wrapf(ErrInvalidAccountOwner, "data length mismatch: declared %d, actual %d", a.DataLen, len(a.Data))
	}
	if a.Executable && a.Owner == zeroAddr {
		return errorsmod.Wrap(ErrInvalidAccountOwner, "executable account must have non-zero owner")
	}
	return nil
}

// Marshal serializes the SVM account to bytes.
// TODO: replace JSON encoding with binary (protobuf or manual) before production use.
func (a *SVMAccount) Marshal() ([]byte, error) {
	return json.Marshal(a)
}

// Unmarshal deserializes the SVMAccount from JSON bytes.
func (a *SVMAccount) Unmarshal(data []byte) error {
	return json.Unmarshal(data, a)
}

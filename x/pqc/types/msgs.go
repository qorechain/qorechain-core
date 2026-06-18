package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NOTE: MsgRegisterPQCKey, MsgRegisterPQCKeyV2, MsgMigratePQCKey,
// MsgDeprecateAlgorithm and MsgDisableAlgorithm are now generated from
// proto/qorechain/pqc/v1/tx.proto (see tx.pb.go). Their ValidateBasic
// implementations live in msgs_validate.go. Only MsgAddAlgorithm remains
// hand-written here because it embeds AlgorithmInfo, which is migrated to
// proto in a later pass.

// ---------------------------------------------------------------------------
// MsgAddAlgorithm — governance message to add a new PQC algorithm
// ---------------------------------------------------------------------------

// MsgAddAlgorithm proposes adding a new PQC algorithm to the registry.
// Must be submitted through governance (MsgSubmitProposal).
type MsgAddAlgorithm struct {
	Authority string        `json:"authority"` // Governance module address
	Algorithm AlgorithmInfo `json:"algorithm"`
}

func (msg *MsgAddAlgorithm) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	return msg.Algorithm.Validate()
}

func (msg *MsgAddAlgorithm) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg *MsgAddAlgorithm) ProtoMessage()  {}
func (msg *MsgAddAlgorithm) Reset()         {}
func (msg *MsgAddAlgorithm) String() string { return "MsgAddAlgorithm" }
func (msg *MsgAddAlgorithm) XXX_MessageName() string {
	return "qorechain.pqc.v1.MsgAddAlgorithm"
}

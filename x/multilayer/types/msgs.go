package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// The MsgRegisterSidechain, MsgRegisterPaychain, MsgAnchorState,
// MsgRouteTransaction, MsgUpdateLayerStatus and MsgChallengeAnchor messages
// (and their responses) are generated from
// proto/qorechain/multilayer/v1/tx.proto (see tx.pb.go). MsgUpdateParams
// remains hand-written (embeds Params; migrated to proto in a later pass).
// The ValidateBasic / GetSigners methods below are attached to the generated
// message types.

const (
	TypeMsgRegisterSidechain = "register_sidechain"
	TypeMsgRegisterPaychain  = "register_paychain"
	TypeMsgAnchorState       = "anchor_state"
	TypeMsgRouteTransaction  = "route_transaction"
	TypeMsgUpdateLayerStatus = "update_layer_status"
	TypeMsgChallengeAnchor   = "challenge_anchor"
	TypeMsgUpdateParams      = "update_params"
)

func (msg MsgRegisterSidechain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.LayerID == "" {
		return fmt.Errorf("layer_id cannot be empty")
	}
	if msg.MinValidators == 0 {
		return fmt.Errorf("min_validators must be > 0")
	}
	if msg.SettlementIntervalBlocks == 0 {
		return fmt.Errorf("settlement_interval_blocks must be > 0")
	}
	return nil
}

func (msg MsgRegisterSidechain) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

func (msg MsgRegisterPaychain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.LayerID == "" {
		return fmt.Errorf("layer_id cannot be empty")
	}
	if msg.SettlementIntervalBlocks == 0 {
		return fmt.Errorf("settlement_interval_blocks must be > 0")
	}
	return nil
}

func (msg MsgRegisterPaychain) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(msg.Creator)
	return []sdk.AccAddress{creator}
}

func (msg MsgAnchorState) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Relayer); err != nil {
		return fmt.Errorf("invalid relayer address: %w", err)
	}
	if msg.LayerID == "" {
		return fmt.Errorf("layer_id cannot be empty")
	}
	if len(msg.StateRoot) == 0 {
		return fmt.Errorf("state_root cannot be empty")
	}
	if len(msg.PQCAggregateSignature) == 0 {
		return fmt.Errorf("pqc_aggregate_signature cannot be empty")
	}
	return nil
}

func (msg MsgAnchorState) GetSigners() []sdk.AccAddress {
	relayer, _ := sdk.AccAddressFromBech32(msg.Relayer)
	return []sdk.AccAddress{relayer}
}

func (msg MsgRouteTransaction) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if len(msg.TransactionPayload) == 0 {
		return fmt.Errorf("transaction_payload cannot be empty")
	}
	return nil
}

func (msg MsgRouteTransaction) GetSigners() []sdk.AccAddress {
	sender, _ := sdk.AccAddressFromBech32(msg.Sender)
	return []sdk.AccAddress{sender}
}

func (msg MsgUpdateLayerStatus) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.LayerID == "" {
		return fmt.Errorf("layer_id cannot be empty")
	}
	if msg.NewStatus == "" {
		return fmt.Errorf("new_status cannot be empty")
	}
	return nil
}

func (msg MsgUpdateLayerStatus) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

func (msg MsgChallengeAnchor) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Challenger); err != nil {
		return fmt.Errorf("invalid challenger address: %w", err)
	}
	if msg.LayerID == "" {
		return fmt.Errorf("layer_id cannot be empty")
	}
	if len(msg.FraudProof) == 0 {
		return fmt.Errorf("fraud_proof cannot be empty")
	}
	return nil
}

func (msg MsgChallengeAnchor) GetSigners() []sdk.AccAddress {
	challenger, _ := sdk.AccAddressFromBech32(msg.Challenger)
	return []sdk.AccAddress{challenger}
}

// MsgUpdateParams updates module parameters (governance only). Hand-written
// (embeds Params); migrated to proto in a later pass.
type MsgUpdateParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return msg.Params.Validate()
}

func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// MsgUpdateParamsResponse is the response for MsgUpdateParams.
type MsgUpdateParamsResponse struct{}

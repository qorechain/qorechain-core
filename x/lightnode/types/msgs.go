package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgRegisterLightNode, MsgHeartbeat, MsgDeregisterLightNode and
// MsgClaimLightNodeRewards are generated from
// proto/qorechain/lightnode/v1/tx.proto (see tx.pb.go). MsgUpdateLightNodeParams
// remains hand-written (embeds Params; migrated to proto in a later pass).
// The methods below are attached to the generated message types.

// Message type constants.
const (
	TypeMsgRegisterLightNode     = "register_light_node"
	TypeMsgHeartbeat             = "heartbeat"
	TypeMsgDeregisterLightNode   = "deregister_light_node"
	TypeMsgClaimLightNodeRewards = "claim_light_node_rewards"
	TypeMsgUpdateLightNodeParams = "update_light_node_params"
)

func (msg MsgRegisterLightNode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return err
	}
	if !ValidNodeType(NodeType(msg.NodeType)) {
		return ErrInvalidNodeType.Wrapf("got %q", msg.NodeType)
	}
	if msg.Version == "" {
		return ErrInvalidVersion.Wrap("version cannot be empty")
	}
	return nil
}

func (msg MsgRegisterLightNode) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgHeartbeat) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return err
	}
	return nil
}

func (msg MsgHeartbeat) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgDeregisterLightNode) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return err
	}
	return nil
}

func (msg MsgDeregisterLightNode) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

func (msg MsgClaimLightNodeRewards) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return err
	}
	return nil
}

func (msg MsgClaimLightNodeRewards) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Operator)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

// MsgUpdateLightNodeParams is a governance message to update module parameters.
type MsgUpdateLightNodeParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg MsgUpdateLightNodeParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return err
	}
	if err := msg.Params.Validate(); err != nil {
		return ErrInvalidParams.Wrap(err.Error())
	}
	return nil
}

func (msg MsgUpdateLightNodeParams) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return nil
	}
	return []sdk.AccAddress{addr}
}

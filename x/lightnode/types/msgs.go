package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type constants.
const (
	TypeMsgRegisterLightNode     = "register_light_node"
	TypeMsgHeartbeat             = "heartbeat"
	TypeMsgDeregisterLightNode   = "deregister_light_node"
	TypeMsgClaimLightNodeRewards = "claim_light_node_rewards"
	TypeMsgUpdateLightNodeParams = "update_light_node_params"
)

// MsgRegisterLightNode registers a new light node on the network.
type MsgRegisterLightNode struct {
	Operator     string   `json:"operator"`
	NodeType     string   `json:"node_type"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

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

// MsgHeartbeat submits a liveness heartbeat for a registered light node.
type MsgHeartbeat struct {
	Operator string `json:"operator"`
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

// MsgDeregisterLightNode removes a light node from the registry.
type MsgDeregisterLightNode struct {
	Operator string `json:"operator"`
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

// MsgClaimLightNodeRewards claims accumulated rewards for a light node.
type MsgClaimLightNodeRewards struct {
	Operator string `json:"operator"`
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

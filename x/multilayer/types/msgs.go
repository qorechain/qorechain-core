package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message types for the QoreChain multi-layer architecture module
const (
	TypeMsgRegisterSidechain = "register_sidechain"
	TypeMsgRegisterPaychain  = "register_paychain"
	TypeMsgAnchorState       = "anchor_state"
	TypeMsgRouteTransaction  = "route_transaction"
	TypeMsgUpdateLayerStatus = "update_layer_status"
	TypeMsgChallengeAnchor   = "challenge_anchor"
	TypeMsgUpdateParams      = "update_params"
)

// MsgRegisterSidechain creates a new sidechain layer in the multi-layer architecture
type MsgRegisterSidechain struct {
	Creator                  string   `json:"creator"`
	LayerID                  string   `json:"layer_id"`
	Description              string   `json:"description"`
	TargetBlockTimeMs        uint64   `json:"target_block_time_ms"`
	MaxTransactionsPerBlock  uint64   `json:"max_transactions_per_block"`
	MinValidators            uint32   `json:"min_validators"`
	SettlementIntervalBlocks uint64   `json:"settlement_interval_blocks"`
	SupportedVMTypes         []string `json:"supported_vm_types"`
	SupportedDomains         []string `json:"supported_domains"`
}

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

// MsgRegisterSidechainResponse is the response for MsgRegisterSidechain
type MsgRegisterSidechainResponse struct {
	LayerID string      `json:"layer_id"`
	ChainID string      `json:"chain_id"` // Assigned ICS chain ID
	Status  LayerStatus `json:"status"`
}

// MsgRegisterPaychain creates a new paychain layer for high-frequency microtransactions
type MsgRegisterPaychain struct {
	Creator                  string `json:"creator"`
	LayerID                  string `json:"layer_id"`
	Description              string `json:"description"`
	MaxTransactionsPerBlock  uint64 `json:"max_transactions_per_block"`
	SettlementIntervalBlocks uint64 `json:"settlement_interval_blocks"` // Batched settlement frequency
	BaseFeeMultiplier        string `json:"base_fee_multiplier"`        // e.g., "0.01" for 1/100th of main chain fees
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

// MsgRegisterPaychainResponse is the response for MsgRegisterPaychain
type MsgRegisterPaychainResponse struct {
	LayerID string      `json:"layer_id"`
	Status  LayerStatus `json:"status"`
}

// MsgAnchorState submits a state root commitment from a subsidiary chain
// using Hierarchical Commitment Schemes (HCS) with PQC-signed attestations
type MsgAnchorState struct {
	Relayer               string `json:"relayer"`
	LayerID               string `json:"layer_id"`
	LayerHeight           uint64 `json:"layer_height"`
	StateRoot             []byte `json:"state_root"`
	ValidatorSetHash      []byte `json:"validator_set_hash"`
	PQCAggregateSignature []byte `json:"pqc_aggregate_signature"`
	TransactionCount      uint64 `json:"transaction_count"`
	CompressedStateProof  []byte `json:"compressed_state_proof"`
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

// MsgAnchorStateResponse is the response for MsgAnchorState
type MsgAnchorStateResponse struct {
	MainChainHeight uint64 `json:"main_chain_height"`
	Accepted        bool   `json:"accepted"`
}

// MsgRouteTransaction requests QCAI routing for a transaction to the optimal layer
type MsgRouteTransaction struct {
	Sender             string `json:"sender"`
	TransactionPayload []byte `json:"transaction_payload"`
	PreferredLayer     string `json:"preferred_layer,omitempty"` // Optional hint
	MaxLatencyMs       uint64 `json:"max_latency_ms"`           // Max acceptable latency
	MaxFee             string `json:"max_fee"`                   // Max fee willing to pay (uqor)
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

// MsgRouteTransactionResponse is the response for MsgRouteTransaction
type MsgRouteTransactionResponse struct {
	Decision             *RoutingDecision `json:"decision"`
	CrossLayerMessageID  string           `json:"cross_layer_message_id,omitempty"`
}

// MsgUpdateLayerStatus changes a layer's status (suspend, activate, decommission)
type MsgUpdateLayerStatus struct {
	Authority string      `json:"authority"` // Governance or admin address
	LayerID   string      `json:"layer_id"`
	NewStatus LayerStatus `json:"new_status"`
	Reason    string      `json:"reason"`
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

// MsgUpdateLayerStatusResponse is the response for MsgUpdateLayerStatus
type MsgUpdateLayerStatusResponse struct{}

// MsgChallengeAnchor disputes a state anchor during the challenge period
type MsgChallengeAnchor struct {
	Challenger      string `json:"challenger"`
	LayerID         string `json:"layer_id"`
	AnchorHeight    uint64 `json:"anchor_height"`
	FraudProof      []byte `json:"fraud_proof"`      // Proof that the anchor is invalid
	ChallengeReason string `json:"challenge_reason"`
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

// MsgChallengeAnchorResponse is the response for MsgChallengeAnchor
type MsgChallengeAnchorResponse struct {
	ChallengeAccepted bool   `json:"challenge_accepted"`
	Resolution        string `json:"resolution"`
}

// MsgUpdateParams updates module parameters (governance only)
type MsgUpdateParams struct {
	Authority string `json:"authority"` // Governance module address
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

// MsgUpdateParamsResponse is the response for MsgUpdateParams
type MsgUpdateParamsResponse struct{}

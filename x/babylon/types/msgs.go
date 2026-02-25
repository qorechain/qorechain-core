package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgSubmitBTCCheckpoint submits a BTC checkpoint to the chain.
type MsgSubmitBTCCheckpoint struct {
	Submitter      string `json:"submitter"`
	EpochNum       uint64 `json:"epoch_num"`
	BTCBlockHash   string `json:"btc_block_hash"`
	BTCBlockHeight int64  `json:"btc_block_height"`
	StateRoot      string `json:"state_root"`
}

func (msg MsgSubmitBTCCheckpoint) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.BTCBlockHash == "" {
		return fmt.Errorf("BTC block hash cannot be empty")
	}
	if msg.StateRoot == "" {
		return fmt.Errorf("state root cannot be empty")
	}
	return nil
}

func (msg MsgSubmitBTCCheckpoint) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{addr}
}

// MsgBTCRestake initiates a BTC restaking position.
type MsgBTCRestake struct {
	Staker        string `json:"staker"`
	BTCTxHash     string `json:"btc_tx_hash"`
	AmountSatoshis int64 `json:"amount_satoshis"`
	ValidatorAddr string `json:"validator_addr"`
}

func (msg MsgBTCRestake) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Staker); err != nil {
		return fmt.Errorf("invalid staker address: %w", err)
	}
	if msg.BTCTxHash == "" {
		return fmt.Errorf("BTC tx hash cannot be empty")
	}
	if msg.AmountSatoshis <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}

func (msg MsgBTCRestake) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(msg.Staker)
	return []sdk.AccAddress{addr}
}

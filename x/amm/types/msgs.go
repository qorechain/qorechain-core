package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AMM message types. Wire-level representation lives in proto/qorechain/amm/v1
// — these Go structs are the canonical types used by the keeper, ante
// handler, and CLI.

// MsgCreatePool creates a new liquidity pool.
type MsgCreatePool struct {
	Creator                  string     `json:"creator"` // bech32
	PoolType                 PoolType   `json:"pool_type"`
	InitialDepositA          sdk.Coin   `json:"initial_deposit_a"`
	InitialDepositB          sdk.Coin   `json:"initial_deposit_b"`
	AmplificationCoefficient uint32     `json:"amplification_coefficient,omitempty"`
}

// MsgAddLiquidity adds proportional liquidity to an existing pool.
type MsgAddLiquidity struct {
	Sender   string   `json:"sender"`
	PoolID   uint64   `json:"pool_id"`
	AmountA  sdk.Coin `json:"amount_a"`
	AmountB  sdk.Coin `json:"amount_b"`
	MinLPOut math.Int `json:"min_lp_out"`
}

// MsgRemoveLiquidity burns LP tokens and returns proportional reserves.
type MsgRemoveLiquidity struct {
	Sender    string   `json:"sender"`
	PoolID    uint64   `json:"pool_id"`
	LPAmount  math.Int `json:"lp_amount"`
	MinAmountA math.Int `json:"min_amount_a"`
	MinAmountB math.Int `json:"min_amount_b"`
}

// MsgSwapExactIn swaps a fixed input amount and enforces a minimum output.
type MsgSwapExactIn struct {
	Sender   string   `json:"sender"`
	PoolID   uint64   `json:"pool_id"`
	TokenIn  sdk.Coin `json:"token_in"`
	DenomOut string   `json:"denom_out"`
	MinOut   math.Int `json:"min_out"`
}

// MsgSwapExactOut swaps to a fixed output amount and enforces a maximum input.
type MsgSwapExactOut struct {
	Sender    string   `json:"sender"`
	PoolID    uint64   `json:"pool_id"`
	DenomIn   string   `json:"denom_in"`
	TokenOut  sdk.Coin `json:"token_out"`
	MaxIn     math.Int `json:"max_in"`
}

// MsgPausePool toggles a pool to PoolStatusPaused. Gov-only.
type MsgPausePool struct {
	Authority string `json:"authority"`
	PoolID    uint64 `json:"pool_id"`
	Reason    string `json:"reason,omitempty"`
}

// MsgResumePool clears the paused flag. Gov-only.
type MsgResumePool struct {
	Authority string `json:"authority"`
	PoolID    uint64 `json:"pool_id"`
}

// MsgSetParams updates module Params. Gov-only.
type MsgSetParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

// ----- ValidateBasic implementations -----

func (m MsgCreatePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return sdkerrors.Wrap(ErrInvalidParams, "creator")
	}
	if m.PoolType != PoolTypeConstantProduct && m.PoolType != PoolTypeStableSwap {
		return sdkerrors.Wrapf(ErrInvalidPoolType, "unknown pool_type %q", m.PoolType)
	}
	if m.InitialDepositA.Denom == m.InitialDepositB.Denom {
		return ErrSameDenom
	}
	if m.InitialDepositA.Amount.IsNil() || !m.InitialDepositA.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "initial_deposit_a")
	}
	if m.InitialDepositB.Amount.IsNil() || !m.InitialDepositB.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "initial_deposit_b")
	}
	if m.PoolType == PoolTypeStableSwap {
		if m.AmplificationCoefficient < 1 || m.AmplificationCoefficient > 5000 {
			return fmt.Errorf("amplification_coefficient must be in [1,5000]")
		}
	}
	return nil
}

func (m MsgAddLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrap(ErrInvalidParams, "sender")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	if m.AmountA.Denom == m.AmountB.Denom {
		return ErrSameDenom
	}
	if m.AmountA.Amount.IsNil() || !m.AmountA.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount_a")
	}
	if m.AmountB.Amount.IsNil() || !m.AmountB.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount_b")
	}
	if m.MinLPOut.IsNil() || m.MinLPOut.IsNegative() {
		return sdkerrors.Wrap(ErrInvalidLPAmount, "min_lp_out")
	}
	return nil
}

func (m MsgRemoveLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrap(ErrInvalidParams, "sender")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	if m.LPAmount.IsNil() || !m.LPAmount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidLPAmount, "lp_amount must be positive")
	}
	if m.MinAmountA.IsNil() || m.MinAmountA.IsNegative() {
		return sdkerrors.Wrap(ErrInvalidAmount, "min_amount_a")
	}
	if m.MinAmountB.IsNil() || m.MinAmountB.IsNegative() {
		return sdkerrors.Wrap(ErrInvalidAmount, "min_amount_b")
	}
	return nil
}

func (m MsgSwapExactIn) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrap(ErrInvalidParams, "sender")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	if m.TokenIn.Amount.IsNil() || !m.TokenIn.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "token_in")
	}
	if m.DenomOut == "" {
		return sdkerrors.Wrap(ErrInvalidDenoms, "denom_out empty")
	}
	if m.TokenIn.Denom == m.DenomOut {
		return ErrSameDenom
	}
	if m.MinOut.IsNil() || m.MinOut.IsNegative() {
		return sdkerrors.Wrap(ErrInvalidAmount, "min_out")
	}
	return nil
}

func (m MsgSwapExactOut) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Sender); err != nil {
		return sdkerrors.Wrap(ErrInvalidParams, "sender")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	if m.DenomIn == "" {
		return sdkerrors.Wrap(ErrInvalidDenoms, "denom_in empty")
	}
	if m.TokenOut.Amount.IsNil() || !m.TokenOut.Amount.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "token_out")
	}
	if m.DenomIn == m.TokenOut.Denom {
		return ErrSameDenom
	}
	if m.MaxIn.IsNil() || !m.MaxIn.IsPositive() {
		return sdkerrors.Wrap(ErrInvalidAmount, "max_in must be positive")
	}
	return nil
}

func (m MsgPausePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.Wrap(ErrUnauthorized, "authority")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	return nil
}

func (m MsgResumePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.Wrap(ErrUnauthorized, "authority")
	}
	if m.PoolID == 0 {
		return sdkerrors.Wrap(ErrPoolNotFound, "pool_id zero")
	}
	return nil
}

func (m MsgSetParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return sdkerrors.Wrap(ErrUnauthorized, "authority")
	}
	return m.Params.Validate()
}

// ----- GetSigners -----

func (m MsgCreatePool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{addr}
}

func (m MsgAddLiquidity) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgRemoveLiquidity) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgSwapExactIn) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgSwapExactOut) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Sender)
	return []sdk.AccAddress{addr}
}

func (m MsgPausePool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m MsgResumePool) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

func (m MsgSetParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

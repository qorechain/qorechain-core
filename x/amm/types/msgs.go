package types

import (
	"fmt"

	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MsgCreatePool, MsgAddLiquidity, MsgRemoveLiquidity, MsgSwapExactIn,
// MsgSwapExactOut, MsgPausePool and MsgResumePool are generated from
// proto/qorechain/amm/v1/tx.proto (see tx.pb.go). MsgSetParams remains
// hand-written (embeds Params; migrated to proto in a later pass). The
// ValidateBasic / GetSigners methods below are attached to the generated types.

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

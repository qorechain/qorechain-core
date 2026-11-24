package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/qorechain/qorechain-core/x/govauth"
	"github.com/qorechain/qorechain-core/x/qca/types"
)

// QDRWTallyEnabled answers, per block and from chain state, whether
// gov-qdrw-tally is on. Wired by the application; unset or false refuses
// MsgUpdateQDRWConfig and keeps the stock governance tally.
var QDRWTallyEnabled func(ctx sdk.Context) bool

// QDRWTallyActive is true when gov-qdrw-tally is on and the stored QDRW
// configuration is enabled.
func (k Keeper) QDRWTallyActive(ctx sdk.Context) bool {
	return QDRWTallyEnabled != nil && QDRWTallyEnabled(ctx) && k.GetConfig(ctx).QDRWConfig.Enabled
}

type msgServer struct{ k Keeper }

// NewMsgServerImpl returns the qca Msg service.
func NewMsgServerImpl(k Keeper) types.MsgServer { return msgServer{k: k} }

// ValidateQDRWMultipliers checks a reputation multiplier band: within
// [0.5, 2] and min <= max.
func ValidateQDRWMultipliers(minS, maxS string) (math.LegacyDec, math.LegacyDec, error) {
	lo, err := math.LegacyNewDecFromStr(minS)
	if err != nil {
		return lo, lo, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "rep_min_multiplier: %s", err)
	}
	hi, err := math.LegacyNewDecFromStr(maxS)
	if err != nil {
		return lo, hi, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "rep_max_multiplier: %s", err)
	}
	floor, ceil := math.LegacyMustNewDecFromStr("0.5"), math.LegacyNewDec(2)
	if lo.LT(floor) || hi.GT(ceil) || hi.LT(lo) {
		return lo, hi, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "reputation multipliers must satisfy 0.5 <= min <= max <= 2, got [%s, %s]", lo, hi)
	}
	return lo, hi, nil
}

func (s msgServer) UpdateQDRWConfig(goCtx context.Context, msg *types.MsgUpdateQDRWConfig) (*types.MsgUpdateQDRWConfigResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// Governance-only: the authority field is the declared signer, so it
	// proves nothing by itself. See x/govauth.
	if err := govauth.Assert(msg.Authority); err != nil {
		return nil, err
	}
	if QDRWTallyEnabled == nil || !QDRWTallyEnabled(ctx) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "MsgUpdateQDRWConfig is not active before gov-qdrw-tally")
	}
	if _, _, err := ValidateQDRWMultipliers(msg.RepMinMultiplier, msg.RepMaxMultiplier); err != nil {
		return nil, err
	}
	cfg := s.k.GetConfig(ctx)
	cfg.QDRWConfig.Enabled = msg.Enabled
	cfg.QDRWConfig.RepMinMultiplier = msg.RepMinMultiplier
	cfg.QDRWConfig.RepMaxMultiplier = msg.RepMaxMultiplier
	if err := s.k.SetConfig(ctx, cfg); err != nil {
		return nil, err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent("qca_qdrw_config_updated",
		sdk.NewAttribute("enabled", strconv.FormatBool(msg.Enabled)),
		sdk.NewAttribute("rep_min_multiplier", msg.RepMinMultiplier),
		sdk.NewAttribute("rep_max_multiplier", msg.RepMaxMultiplier),
	))
	return &types.MsgUpdateQDRWConfigResponse{}, nil
}

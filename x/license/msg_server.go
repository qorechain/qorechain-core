package license

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/govauth"
	"github.com/qorechain/qorechain-core/x/license/types"
)

// msgServer implements the proto-generated license MsgServer by delegating to
// the LicenseKeeper interface (concrete keeper is supplied at app wiring).
type msgServer struct {
	keeper LicenseKeeper
}

// NewMsgServer returns a license MsgServer backed by the given keeper.
func NewMsgServer(k LicenseKeeper) types.MsgServer {
	return msgServer{keeper: k}
}

var _ types.MsgServer = msgServer{}

func (s msgServer) GrantLicense(goCtx context.Context, msg *types.MsgGrantLicense) (*types.MsgGrantLicenseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	lic := types.License{
		Grantee:   msg.Grantee,
		FeatureID: msg.FeatureID,
		ExpiresAt: msg.ExpiresAt,
		GrantedAt: ctx.BlockHeight(),
		GrantedBy: msg.Authority,
		Metadata:  msg.Metadata,
	}
	if err := s.keeper.GrantLicense(ctx, msg.Authority, lic); err != nil {
		return nil, err
	}
	return &types.MsgGrantLicenseResponse{}, nil
}

func (s msgServer) RevokeLicense(goCtx context.Context, msg *types.MsgRevokeLicense) (*types.MsgRevokeLicenseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.RevokeLicense(ctx, msg.Authority, msg.Grantee, msg.FeatureID); err != nil {
		return nil, err
	}
	return &types.MsgRevokeLicenseResponse{}, nil
}

func (s msgServer) SuspendLicense(goCtx context.Context, msg *types.MsgSuspendLicense) (*types.MsgSuspendLicenseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.SuspendLicense(ctx, msg.Authority, msg.Grantee, msg.FeatureID); err != nil {
		return nil, err
	}
	return &types.MsgSuspendLicenseResponse{}, nil
}

func (s msgServer) ResumeLicense(goCtx context.Context, msg *types.MsgResumeLicense) (*types.MsgResumeLicenseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.ResumeLicense(ctx, msg.Authority, msg.Grantee, msg.FeatureID); err != nil {
		return nil, err
	}
	return &types.MsgResumeLicenseResponse{}, nil
}

// AuthorityUpdateEnabled answers, per block and from chain state, whether
// MsgUpdateAuthority may execute. The application wires it to the
// license-authority-update protocol feature, so every node accepts the message
// from the same height; unset, the message is refused.
var AuthorityUpdateEnabled func(ctx sdk.Context) bool

// UpdateAuthority replaces the licence grant authority. Only the governance
// module account may call it, which means it runs only from a passed proposal.
// This is the recovery path when the authority's key is lost.
func (s msgServer) UpdateAuthority(goCtx context.Context, msg *types.MsgUpdateAuthority) (*types.MsgUpdateAuthorityResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if AuthorityUpdateEnabled == nil || !AuthorityUpdateEnabled(ctx) {
		return nil, fmt.Errorf("licence authority updates are not enabled at this height")
	}
	if err := govauth.Assert(msg.Authority); err != nil {
		return nil, err
	}
	if _, err := sdk.AccAddressFromBech32(msg.NewAuthority); err != nil {
		return nil, fmt.Errorf("invalid new authority address: %w", err)
	}
	s.keeper.SetAuthority(ctx, msg.NewAuthority)
	ctx.EventManager().EmitEvent(sdk.NewEvent("license_authority_updated",
		sdk.NewAttribute("new_authority", msg.NewAuthority),
	))
	return &types.MsgUpdateAuthorityResponse{}, nil
}

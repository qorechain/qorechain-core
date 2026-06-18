package license

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

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
	if err := s.keeper.SuspendLicense(ctx, msg.Grantee, msg.FeatureID); err != nil {
		return nil, err
	}
	return &types.MsgSuspendLicenseResponse{}, nil
}

func (s msgServer) ResumeLicense(goCtx context.Context, msg *types.MsgResumeLicense) (*types.MsgResumeLicenseResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := s.keeper.ResumeLicense(ctx, msg.Grantee, msg.FeatureID); err != nil {
		return nil, err
	}
	return &types.MsgResumeLicenseResponse{}, nil
}

package license

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// queryServer implements the license gRPC Query service over the LicenseKeeper
// interface.
type queryServer struct {
	keeper LicenseKeeper
}

// NewQueryServer returns a license QueryServer backed by the given keeper.
func NewQueryServer(k LicenseKeeper) types.QueryServer {
	return queryServer{keeper: k}
}

var _ types.QueryServer = queryServer{}

// licenseView renders a grant as seen at height. Active uses the same predicate
// as the keeper's on-chain gate (License.IsActive), so an expired grant that is
// still waiting for the EndBlocker sweep is not reported as active.
func licenseView(l types.License, height int64) *types.LicenseView {
	return &types.LicenseView{
		Grantee:   l.Grantee,
		FeatureId: l.FeatureID,
		ExpiresAt: l.ExpiresAt,
		GrantedAt: l.GrantedAt,
		GrantedBy: l.GrantedBy,
		Suspended: l.Suspended,
		Metadata:  l.Metadata,
		Active:    l.IsActive(height),
	}
}

func licenseViews(ls []types.License, height int64) []*types.LicenseView {
	out := make([]*types.LicenseView, len(ls))
	for i := range ls {
		out[i] = licenseView(ls[i], height)
	}
	return out
}

func (q queryServer) Check(goCtx context.Context, req *types.QueryCheckRequest) (*types.QueryCheckResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	l, err := q.keeper.GetLicense(ctx, req.Grantee, req.FeatureId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "no license for %s/%s", req.Grantee, req.FeatureId)
	}
	height := ctx.BlockHeight()
	return &types.QueryCheckResponse{License: licenseView(l, height), Active: l.IsActive(height)}, nil
}

func (q queryServer) Holders(goCtx context.Context, req *types.QueryHoldersRequest) (*types.QueryHoldersResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ls, err := q.keeper.GetLicenseHolders(ctx, req.FeatureId)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryHoldersResponse{Licenses: licenseViews(ls, ctx.BlockHeight())}, nil
}

func (q queryServer) List(goCtx context.Context, req *types.QueryListRequest) (*types.QueryListResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ls, err := q.keeper.GetLicenses(ctx, req.Grantee)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryListResponse{Licenses: licenseViews(ls, ctx.BlockHeight())}, nil
}

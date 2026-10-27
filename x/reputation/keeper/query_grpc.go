package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// queryServer implements the proto-generated reputation Query service.
type queryServer struct {
	keeper Keeper
}

// NewQueryServer returns a QueryServer backed by the reputation keeper.
func NewQueryServer(k Keeper) types.QueryServer {
	return queryServer{keeper: k}
}

var _ types.QueryServer = queryServer{}

func (q queryServer) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := q.keeper.GetParams(ctx)
	return &types.QueryParamsResponse{
		Alpha:    p.Alpha,
		Beta:     p.Beta,
		Gamma:    p.Gamma,
		Delta:    p.Delta,
		Lambda:   p.Lambda,
		MinScore: p.MinScore,
	}, nil
}

// Reputation returns one validator's record. The store is keyed by consensus
// address; an operator address is resolved through x/staking first.
func (q queryServer) Reputation(goCtx context.Context, req *types.QueryReputationRequest) (*types.QueryReputationResponse, error) {
	if req == nil || req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "validator_address is required")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	consAddr, operator, err := q.resolve(ctx, req.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	rep, found := q.keeper.GetValidatorReputation(ctx, consAddr)
	if !found {
		return nil, status.Errorf(codes.NotFound, "no reputation record for %s", req.ValidatorAddress)
	}
	rec := toRecord(rep)
	if operator != "" {
		rec.OperatorAddress = operator
	} else {
		rec.OperatorAddress = q.operatorOf(ctx, consAddr)
	}
	return &types.QueryReputationResponse{Reputation: rec}, nil
}

// Reputations returns every stored record (one per validator ever scored),
// in store order, i.e. by consensus address.
func (q queryServer) Reputations(goCtx context.Context, _ *types.QueryReputationsRequest) (*types.QueryReputationsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	reps := q.keeper.GetAllValidatorReputations(ctx)
	out := make([]*types.ReputationRecord, 0, len(reps))
	for _, rep := range reps {
		rec := toRecord(rep)
		rec.OperatorAddress = q.operatorOf(ctx, rep.Address)
		out = append(out, rec)
	}
	return &types.QueryReputationsResponse{Reputations: out}, nil
}

// resolve maps a qorvaloper or qorvalcons address to the consensus address
// the store is keyed by, and returns the operator address when it was given.
func (q queryServer) resolve(ctx sdk.Context, addr string) (string, string, error) {
	if valAddr, err := sdk.ValAddressFromBech32(addr); err == nil {
		if q.keeper.staking == nil {
			return "", "", status.Error(codes.Unavailable, "staking reader not configured")
		}
		val, err := q.keeper.staking.GetValidator(ctx, valAddr)
		if err != nil {
			return "", "", status.Errorf(codes.NotFound, "validator %s not found", addr)
		}
		consBz, err := val.GetConsAddr()
		if err != nil {
			return "", "", status.Errorf(codes.Internal, "consensus address of %s: %v", addr, err)
		}
		return sdk.ConsAddress(consBz).String(), addr, nil
	}
	if consAddr, err := sdk.ConsAddressFromBech32(addr); err == nil {
		return consAddr.String(), "", nil
	}
	return "", "", status.Errorf(codes.InvalidArgument, "%s is neither a validator operator nor a consensus address", addr)
}

// operatorOf returns the operator address of a consensus address, or "" when
// x/staking does not know it (for example a validator that was removed).
func (q queryServer) operatorOf(ctx sdk.Context, consAddr string) string {
	if q.keeper.staking == nil {
		return ""
	}
	ca, err := sdk.ConsAddressFromBech32(consAddr)
	if err != nil {
		return ""
	}
	val, err := q.keeper.staking.GetValidatorByConsAddr(ctx, ca)
	if err != nil {
		return ""
	}
	return val.GetOperator()
}

func toRecord(rep types.ValidatorReputation) *types.ReputationRecord {
	return &types.ReputationRecord{
		ConsensusAddress:  rep.Address,
		StakeScore:        rep.StakeScore,
		PerformanceScore:  rep.PerformanceScore,
		ContributionScore: rep.ContributionScore,
		TimeScore:         rep.TimeScore,
		CompositeScore:    rep.CompositeScore,
		LastUpdatedHeight: rep.LastUpdatedHeight,
		UptimeBlocks:      rep.UptimeBlocks,
		ProposedBlocks:    rep.ProposedBlocks,
		MissedBlocks:      rep.MissedBlocks,
		SlashingEvents:    rep.SlashingEvents,
		CommunityVotes:    rep.CommunityVotes,
		JoinedAtHeight:    rep.JoinedAtHeight,
		GovParticipation:  rep.GovParticipation,
		InactiveSince:     rep.InactiveSince,
	}
}

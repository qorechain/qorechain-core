package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

func TestQueryReputationByOperatorAndConsensusAddress(t *testing.T) {
	k, ctx, _ := setupRepV2(t, true)
	valA, consA := bondedValidator(t, 1, stakingtypes.Bonded, 100)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	consAStr := sdk.ConsAddress(consA).String()
	k.SetValidatorReputation(ctx, types.ValidatorReputation{
		Address: consAStr, CompositeScore: "0.731000000000000000", MissedBlocks: 3, JoinedAtHeight: 7, InactiveSince: 0,
	})
	// A record whose validator x/staking no longer knows.
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: consStr(9), CompositeScore: "0.1"})
	q := NewQueryServer(*k)

	byOp, err := q.Reputation(ctx, &types.QueryReputationRequest{ValidatorAddress: valA.OperatorAddress})
	if err != nil {
		t.Fatal(err)
	}
	r := byOp.Reputation
	if r.ConsensusAddress != consAStr || r.OperatorAddress != valA.OperatorAddress || r.CompositeScore != "0.731000000000000000" || r.MissedBlocks != 3 || r.JoinedAtHeight != 7 {
		t.Fatalf("by operator: %+v", r)
	}

	byCons, err := q.Reputation(ctx, &types.QueryReputationRequest{ValidatorAddress: consAStr})
	if err != nil {
		t.Fatal(err)
	}
	if byCons.Reputation.OperatorAddress != valA.OperatorAddress || byCons.Reputation.CompositeScore != r.CompositeScore {
		t.Fatalf("by consensus: %+v", byCons.Reputation)
	}

	gone, err := q.Reputation(ctx, &types.QueryReputationRequest{ValidatorAddress: consStr(9)})
	if err != nil {
		t.Fatal(err)
	}
	if gone.Reputation.OperatorAddress != "" {
		t.Fatalf("unknown validator got operator %q", gone.Reputation.OperatorAddress)
	}

	all, err := q.Reputations(ctx, &types.QueryReputationsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Reputations) != 2 {
		t.Fatalf("want 2 records, got %d", len(all.Reputations))
	}
	for _, rec := range all.Reputations {
		if rec.ConsensusAddress == consAStr && rec.OperatorAddress != valA.OperatorAddress {
			t.Fatalf("list lost the operator: %+v", rec)
		}
	}
}

func TestQueryReputationErrors(t *testing.T) {
	k, ctx, _ := setupRepV2(t, true)
	valA, _ := bondedValidator(t, 1, stakingtypes.Bonded, 100)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	q := NewQueryServer(*k)

	cases := []struct {
		addr string
		want codes.Code
	}{
		{"", codes.InvalidArgument},
		{"not-an-address", codes.InvalidArgument},
		{sdk.AccAddress(cons(4)).String(), codes.InvalidArgument},
		{sdk.ValAddress(cons(5)).String(), codes.NotFound}, // operator unknown to staking
		{valA.OperatorAddress, codes.NotFound},             // known validator, no record yet
		{consStr(6), codes.NotFound},
	}
	for _, c := range cases {
		_, err := q.Reputation(ctx, &types.QueryReputationRequest{ValidatorAddress: c.addr})
		if status.Code(err) != c.want {
			t.Fatalf("%q: want %v, got %v", c.addr, c.want, err)
		}
	}
	if _, err := q.Reputation(ctx, nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("nil request: %v", err)
	}
}

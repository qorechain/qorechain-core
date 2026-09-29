package license

import (
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// fakeLicenseKeeper serves a fixed set of grants to the query server.
type fakeLicenseKeeper struct {
	LicenseKeeper
	licenses []types.License
}

func (f fakeLicenseKeeper) GetLicense(_ sdk.Context, grantee, featureID string) (types.License, error) {
	for _, l := range f.licenses {
		if l.Grantee == grantee && l.FeatureID == featureID {
			return l, nil
		}
	}
	return types.License{}, types.ErrLicenseNotFound
}

func (f fakeLicenseKeeper) GetLicenses(_ sdk.Context, grantee string) ([]types.License, error) {
	var out []types.License
	for _, l := range f.licenses {
		if l.Grantee == grantee {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f fakeLicenseKeeper) GetLicenseHolders(_ sdk.Context, featureID string) ([]types.License, error) {
	var out []types.License
	for _, l := range f.licenses {
		if l.FeatureID == featureID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (f fakeLicenseKeeper) Logger() log.Logger { return log.NewNopLogger() }

// TestQueryActiveHonoursExpiry pins QSR-2026-0041: Check, Holders and List
// report a grant active only while it is not suspended and not expired at the
// served height, the same predicate the on-chain gate uses.
func TestQueryActiveHonoursExpiry(t *testing.T) {
	const height = 1000
	const feature = "validator_operator"
	grants := []types.License{
		{Grantee: "valid", FeatureID: feature, ExpiresAt: height + 1},
		{Grantee: "perpetual", FeatureID: feature, ExpiresAt: 0},
		{Grantee: "expires-now", FeatureID: feature, ExpiresAt: height},
		{Grantee: "expired", FeatureID: feature, ExpiresAt: height - 50},
		{Grantee: "suspended", FeatureID: feature, ExpiresAt: 0, Suspended: true},
		{Grantee: "suspended-expired", FeatureID: feature, ExpiresAt: height - 1, Suspended: true},
	}
	want := map[string]bool{
		"valid":             true,
		"perpetual":         true,
		"expires-now":       false,
		"expired":           false,
		"suspended":         false,
		"suspended-expired": false,
	}

	q := NewQueryServer(fakeLicenseKeeper{licenses: grants})
	ctx := sdk.NewContext(nil, cmtproto.Header{Height: height}, false, log.NewNopLogger())

	for grantee, active := range want {
		resp, err := q.Check(ctx, &types.QueryCheckRequest{Grantee: grantee, FeatureId: feature})
		if err != nil {
			t.Fatalf("Check(%s): %v", grantee, err)
		}
		if resp.Active != active {
			t.Errorf("Check(%s).Active = %v, want %v", grantee, resp.Active, active)
		}
		if resp.License.Active != active {
			t.Errorf("Check(%s).License.Active = %v, want %v", grantee, resp.License.Active, active)
		}
		list, err := q.List(ctx, &types.QueryListRequest{Grantee: grantee})
		if err != nil || len(list.Licenses) != 1 {
			t.Fatalf("List(%s): %v %v", grantee, list, err)
		}
		if list.Licenses[0].Active != active {
			t.Errorf("List(%s).Active = %v, want %v", grantee, list.Licenses[0].Active, active)
		}
	}

	holders, err := q.Holders(ctx, &types.QueryHoldersRequest{FeatureId: feature})
	if err != nil {
		t.Fatalf("Holders: %v", err)
	}
	if len(holders.Licenses) != len(grants) {
		t.Fatalf("Holders returned %d rows, want %d", len(holders.Licenses), len(grants))
	}
	for _, v := range holders.Licenses {
		if v.Active != want[v.Grantee] {
			t.Errorf("Holders[%s].Active = %v, want %v", v.Grantee, v.Active, want[v.Grantee])
		}
	}
}

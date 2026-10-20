package app

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"cosmossdk.io/core/address"
)

type liabilityAccounts struct{ distrtypes.AccountKeeper }

func (liabilityAccounts) GetModuleAddress(name string) sdk.AccAddress {
	return authtypes.NewModuleAddress(name)
}

type liabilityStaking struct {
	distrtypes.StakingKeeper
	valCodec address.Codec
}

func (s liabilityStaking) ValidatorAddressCodec() address.Codec { return s.valCodec }

// QSR-2026-0341: the SDK credits a validator's whole allocation, commission
// included, to its outstanding rewards and also records the commission slice
// as accumulated commission. ClaimsOwed counts it once; TotalOwed (kept for
// the v3.2.0 handler's replay) counts it twice.
func TestDistributionLiabilityCountsCommissionOnce(t *testing.T) {
	key := storetypes.NewKVStoreKey(distrtypes.StoreKey)
	tctx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := tctx.Ctx

	valCodec := addresscodec.NewBech32Codec("qorvaloper")
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := distrkeeper.NewKeeper(cdc, runtime.NewKVStoreService(key),
		liabilityAccounts{}, nil, liabilityStaking{valCodec: valCodec},
		authtypes.FeeCollectorName, authtypes.NewModuleAddress("gov").String())

	pool := sdk.NewDecCoins(sdk.NewDecCoin("uqor", math.NewInt(7)))
	if err := k.FeePool.Set(ctx, distrtypes.FeePool{CommunityPool: pool}); err != nil {
		t.Fatal(err)
	}

	valBz := sdk.ValAddress([]byte("validator-operator-1"))
	valStr, err := valCodec.BytesToString(valBz)
	if err != nil {
		t.Fatal(err)
	}
	if err := k.SetValidatorCurrentRewards(ctx, valBz, distrtypes.NewValidatorCurrentRewards(sdk.DecCoins{}, 1)); err != nil {
		t.Fatal(err)
	}
	if err := k.SetValidatorAccumulatedCommission(ctx, valBz, distrtypes.InitialValidatorAccumulatedCommission()); err != nil {
		t.Fatal(err)
	}
	if err := k.SetValidatorOutstandingRewards(ctx, valBz, distrtypes.ValidatorOutstandingRewards{Rewards: sdk.DecCoins{}}); err != nil {
		t.Fatal(err)
	}
	val := stakingtypes.Validator{
		OperatorAddress: valStr,
		Commission:      stakingtypes.NewCommission(math.LegacyNewDecWithPrec(10, 2), math.LegacyOneDec(), math.LegacyZeroDec()),
	}
	tokens := sdk.NewDecCoins(sdk.NewDecCoin("uqor", math.NewInt(1_000)))
	if err := k.AllocateTokensToValidator(ctx, val, tokens); err != nil {
		t.Fatal(err)
	}

	d := distributionLiability{k: k}
	// outstanding 1000 (commission 100 included) + pool 7.
	if got := d.ClaimsOwed(ctx); !got.Equal(math.NewInt(1_007)) {
		t.Fatalf("ClaimsOwed %s, want 1007 (outstanding + pool)", got)
	}
	// The v3.2.0 formula adds the commission a second time.
	if got := d.TotalOwed(ctx); !got.Equal(math.NewInt(1_107)) {
		t.Fatalf("TotalOwed %s, want 1107 (the historical double count)", got)
	}
}

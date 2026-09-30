package app

import (
	"context"
	"strings"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"google.golang.org/protobuf/proto"

	licensetypes "github.com/qorechain/qorechain-core/x/license/types"
	pqctypes "github.com/qorechain/qorechain-core/x/pqc/types"
)

// ---- fakes ----------------------------------------------------------------

type fakeAdmissionLicences map[string]bool

func (f fakeAdmissionLicences) HasActiveLicense(_ sdk.Context, grantee, featureID string) bool {
	return featureID == licensetypes.FeatureValidatorOperator && f[grantee]
}

type fakeAdmissionStaking struct {
	validators  map[string]stakingtypes.Validator
	delegations map[string]stakingtypes.Delegation // key: del|val
}

func (f *fakeAdmissionStaking) GetValidator(_ context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	v, ok := f.validators[addr.String()]
	if !ok {
		return v, stakingtypes.ErrNoValidatorFound
	}
	return v, nil
}

func (f *fakeAdmissionStaking) GetDelegation(_ context.Context, del sdk.AccAddress, val sdk.ValAddress) (stakingtypes.Delegation, error) {
	d, ok := f.delegations[del.String()+"|"+val.String()]
	if !ok {
		return d, stakingtypes.ErrNoDelegation
	}
	return d, nil
}

type fakeFeatures map[string]bool

func (f fakeFeatures) ProtocolFeatureEnabled(_ sdk.Context, name string) bool { return f[name] }

type fakeTx struct{ msgs []sdk.Msg }

func (t fakeTx) GetMsgs() []sdk.Msg                  { return t.msgs }
func (t fakeTx) GetMsgsV2() ([]proto.Message, error) { return nil, nil }

var (
	opVal   = sdk.ValAddress([]byte("operator____________"))
	opAcc   = sdk.AccAddress(opVal)
	otherAc = sdk.AccAddress([]byte("delegator___________"))
	qor     = func(n int64) math.Int { return math.NewInt(n).MulRaw(1_000_000) }
)

// A validator whose operator holds selfBond; tokens == shares so a share is a uqor.
func stakingWithSelfBond(selfBond math.Int) *fakeAdmissionStaking {
	total := selfBond.Add(qor(1_000_000))
	return &fakeAdmissionStaking{
		validators: map[string]stakingtypes.Validator{
			opVal.String(): {
				OperatorAddress: opVal.String(),
				Tokens:          total,
				DelegatorShares: math.LegacyNewDecFromInt(total),
			},
		},
		delegations: map[string]stakingtypes.Delegation{
			opAcc.String() + "|" + opVal.String(): {
				DelegatorAddress: opAcc.String(), ValidatorAddress: opVal.String(),
				Shares: math.LegacyNewDecFromInt(selfBond),
			},
			otherAc.String() + "|" + opVal.String(): {
				DelegatorAddress: otherAc.String(), ValidatorAddress: opVal.String(),
				Shares: math.LegacyNewDecFromInt(qor(1_000_000)),
			},
		},
	}
}

func createValidatorMsg(value math.Int) *stakingtypes.MsgCreateValidator {
	return &stakingtypes.MsgCreateValidator{
		ValidatorAddress: opVal.String(),
		Value:            sdk.NewCoin("uqor", value),
	}
}

func undelegate(del sdk.AccAddress, amount math.Int) *stakingtypes.MsgUndelegate {
	return &stakingtypes.MsgUndelegate{DelegatorAddress: del.String(), ValidatorAddress: opVal.String(), Amount: sdk.NewCoin("uqor", amount)}
}

func redelegate(del sdk.AccAddress, amount math.Int) *stakingtypes.MsgBeginRedelegate {
	return &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress: del.String(), ValidatorSrcAddress: opVal.String(),
		ValidatorDstAddress: sdk.ValAddress([]byte("elsewhere___________")).String(),
		Amount:              sdk.NewCoin("uqor", amount),
	}
}

func testCtx(t *testing.T, height int64) sdk.Context {
	t.Helper()
	key := storetypes.NewKVStoreKey("admission")
	return testutil.DefaultContext(key, storetypes.NewTransientStoreKey("t_admission")).WithBlockHeight(height)
}

// ---- the rule ---------------------------------------------------------------

func TestCheckValidatorAdmission_CreateValidator(t *testing.T) {
	// Every case is judged the same by both versions of the rule: the switch
	// changes only what a full exit does.
	ctx := testCtx(t, 10)
	licensed := fakeAdmissionLicences{opAcc.String(): true}
	unlicensed := fakeAdmissionLicences{}
	for _, tc := range []struct {
		name    string
		lk      fakeAdmissionLicences
		value   math.Int
		wantErr string
	}{
		{"licensed, above floor", licensed, qor(100_001), ""},
		{"licensed, at floor", licensed, qor(100_000), ""},
		{"licensed, below floor", licensed, qor(100_000).SubRaw(1), "below required minimum"},
		{"licensed, 1 QOR", licensed, qor(1), "below required minimum"},
		{"unlicensed, above floor", unlicensed, qor(900_000), "validator_operator"},
		{"unlicensed, below floor", unlicensed, qor(1), "below required minimum"},
	} {
		for _, exit := range []selfBondExit{exitRefused, exitAllowed} {
			err := checkValidatorAdmission(ctx, tc.lk, stakingWithSelfBond(qor(0)), createValidatorMsg(tc.value), exit)
			if tc.wantErr == "" && err != nil {
				t.Errorf("%s (exit %v): unexpected refusal %v", tc.name, exit, err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("%s (exit %v): got %v, want an error containing %q", tc.name, exit, err, tc.wantErr)
			}
		}
	}
}

func TestCheckValidatorAdmission_OperatingFloor(t *testing.T) {
	ctx := testCtx(t, 10)
	lk := fakeAdmissionLicences{}
	for _, tc := range []struct {
		name       string
		selfBond   math.Int
		msg        sdk.Msg
		refusedOld bool // v3.2.0 rule, below the switch
		refusedNew bool // from the switch
	}{
		{"operator leaves the floor exactly", qor(150_000), undelegate(opAcc, qor(50_000)), false, false},
		{"operator goes 1 uqor under the floor", qor(150_000), undelegate(opAcc, qor(50_000).AddRaw(1)), true, true},
		{"operator full exit", qor(100_000), undelegate(opAcc, qor(100_000)), true, false},
		{"operator full exit by redelegation", qor(250_000), redelegate(opAcc, qor(250_000)), true, false},
		{"operator partial redelegation under the floor", qor(250_000), redelegate(opAcc, qor(200_000)), true, true},
		{"slashed operator under the floor, partial", qor(99_000), undelegate(opAcc, qor(1_000)), true, true},
		{"slashed operator under the floor, full exit", qor(99_000), undelegate(opAcc, qor(99_000)), true, false},
		{"ordinary delegator is never limited", qor(100_000), undelegate(otherAc, qor(1_000_000)), false, false},
		{"ordinary delegator redelegation", qor(100_000), redelegate(otherAc, qor(999_999)), false, false},
	} {
		sk := stakingWithSelfBond(tc.selfBond)
		errOld := checkValidatorAdmission(ctx, lk, sk, tc.msg, exitRefused)
		errNew := checkValidatorAdmission(ctx, lk, sk, tc.msg, exitAllowed)
		if (errOld != nil) != tc.refusedOld {
			t.Errorf("%s: old rule refused=%v (%v), want %v", tc.name, errOld != nil, errOld, tc.refusedOld)
		}
		if (errNew != nil) != tc.refusedNew {
			t.Errorf("%s: new rule refused=%v (%v), want %v", tc.name, errNew != nil, errNew, tc.refusedNew)
		}
	}
}

// ---- the ante decorator: old behaviour below the switch, new from it --------

func runLicenseAnte(t *testing.T, ctx sdk.Context, d ValidatorLicenseDecorator, msgs ...sdk.Msg) error {
	t.Helper()
	next := func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil }
	_, err := d.AnteHandle(ctx, fakeTx{msgs: msgs}, false, next)
	return err
}

func TestValidatorLicenseDecorator_FeatureSwitch(t *testing.T) {
	ctx := testCtx(t, 10)
	full := undelegate(opAcc, qor(100_000))

	off := ValidatorLicenseDecorator{stakingKeeper: stakingWithSelfBond(qor(100_000)), features: fakeFeatures{}}
	if err := runLicenseAnte(t, ctx, off, full); err == nil {
		t.Fatal("feature off: a full exit must still be refused (v3.2.0 replay)")
	}
	on := ValidatorLicenseDecorator{
		stakingKeeper: stakingWithSelfBond(qor(100_000)),
		features:      fakeFeatures{pqctypes.FeatureStakingAdmissionGate: true},
	}
	if err := runLicenseAnte(t, ctx, on, full); err != nil {
		t.Fatalf("feature on: full exit refused: %v", err)
	}
	// A partial withdrawal below the floor stays refused on both sides, nested
	// in authz too.
	exec := authz.NewMsgExec(opAcc, []sdk.Msg{undelegate(opAcc, qor(1))})
	for name, d := range map[string]ValidatorLicenseDecorator{"off": off, "on": on} {
		if err := runLicenseAnte(t, ctx, d, &exec); err == nil {
			t.Errorf("feature %s: authz-nested partial undelegation below the floor admitted", name)
		}
	}
}

func TestValidatorLicenseDecorator_CreateValidatorBothSides(t *testing.T) {
	ctx := testCtx(t, 10)
	for name, f := range map[string]fakeFeatures{"off": {}, "on": {pqctypes.FeatureStakingAdmissionGate: true}} {
		unlicensed := ValidatorLicenseDecorator{licenseKeeper: fakeAdmissionLicences{}, features: f}
		licensed := ValidatorLicenseDecorator{licenseKeeper: fakeAdmissionLicences{opAcc.String(): true}, features: f}
		if err := runLicenseAnte(t, ctx, unlicensed, createValidatorMsg(qor(200_000))); err == nil {
			t.Errorf("feature %s: unlicensed creation admitted", name)
		}
		exec := authz.NewMsgExec(opAcc, []sdk.Msg{createValidatorMsg(qor(200_000))})
		if err := runLicenseAnte(t, ctx, unlicensed, &exec); err == nil {
			t.Errorf("feature %s: authz-nested unlicensed creation admitted", name)
		}
		if err := runLicenseAnte(t, ctx, licensed, createValidatorMsg(qor(99_999))); err == nil {
			t.Errorf("feature %s: licensed creation under the floor admitted", name)
		}
		if err := runLicenseAnte(t, ctx, licensed, createValidatorMsg(qor(100_000))); err != nil {
			t.Errorf("feature %s: licensed creation at the floor refused: %v", name, err)
		}
	}
}

func TestValidatorLicenseDecorator_GenesisExemption(t *testing.T) {
	msg := createValidatorMsg(qor(1)) // unlicensed and under the floor
	d := ValidatorLicenseDecorator{
		licenseKeeper: fakeAdmissionLicences{},
		features:      fakeFeatures{pqctypes.FeatureStakingAdmissionGate: true},
	}

	if err := runLicenseAnte(t, testCtx(t, 0), d, msg); err != nil {
		t.Fatalf("height-0 gentx refused: %v", err)
	}
	// initial_height > 1: InitChain runs at that height.
	inGenesis := true
	d.executingGenesis = func(sdk.Context) bool { return inGenesis }
	if err := runLicenseAnte(t, testCtx(t, 1000), d, msg); err != nil {
		t.Fatalf("gentx at initial_height 1000 refused: %v", err)
	}
	inGenesis = false
	if err := runLicenseAnte(t, testCtx(t, 1000), d, msg); err == nil {
		t.Fatal("the same message outside InitChain was admitted")
	}
}

func TestExecutingGenesisFlag(t *testing.T) {
	app := &QoreChainApp{}
	if app.executingGenesis(testCtx(t, 5)) {
		t.Fatal("height 5 outside InitChain reported as genesis")
	}
	if !app.executingGenesis(testCtx(t, 0)) {
		t.Fatal("height 0 not reported as genesis")
	}
	app.inInitChain = true
	if !app.executingGenesis(testCtx(t, 5)) {
		t.Fatal("InitChain at initial height 5 not reported as genesis")
	}
}

// The gate is inert with the feature off: no refusal whatever the message.
func TestStakingAdmissionGate_OffBelowSwitch(t *testing.T) {
	app := &QoreChainApp{}
	ctx := testCtx(t, 10)
	for _, msg := range []sdk.Msg{createValidatorMsg(qor(1)), undelegate(opAcc, qor(1))} {
		if err := app.stakingAdmissionGate(ctx, msg); err != nil {
			t.Fatalf("gate refused with no feature record: %v", err)
		}
	}
}

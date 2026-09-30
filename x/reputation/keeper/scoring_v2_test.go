package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// ---- fixtures ----

type testVal struct {
	cons  []byte
	power int64
}

func (v testVal) Address() []byte { return v.cons }
func (v testVal) Power() int64    { return v.power }

type testVote struct {
	val  testVal
	flag comet.BlockIDFlag
}

func (v testVote) Validator() comet.Validator        { return v.val }
func (v testVote) GetBlockIDFlag() comet.BlockIDFlag { return v.flag }

type testVotes []testVote

func (v testVotes) Len() int                 { return len(v) }
func (v testVotes) Get(i int) comet.VoteInfo { return v[i] }

type testCommit struct{ votes testVotes }

func (c testCommit) Round() int32           { return 0 }
func (c testCommit) Votes() comet.VoteInfos { return c.votes }

type testBlockInfo struct{ commit testCommit }

func (b testBlockInfo) GetEvidence() comet.EvidenceList { return nil }
func (b testBlockInfo) GetValidatorsHash() []byte       { return nil }
func (b testBlockInfo) GetProposerAddress() []byte      { return nil }
func (b testBlockInfo) GetLastCommit() comet.CommitInfo { return b.commit }

// fakeStaking is a StakingReader over a fixed validator list.
type fakeStaking struct {
	vals []stakingtypes.Validator // bonded ones are the ones with Status Bonded
}

func (f *fakeStaking) GetValidator(_ context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	for _, v := range f.vals {
		if v.OperatorAddress == addr.String() {
			return v, nil
		}
	}
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

func (f *fakeStaking) GetValidatorByConsAddr(_ context.Context, cons sdk.ConsAddress) (stakingtypes.Validator, error) {
	for _, v := range f.vals {
		c, _ := v.GetConsAddr()
		if bytes.Equal(c, cons) {
			return v, nil
		}
	}
	return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
}

func (f *fakeStaking) IterateBondedValidatorsByPower(_ context.Context, fn func(int64, stakingtypes.ValidatorI) bool) error {
	for i, v := range f.vals {
		if v.Status != stakingtypes.Bonded {
			continue
		}
		if fn(int64(i), v) {
			return nil
		}
	}
	return nil
}

// fakeSlashing serves signing info per consensus address.
type fakeSlashing struct {
	window int64
	missed map[string]int64
}

func (f *fakeSlashing) GetValidatorSigningInfo(_ context.Context, cons sdk.ConsAddress) (slashingtypes.ValidatorSigningInfo, error) {
	m, ok := f.missed[cons.String()]
	if !ok {
		return slashingtypes.ValidatorSigningInfo{}, slashingtypes.ErrNoSigningInfoFound
	}
	return slashingtypes.ValidatorSigningInfo{Address: cons.String(), MissedBlocksCounter: m}, nil
}

func (f *fakeSlashing) SignedBlocksWindow(context.Context) (int64, error) { return f.window, nil }

// bondedValidator is validator() with tokens.
func bondedValidator(t *testing.T, n byte, status stakingtypes.BondStatus, tokens int64) (stakingtypes.Validator, []byte) {
	t.Helper()
	v, c := validator(t, n, status)
	v.Tokens = sdkmath.NewInt(tokens)
	return v, c
}

// validator builds a staking validator whose consensus address is cons.
func validator(t *testing.T, n byte, status stakingtypes.BondStatus) (stakingtypes.Validator, []byte) {
	t.Helper()
	pk := ed25519.GenPrivKeyFromSecret([]byte{n}).PubKey()
	cons := []byte(pk.Address())
	anyPk, err := codectypes.NewAnyWithValue(pk)
	if err != nil {
		t.Fatal(err)
	}
	return stakingtypes.Validator{
		OperatorAddress: sdk.ValAddress(bytes.Repeat([]byte{0xA0 + n}, 20)).String(),
		ConsensusPubkey: anyPk,
		Status:          status,
	}, cons
}

func setupRepV2(t *testing.T, on bool) (*Keeper, sdk.Context, *storetypes.KVStoreKey) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	ms.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := ms.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	k := NewKeeper(cdc, storeKey, nil, log.NewNopLogger())
	k.SetFeatureGate(func(sdk.Context) bool { return on })
	ctx := sdk.NewContext(ms, cmtproto.Header{Height: 100, ProposerAddress: bytes.Repeat([]byte{1}, 20)}, false, log.NewNopLogger()).
		WithBlockTime(time.Unix(1_700_000_000, 0))
	return &k, ctx, storeKey
}

func withCommit(ctx sdk.Context, height int64, votes ...testVote) sdk.Context {
	return ctx.WithBlockHeight(height).WithCometInfo(testBlockInfo{commit: testCommit{votes: votes}})
}

func cons(n byte) []byte { return bytes.Repeat([]byte{n}, 20) }

func consStr(n byte) string { return sdk.ConsAddress(cons(n)).String() }

// dumpStore returns every key/value in the store, sorted.
func dumpStore(ctx sdk.Context, key storetypes.StoreKey) string {
	st := ctx.KVStore(key)
	it := st.Iterator(nil, nil)
	defer it.Close()
	var lines []string
	for ; it.Valid(); it.Next() {
		lines = append(lines, fmt.Sprintf("%x=%x", it.Key(), it.Value()))
	}
	sort.Strings(lines)
	return fmt.Sprint(lines)
}

// ---- tests ----

// Feature off: the EndBlocker is the historical one, including charging a
// stored validator that is not in the commit a missed block every block.
func TestReputationV2OffKeepsLegacyBehaviour(t *testing.T) {
	k, ctx, key := setupRepV2(t, false)
	retired := types.ValidatorReputation{Address: consStr(9), JoinedAtHeight: 1, UptimeBlocks: 10}
	k.SetValidatorReputation(ctx, retired)

	// Run the same blocks through a keeper with no gate at all (the v3.2.1
	// code path) and compare the full store: byte-identical.
	k2, ctx2, key2 := setupRepV2(t, false)
	k2.SetFeatureGate(nil)
	k2.SetValidatorReputation(ctx2, retired)

	for h := int64(101); h <= 105; h++ {
		votes := []testVote{
			{testVal{cons(1), 100}, comet.BlockIDFlagCommit},
			{testVal{cons(2), 50}, comet.BlockIDFlagAbsent},
		}
		if err := k.EndBlocker(withCommit(ctx, h, votes...)); err != nil {
			t.Fatal(err)
		}
		if err := k2.EndBlocker(withCommit(ctx2, h, votes...)); err != nil {
			t.Fatal(err)
		}
	}
	if dumpStore(ctx, key) != dumpStore(ctx2, key2) {
		t.Fatal("feature off must be byte-identical to the ungated keeper")
	}
	got, _ := k.GetValidatorReputation(ctx, consStr(9))
	if got.MissedBlocks != 5 {
		t.Fatalf("legacy: retired validator missed = %d, want 5 (the historical fake misses)", got.MissedBlocks)
	}
	if got.StakeScore != "0.000000000000000000" {
		t.Fatalf("legacy: stake score = %q, want zero (never fed)", got.StakeScore)
	}
}

// epochCtx is ctx at an epoch boundary (the default 100 blocks).
func epochCtx(ctx sdk.Context, epoch int64) sdk.Context {
	return ctx.WithBlockHeight(epoch * types.DefaultEpochBlocks)
}

func TestReputationV2NonEpochBlockWritesOnlyTheProposer(t *testing.T) {
	k, ctx, key := setupRepV2(t, true)
	valA, _ := bondedValidator(t, 1, stakingtypes.Bonded, 100)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	// 5,000 records of validators long gone from the set.
	for i := 0; i < 5000; i++ {
		k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: fmt.Sprintf("old%05d", i), JoinedAtHeight: 1, CompositeScore: "0.3"})
	}
	before := dumpStore(ctx, key)

	// The store work of one non-epoch block does not depend on how many
	// records exist: the same block on an empty store costs the same.
	gm := storetypes.NewInfiniteGasMeter()
	if err := k.EndBlocker(ctx.WithBlockHeight(101).WithGasMeter(gm)); err != nil {
		t.Fatal(err)
	}
	kEmpty, ctxEmpty, _ := setupRepV2(t, true)
	gmEmpty := storetypes.NewInfiniteGasMeter()
	if err := kEmpty.EndBlocker(ctxEmpty.WithBlockHeight(101).WithGasMeter(gmEmpty)); err != nil {
		t.Fatal(err)
	}
	if gm.GasConsumed() != gmEmpty.GasConsumed() {
		t.Fatalf("a non-epoch block cost %d with 5,000 old records and %d without; want O(1)", gm.GasConsumed(), gmEmpty.GasConsumed())
	}
	after := dumpStore(ctx, key)
	if before == after {
		t.Fatal("proposer's block not counted")
	}
	prop, found := k.GetValidatorReputation(ctx, sdk.ConsAddress(bytes.Repeat([]byte{1}, 20)).String())
	if !found || prop.ProposedBlocks != 1 {
		t.Fatalf("proposer record %+v", prop)
	}
	old, _ := k.GetValidatorReputation(ctx, "old00042")
	if old.MissedBlocks != 0 || old.LastUpdatedHeight != 0 {
		t.Fatalf("an old record was touched: %+v", old)
	}
}

func TestReputationV2EpochRescoresTheBondedSet(t *testing.T) {
	k, ctx, key := setupRepV2(t, true)
	valA, consA := bondedValidator(t, 1, stakingtypes.Bonded, 600) // twice the equal share
	valB, consB := bondedValidator(t, 2, stakingtypes.Bonded, 100) // a third of it
	valC, consC := bondedValidator(t, 3, stakingtypes.Bonded, 200) // two thirds of the equal share
	valD, consD := bondedValidator(t, 4, stakingtypes.Unbonded, 900)
	// Bonded total 900 over 3 validators: equal share 300.
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA, valB, valC, valD}})
	k.SetSlashingReader(&fakeSlashing{window: 1000, missed: map[string]int64{
		sdk.ConsAddress(consA).String(): 0,
		sdk.ConsAddress(consB).String(): 250,
	}})
	retired := types.ValidatorReputation{Address: sdk.ConsAddress(consD).String(), JoinedAtHeight: 1, MissedBlocks: 3_500_000, CompositeScore: "0.1", InactiveSince: 50}
	k.SetValidatorReputation(ctx, retired)
	retiredBz := ctx.KVStore(key).Get(validatorKey(retired.Address))

	if err := k.EndBlocker(epochCtx(ctx, 3)); err != nil {
		t.Fatal(err)
	}

	a, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String())
	b, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consB).String())
	c, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consC).String())
	// S = min(1, tokens / equal share).
	if !a.GetStakeScoreDec().Equal(sdkmath.LegacyOneDec()) {
		t.Fatalf("S(2x share) = %s, want 1 (capped)", a.StakeScore)
	}
	if !b.GetStakeScoreDec().Equal(sdkmath.LegacyOneDec().Quo(sdkmath.LegacyNewDec(3))) {
		t.Fatalf("S(1/3 share) = %s", b.StakeScore)
	}
	if !c.GetStakeScoreDec().Equal(sdkmath.LegacyNewDec(2).Quo(sdkmath.LegacyNewDec(3))) {
		t.Fatalf("S(2/3 share) = %s", c.StakeScore)
	}
	// P = 1 - missed / window; neutral without signing info.
	if !a.GetPerformanceScoreDec().Equal(sdkmath.LegacyOneDec()) ||
		!b.GetPerformanceScoreDec().Equal(sdkmath.LegacyNewDecWithPrec(75, 2)) ||
		!c.GetPerformanceScoreDec().Equal(sdkmath.LegacyNewDecWithPrec(5, 1)) {
		t.Fatalf("P: a=%s b=%s c=%s", a.PerformanceScore, b.PerformanceScore, c.PerformanceScore)
	}
	// C starts neutral; the lifetime counters are not advanced.
	if !a.GetContributionScoreDec().Equal(sdkmath.LegacyNewDecWithPrec(5, 1)) || a.UptimeBlocks != 0 || a.MissedBlocks != 0 {
		t.Fatalf("a = %+v", a)
	}
	// One history entry per bonded validator, none for the unbonded one,
	// whose record is untouched.
	if !bytes.Equal(ctx.KVStore(key).Get(validatorKey(retired.Address)), retiredBz) {
		t.Fatal("a record outside the bonded set was rewritten")
	}
	for addr, want := range map[string]int{a.Address: 1, b.Address: 1, c.Address: 1, retired.Address: 0} {
		if got := historyCount(ctx, key, addr); got != want {
			t.Fatalf("%s history entries %d, want %d", addr, got, want)
		}
	}

	// A composite with S and C fed can pass what the half-fed formula
	// allowed (beta x 1 + delta x T <= 0.5).
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: a.Address, JoinedAtHeight: 1, GovParticipation: "1.0", CompositeScore: "0.9"})
	if err := k.EndBlocker(epochCtx(ctx, 500)); err != nil {
		t.Fatal(err)
	}
	a, _ = k.GetValidatorReputation(ctx, a.Address)
	if !a.GetCompositeScoreDec().GT(sdkmath.LegacyNewDecWithPrec(5, 1)) {
		t.Fatalf("composite %s, want > 0.5", a.CompositeScore)
	}
}

func historyCount(ctx sdk.Context, key storetypes.StoreKey, addr string) int {
	it := storetypes.KVStorePrefixIterator(ctx.KVStore(key), append(append([]byte{}, types.HistoryPrefix...), []byte(addr+"/")...))
	defer it.Close()
	n := 0
	for ; it.Valid(); it.Next() {
		n++
	}
	return n
}

func TestReputationV2BondedTransitionsAndSlashes(t *testing.T) {
	k, ctx, _ := setupRepV2(t, true)
	valA, consA := bondedValidator(t, 1, stakingtypes.Bonded, 100)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	addr := sdk.ConsAddress(consA).String()
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: addr, JoinedAtHeight: 1})
	opA, _ := sdk.ValAddressFromBech32(valA.OperatorAddress)
	h := NewHooks(k)

	gm := storetypes.NewGasMeter(1_000_000)
	hctx := ctx.WithBlockHeight(777).WithGasMeter(gm)
	if err := h.AfterValidatorBeginUnbonding(hctx, sdk.ConsAddress(consA), opA); err != nil {
		t.Fatal(err)
	}
	if err := h.BeforeValidatorSlashed(hctx, opA, sdkmath.LegacyNewDecWithPrec(1, 2)); err != nil {
		t.Fatal(err)
	}
	if gm.GasConsumed() != 0 {
		t.Fatalf("hooks consumed %d gas; must be unmetered", gm.GasConsumed())
	}
	r, _ := k.GetValidatorReputation(ctx, addr)
	if r.InactiveSince != 777 || r.SlashingEvents != 1 {
		t.Fatalf("after unbonding and a slash: %+v", r)
	}
	if err := h.AfterValidatorBonded(hctx, sdk.ConsAddress(consA), opA); err != nil {
		t.Fatal(err)
	}
	r, _ = k.GetValidatorReputation(ctx, addr)
	if r.InactiveSince != 0 {
		t.Fatalf("re-bonded but still inactive: %+v", r)
	}
}

func TestMarkInactiveRecords(t *testing.T) {
	k, ctx, _ := setupRepV2(t, true)
	valA, consA := bondedValidator(t, 1, stakingtypes.Bonded, 100)
	valB, consB := bondedValidator(t, 2, stakingtypes.Unbonding, 100)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA, valB}})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consA).String()})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consB).String(), MissedBlocks: 99})
	n, err := k.MarkInactiveRecords(ctx)
	if err != nil || n != 1 {
		t.Fatalf("marked %d, %v; want 1", n, err)
	}
	a, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String())
	b, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consB).String())
	if a.InactiveSince != 0 || b.InactiveSince != ctx.BlockHeight() || b.MissedBlocks != 99 {
		t.Fatalf("a=%+v b=%+v", a, b)
	}
}

func TestReputationV2GovParticipation(t *testing.T) {
	k, ctx, key := setupRepV2(t, true)
	valA, consA := validator(t, 1, stakingtypes.Bonded)
	valB, consB := validator(t, 2, stakingtypes.Bonded)
	valC, _ := validator(t, 3, stakingtypes.Unbonded)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA, valB, valC}})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consA).String(), JoinedAtHeight: 1})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consB).String(), JoinedAtHeight: 1})

	opA, _ := sdk.ValAddressFromBech32(valA.OperatorAddress)
	opC, _ := sdk.ValAddressFromBech32(valC.OperatorAddress)
	h := NewHooks(k)

	// Proposal 1: A votes (twice, re-vote), a plain account votes, C (not
	// bonded) votes. Metered gas must not move: the bookkeeping is unmetered.
	gm := storetypes.NewGasMeter(1_000_000)
	vctx := ctx.WithGasMeter(gm)
	for _, voter := range []sdk.AccAddress{sdk.AccAddress(opA), sdk.AccAddress(opA), sdk.AccAddress(bytes.Repeat([]byte{7}, 20)), sdk.AccAddress(opC)} {
		if err := h.AfterProposalVote(vctx, 1, voter); err != nil {
			t.Fatal(err)
		}
	}
	if gm.GasConsumed() != 0 {
		t.Fatalf("vote hook consumed %d gas; must be unmetered", gm.GasConsumed())
	}
	if err := h.AfterProposalVotingPeriodEnded(ctx, 1); err != nil {
		t.Fatal(err)
	}
	a, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String())
	b, _ := k.GetValidatorReputation(ctx, sdk.ConsAddress(consB).String())
	if a.GovParticipation != sdkmath.LegacyOneDec().String() || b.GovParticipation != sdkmath.LegacyZeroDec().String() {
		t.Fatalf("after proposal 1: A=%q B=%q", a.GovParticipation, b.GovParticipation)
	}
	// Vote marks for proposal 1 are consumed.
	it := storetypes.KVStorePrefixIterator(ctx.KVStore(key), govVoteProposalPrefix(1))
	if it.Valid() {
		t.Fatal("vote marks left behind")
	}
	it.Close()

	// Proposal 2: nobody votes. A = 1/2.
	if err := h.AfterProposalVotingPeriodEnded(ctx, 2); err != nil {
		t.Fatal(err)
	}
	a, _ = k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String())
	if a.GovParticipation != sdkmath.LegacyNewDecWithPrec(5, 1).String() {
		t.Fatalf("after proposal 2: A=%q", a.GovParticipation)
	}

	// The window keeps the last GovParticipationWindow proposals only.
	for p := uint64(3); p < 3+GovParticipationWindow; p++ {
		if err := h.AfterProposalVotingPeriodEnded(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	a, _ = k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String())
	if a.GovParticipation != sdkmath.LegacyZeroDec().String() {
		t.Fatalf("window did not roll: A=%q", a.GovParticipation)
	}
	if n := len(ctx.KVStore(key).Get(govParticipationKey(opA))); n != GovParticipationWindow {
		t.Fatalf("window length %d", n)
	}
}

func TestReputationV2HooksNoOpWhenOff(t *testing.T) {
	k, ctx, key := setupRepV2(t, false)
	valA, consA := validator(t, 1, stakingtypes.Bonded)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consA).String(), JoinedAtHeight: 1})
	before := dumpStore(ctx, key)
	opA, _ := sdk.ValAddressFromBech32(valA.OperatorAddress)
	h := NewHooks(k)
	_ = h.AfterProposalVote(ctx, 1, sdk.AccAddress(opA))
	_ = h.AfterProposalVotingPeriodEnded(ctx, 1)
	_ = h.AfterValidatorRemoved(ctx, sdk.ConsAddress(consA), opA)
	if dumpStore(ctx, key) != before {
		t.Fatal("hooks wrote state with the feature off")
	}
}

func TestReputationV2RemovedValidatorIsDeleted(t *testing.T) {
	k, ctx, key := setupRepV2(t, true)
	valA, consA := bondedValidator(t, 1, stakingtypes.Bonded, 10)
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	if err := k.EndBlocker(epochCtx(ctx, 2).WithBlockHeader(cmtproto.Header{Height: 200})); err != nil {
		t.Fatal(err)
	}
	opA, _ := sdk.ValAddressFromBech32(valA.OperatorAddress)
	if err := NewHooks(k).AfterValidatorRemoved(ctx, sdk.ConsAddress(consA), opA); err != nil {
		t.Fatal(err)
	}
	if _, found := k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String()); found {
		t.Fatal("record survived removal")
	}
	if dump := dumpStore(ctx, key); dump != "[]" {
		t.Fatalf("state left behind: %s", dump)
	}
}

func TestPruneRemovedValidators(t *testing.T) {
	k, ctx, _ := setupRepV2(t, true)
	valA, consA := validator(t, 1, stakingtypes.Unbonded) // unbonded but still exists: kept
	k.SetStakingReader(&fakeStaking{vals: []stakingtypes.Validator{valA}})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: sdk.ConsAddress(consA).String()})
	k.SetValidatorReputation(ctx, types.ValidatorReputation{Address: consStr(9)}) // gone
	k.RecordHistoricalScore(ctx, consStr(9), "0.1")
	n, err := k.PruneRemovedValidators(ctx)
	if err != nil || n != 1 {
		t.Fatalf("pruned %d, %v; want 1", n, err)
	}
	if _, found := k.GetValidatorReputation(ctx, sdk.ConsAddress(consA).String()); !found {
		t.Fatal("existing validator's record was pruned")
	}
	if _, found := k.GetValidatorReputation(ctx, consStr(9)); found {
		t.Fatal("removed validator's record kept")
	}
}

// Records written before the new field existed serialise byte-identically.
func TestValidatorReputationJSONUnchangedWithoutParticipation(t *testing.T) {
	legacy := `{"address":"a","stake_score":"","performance_score":"","contribution_score":"","time_score":"","composite_score":"","last_updated_height":0,"uptime_blocks":0,"proposed_blocks":0,"missed_blocks":0,"slashing_events":0,"community_votes":0,"joined_at_height":0}`
	var r types.ValidatorReputation
	if err := json.Unmarshal([]byte(legacy), &r); err != nil {
		t.Fatal(err)
	}
	bz, _ := json.Marshal(r)
	if string(bz) != legacy {
		t.Fatalf("round trip changed bytes:\n%s\n%s", bz, legacy)
	}
}

package keeper

import (
	"context"
	"encoding/binary"
	"errors"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/qorechain/qorechain-core/x/reputation/types"
)

// Reputation scoring v2 (v3.2.2, QSR-2026-0030, -0312, -0130, -0145).
//
// The historical EndBlocker rescored and rewrote every record ever created,
// every block, and charged each validator missing from the last commit a missed
// block, including validators that had left the set long ago; their uptime
// ratio could never recover and their records and history churned forever. Two
// of the four formula components were never fed: StakeScore stayed zero and
// ContributionScore read a vote counter nothing wrote.
//
// With the switch on:
//   - Per block the only write is the proposer's ProposedBlocks count.
//   - Every EpochBlocks blocks (default 100) the bonded set, and only it, is
//     rescored from the chain's authoritative data, with one history entry
//     per validator per epoch:
//     P is the windowed signing ratio x/slashing keeps and enforces,
//     1 - MissedBlocksCounter / SignedBlocksWindow;
//     S is the validator's bonded tokens against an equal share of the bonded
//     total, min(1, tokens x N / total), so stake above the equal share does
//     not amplify reputation further;
//     C is the validator's governance participation: the share of the last
//     GovParticipationWindow proposals, among those that ended while it was
//     bonded, on which its operator voted (0.5 until it has any);
//     T is tenure from JoinedAtHeight, as before.
//   - A record outside the bonded set is not touched, so a validator that left
//     the set stops accruing anything; its record carries the height it left
//     (inactive_since). A validator removed from x/staking has its record and
//     history deleted. A slash increments the record's SlashingEvents.
//   - The lifetime UptimeBlocks and MissedBlocks counters are no longer
//     advanced; performance comes from x/slashing instead.
//
// Below the switch the historical code runs unchanged.

// GovParticipationWindow is how many of a validator's most recent eligible
// proposals its participation ratio is computed over.
const GovParticipationWindow = 10

var (
	// GovVotePrefix marks that a validator operator voted on a proposal:
	// prefix | proposalID (8 bytes, big-endian) | operator address bytes.
	// Consumed and deleted when the proposal's voting period ends.
	GovVotePrefix = []byte("reputation/govvote/")
	// GovParticipationPrefix holds a validator's participation window:
	// prefix | operator address bytes -> one byte per eligible proposal,
	// oldest first, 1 = voted, at most GovParticipationWindow bytes.
	GovParticipationPrefix = []byte("reputation/govpart/")
)

// neutralScore is P for a validator with no signing information and C for one
// with no eligible proposal yet.
var neutralScore = sdkmath.LegacyNewDecWithPrec(5, 1)

// StakingReader is what the scoring reads from x/staking.
type StakingReader interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error)
	IterateBondedValidatorsByPower(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
}

// SlashingReader is what the scoring reads from x/slashing.
type SlashingReader interface {
	GetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress) (slashingtypes.ValidatorSigningInfo, error)
	SignedBlocksWindow(ctx context.Context) (int64, error)
}

// SetFeatureGate installs the per-block switch for scoring v2. It must be
// called before the keeper is copied into the module, the hooks or QCA.
func (k *Keeper) SetFeatureGate(enabled func(ctx sdk.Context) bool) { k.scoringV2 = enabled }

// SetStakingReader replaces the staking reader (tests).
func (k *Keeper) SetStakingReader(r StakingReader) { k.staking = r }

// SetSlashingReader installs the slashing reader. Like the gate, it must be
// set before the keeper is copied.
func (k *Keeper) SetSlashingReader(r SlashingReader) { k.slashing = r }

func (k Keeper) scoringV2Enabled(ctx sdk.Context) bool {
	return k.scoringV2 != nil && k.scoringV2(ctx)
}

// endBlockerV2 counts the proposer's block, and rescores the bonded set at
// each epoch boundary.
func (k Keeper) endBlockerV2(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	if proposer := ctx.BlockHeader().ProposerAddress; proposer != nil {
		addr := sdk.ConsAddress(proposer).String()
		rep, found := k.GetValidatorReputation(ctx, addr)
		if !found {
			rep = newReputation(ctx, addr, params)
		}
		rep.ProposedBlocks++
		k.SetValidatorReputation(ctx, rep)
	}
	if ctx.BlockHeight()%params.ParamEpochBlocks() != 0 {
		return nil
	}
	return k.rescoreBondedSet(ctx, params)
}

func newReputation(ctx sdk.Context, addr string, params types.ReputationParams) types.ValidatorReputation {
	return types.ValidatorReputation{
		Address:        addr,
		JoinedAtHeight: ctx.BlockHeight(),
		CompositeScore: params.ParamMinScore().String(),
	}
}

type bondedEntry struct {
	cons   sdk.ConsAddress
	tokens sdkmath.Int
}

// rescoreBondedSet recomputes every bonded validator's record. The work is
// bounded by the bonded set (max_validators), not by the number of records.
func (k Keeper) rescoreBondedSet(ctx sdk.Context, params types.ReputationParams) error {
	if k.staking == nil {
		return nil
	}
	var (
		set   []bondedEntry
		total = sdkmath.ZeroInt()
	)
	err := k.staking.IterateBondedValidatorsByPower(ctx, func(_ int64, v stakingtypes.ValidatorI) bool {
		consBz, err := v.GetConsAddr()
		if err != nil || len(consBz) == 0 {
			return false
		}
		tokens := v.GetBondedTokens()
		set = append(set, bondedEntry{cons: sdk.ConsAddress(consBz), tokens: tokens})
		total = total.Add(tokens)
		return false
	})
	if err != nil {
		return err
	}
	if len(set) == 0 {
		return nil
	}
	window := int64(0)
	if k.slashing != nil {
		if w, err := k.slashing.SignedBlocksWindow(ctx); err == nil {
			window = w
		}
	}
	n := int64(len(set))
	for _, e := range set {
		addr := e.cons.String()
		rep, found := k.GetValidatorReputation(ctx, addr)
		if !found {
			rep = newReputation(ctx, addr, params)
		}
		rep.InactiveSince = 0
		S := stakeScore(e.tokens, total, n)
		P := k.signingScore(ctx, e.cons, window)
		C := contributionScoreV2(rep)
		T := k.calculateTimeScore(ctx, rep)
		rep = k.composeReputation(ctx, rep, params, S, P, C, T)
		k.SetValidatorReputation(ctx, rep)
		k.RecordHistoricalScore(ctx, rep.Address, rep.CompositeScore)
	}
	return nil
}

// stakeScore is min(1, tokens / (total / n)): 1 at or above an equal share of
// the bonded total, proportionally less below it.
func stakeScore(tokens, total sdkmath.Int, n int64) sdkmath.LegacyDec {
	if n <= 0 || tokens.IsNil() || total.IsNil() || !tokens.IsPositive() || !total.IsPositive() {
		return sdkmath.LegacyZeroDec()
	}
	s := sdkmath.LegacyNewDecFromInt(tokens.MulRaw(n)).Quo(sdkmath.LegacyNewDecFromInt(total))
	if s.GT(sdkmath.LegacyOneDec()) {
		return sdkmath.LegacyOneDec()
	}
	return s
}

// signingScore is 1 - missed / window from x/slashing's signing information,
// the windowed uptime the chain enforces. Neutral when it is unavailable.
func (k Keeper) signingScore(ctx sdk.Context, cons sdk.ConsAddress, window int64) sdkmath.LegacyDec {
	if k.slashing == nil || window <= 0 {
		return neutralScore
	}
	info, err := k.slashing.GetValidatorSigningInfo(ctx, cons)
	if err != nil {
		return neutralScore
	}
	missed := info.MissedBlocksCounter
	if missed < 0 {
		missed = 0
	}
	if missed > window {
		missed = window
	}
	return sdkmath.LegacyOneDec().Sub(sdkmath.LegacyNewDec(missed).Quo(sdkmath.LegacyNewDec(window)))
}

// contributionScoreV2 is the stored governance participation ratio, or the
// neutral 0.5 when the validator has had no eligible proposal.
func contributionScoreV2(rep types.ValidatorReputation) sdkmath.LegacyDec {
	if rep.GovParticipation == "" {
		return neutralScore
	}
	d, err := sdkmath.LegacyNewDecFromStr(rep.GovParticipation)
	if err != nil || d.IsNegative() {
		return neutralScore
	}
	if d.GT(sdkmath.LegacyOneDec()) {
		return sdkmath.LegacyOneDec()
	}
	return d
}

// calculateReputationV2WithParams is the reputation a reader sees between
// epochs: the stored components of the last rescore, decayed to this height.
func (k Keeper) calculateReputationV2WithParams(ctx sdk.Context, valAddr string, params types.ReputationParams) float64 {
	rep, found := k.GetValidatorReputation(ctx, valAddr)
	if !found {
		f, _ := params.ParamMinScore().Float64()
		return f
	}
	P := neutralScore
	if rep.PerformanceScore != "" {
		P = rep.GetPerformanceScoreDec()
	}
	updated := k.composeReputation(ctx, rep, params,
		rep.GetStakeScoreDec(), P, contributionScoreV2(rep), k.calculateTimeScore(ctx, rep))
	f, _ := updated.GetCompositeScoreDec().Float64()
	return f
}

// ---- Governance participation ----

func govVoteKey(proposalID uint64, valAddr sdk.ValAddress) []byte {
	key := make([]byte, 0, len(GovVotePrefix)+8+len(valAddr))
	key = append(key, GovVotePrefix...)
	key = binary.BigEndian.AppendUint64(key, proposalID)
	return append(key, valAddr...)
}

func govVoteProposalPrefix(proposalID uint64) []byte {
	key := make([]byte, 0, len(GovVotePrefix)+8)
	key = append(key, GovVotePrefix...)
	return binary.BigEndian.AppendUint64(key, proposalID)
}

func govParticipationKey(valAddr sdk.ValAddress) []byte {
	return append(append([]byte{}, GovParticipationPrefix...), valAddr...)
}

// unmetered returns ctx with an infinite gas meter. The reputation hooks run
// inside users' transactions (MsgVote, staking messages); their bounded
// bookkeeping must not change what those transactions cost.
func unmetered(ctx sdk.Context) sdk.Context {
	return ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
}

// RecordGovVote notes that voter voted on proposalID, when voter is a
// validator operator. No-op below the switch.
func (k Keeper) RecordGovVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress) {
	if !k.scoringV2Enabled(ctx) || k.staking == nil {
		return
	}
	ctx = unmetered(ctx)
	valAddr := sdk.ValAddress(voter)
	if _, err := k.staking.GetValidator(ctx, valAddr); err != nil {
		return
	}
	ctx.KVStore(k.storeKey).Set(govVoteKey(proposalID, valAddr), []byte{1})
}

// SettleGovParticipation runs when a proposal's voting period ends: every
// bonded validator gets one more entry in its participation window (voted or
// not), and its record's participation ratio is updated. The proposal's vote
// marks are deleted. No-op below the switch.
func (k Keeper) SettleGovParticipation(ctx sdk.Context, proposalID uint64) error {
	if !k.scoringV2Enabled(ctx) || k.staking == nil {
		return nil
	}
	ctx = unmetered(ctx)
	store := ctx.KVStore(k.storeKey)

	type bonded struct {
		val  sdk.ValAddress
		cons sdk.ConsAddress
	}
	var set []bonded
	err := k.staking.IterateBondedValidatorsByPower(ctx, func(_ int64, v stakingtypes.ValidatorI) bool {
		valBz, err := sdk.ValAddressFromBech32(v.GetOperator())
		if err != nil {
			return false
		}
		consBz, err := v.GetConsAddr()
		if err != nil {
			return false
		}
		set = append(set, bonded{val: valBz, cons: sdk.ConsAddress(consBz)})
		return false
	})
	if err != nil {
		return err
	}

	for _, b := range set {
		voted := store.Has(govVoteKey(proposalID, b.val))
		window := append([]byte{}, store.Get(govParticipationKey(b.val))...)
		mark := byte(0)
		if voted {
			mark = 1
		}
		window = append(window, mark)
		if len(window) > GovParticipationWindow {
			window = window[len(window)-GovParticipationWindow:]
		}
		store.Set(govParticipationKey(b.val), window)

		rep, found := k.GetValidatorReputation(ctx, b.cons.String())
		if !found {
			continue
		}
		rep.GovParticipation = participationRatio(window).String()
		k.SetValidatorReputation(ctx, rep)
	}

	// The marks for this proposal have been consumed.
	var keys [][]byte
	iter := storetypes.KVStorePrefixIterator(store, govVoteProposalPrefix(proposalID))
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, append([]byte{}, iter.Key()...))
	}
	iter.Close()
	for _, key := range keys {
		store.Delete(key)
	}
	return nil
}

func participationRatio(window []byte) sdkmath.LegacyDec {
	if len(window) == 0 {
		return neutralScore
	}
	voted := int64(0)
	for _, b := range window {
		if b == 1 {
			voted++
		}
	}
	return sdkmath.LegacyNewDec(voted).Quo(sdkmath.LegacyNewDec(int64(len(window))))
}

// ---- Removal ----

// DeleteValidatorReputation deletes a record and its score history.
func (k Keeper) DeleteValidatorReputation(ctx sdk.Context, consAddr string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(validatorKey(consAddr))

	prefix := append(append([]byte{}, types.HistoryPrefix...), []byte(consAddr+"/")...)
	var keys [][]byte
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, append([]byte{}, iter.Key()...))
	}
	iter.Close()
	for _, key := range keys {
		store.Delete(key)
	}
}

// OnValidatorRemoved deletes the removed validator's record, history and
// participation window. No-op below the switch.
func (k Keeper) OnValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	if !k.scoringV2Enabled(ctx) {
		return
	}
	ctx = unmetered(ctx)
	k.DeleteValidatorReputation(ctx, consAddr.String())
	ctx.KVStore(k.storeKey).Delete(govParticipationKey(valAddr))
}

// PruneRemovedValidators deletes the records (and history) of validators that
// no longer exist in x/staking. The removal hook covers every removal from the
// switch on; this covers the ones before it. It returns how many it deleted.
func (k Keeper) PruneRemovedValidators(ctx sdk.Context) (int, error) {
	if k.staking == nil {
		return 0, nil
	}
	var gone []string
	for _, rep := range k.GetAllValidatorReputations(ctx) {
		consAddr, err := sdk.ConsAddressFromBech32(rep.Address)
		if err != nil {
			// Not a consensus address; no validator can ever match it.
			gone = append(gone, rep.Address)
			continue
		}
		_, err = k.staking.GetValidatorByConsAddr(ctx, consAddr)
		switch {
		case err == nil:
		case isNotFound(err):
			gone = append(gone, rep.Address)
		default:
			return 0, err
		}
	}
	for _, addr := range gone {
		k.DeleteValidatorReputation(ctx, addr)
	}
	return len(gone), nil
}

func isNotFound(err error) bool {
	return errors.Is(err, stakingtypes.ErrNoValidatorFound)
}

// ---- Bonded-set transitions and slashes ----

// setInactiveSince records when a validator left the bonded set (height > 0),
// or clears it when it re-enters (height 0). Only an existing record is
// touched. No-op below the switch.
func (k Keeper) setInactiveSince(ctx sdk.Context, consAddr sdk.ConsAddress, height int64) {
	if !k.scoringV2Enabled(ctx) {
		return
	}
	ctx = unmetered(ctx)
	rep, found := k.GetValidatorReputation(ctx, consAddr.String())
	if !found || rep.InactiveSince == height {
		return
	}
	rep.InactiveSince = height
	k.SetValidatorReputation(ctx, rep)
}

// recordSlash counts a slash on the validator's record. No-op below the
// switch.
func (k Keeper) recordSlash(ctx sdk.Context, valAddr sdk.ValAddress) {
	if !k.scoringV2Enabled(ctx) || k.staking == nil {
		return
	}
	ctx = unmetered(ctx)
	v, err := k.staking.GetValidator(ctx, valAddr)
	if err != nil {
		return
	}
	consBz, err := v.GetConsAddr()
	if err != nil {
		return
	}
	rep, found := k.GetValidatorReputation(ctx, sdk.ConsAddress(consBz).String())
	if !found {
		return
	}
	rep.SlashingEvents++
	k.SetValidatorReputation(ctx, rep)
}

// MarkInactiveRecords sets inactive_since = the current height on every record
// of a validator that exists but is not bonded, so the records scoring v2 no
// longer touches say so from the switch on. Run once by the release handler;
// bounded by the number of validators the chain has ever had. It returns how
// many records it marked.
func (k Keeper) MarkInactiveRecords(ctx sdk.Context) (int, error) {
	if k.staking == nil {
		return 0, nil
	}
	marked := 0
	for _, rep := range k.GetAllValidatorReputations(ctx) {
		consAddr, err := sdk.ConsAddressFromBech32(rep.Address)
		if err != nil {
			continue
		}
		v, err := k.staking.GetValidatorByConsAddr(ctx, consAddr)
		switch {
		case err == nil:
		case isNotFound(err):
			continue
		default:
			return 0, err
		}
		if v.IsBonded() || rep.InactiveSince != 0 {
			continue
		}
		rep.InactiveSince = ctx.BlockHeight()
		k.SetValidatorReputation(ctx, rep)
		marked++
	}
	return marked, nil
}

// ---- Hooks ----

// Hooks connects the keeper to x/gov and x/staking. Every method is a no-op
// below the switch. It holds a pointer so that a keeper whose feature gate is
// installed after the hooks are provided is still the one they call.
type Hooks struct{ k *Keeper }

// NewHooks returns the gov and staking hooks for the keeper at k.
func NewHooks(k *Keeper) Hooks { return Hooks{k: k} }

func (h Hooks) keeper() (*Keeper, bool) { return h.k, h.k != nil }

// Gov hooks.

func (h Hooks) AfterProposalSubmission(context.Context, uint64) error { return nil }
func (h Hooks) AfterProposalDeposit(context.Context, uint64, sdk.AccAddress) error {
	return nil
}
func (h Hooks) AfterProposalVote(ctx context.Context, proposalID uint64, voter sdk.AccAddress) error {
	if k, ok := h.keeper(); ok {
		k.RecordGovVote(sdk.UnwrapSDKContext(ctx), proposalID, voter)
	}
	return nil
}
func (h Hooks) AfterProposalFailedMinDeposit(context.Context, uint64) error { return nil }
func (h Hooks) AfterProposalVotingPeriodEnded(ctx context.Context, proposalID uint64) error {
	if k, ok := h.keeper(); ok {
		return k.SettleGovParticipation(sdk.UnwrapSDKContext(ctx), proposalID)
	}
	return nil
}

// Staking hooks.

func (h Hooks) AfterValidatorCreated(context.Context, sdk.ValAddress) error   { return nil }
func (h Hooks) BeforeValidatorModified(context.Context, sdk.ValAddress) error { return nil }
func (h Hooks) AfterValidatorRemoved(ctx context.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) error {
	if k, ok := h.keeper(); ok {
		k.OnValidatorRemoved(sdk.UnwrapSDKContext(ctx), consAddr, valAddr)
	}
	return nil
}
func (h Hooks) AfterValidatorBonded(ctx context.Context, consAddr sdk.ConsAddress, _ sdk.ValAddress) error {
	if k, ok := h.keeper(); ok {
		k.setInactiveSince(sdk.UnwrapSDKContext(ctx), consAddr, 0)
	}
	return nil
}
func (h Hooks) AfterValidatorBeginUnbonding(ctx context.Context, consAddr sdk.ConsAddress, _ sdk.ValAddress) error {
	if k, ok := h.keeper(); ok {
		sctx := sdk.UnwrapSDKContext(ctx)
		k.setInactiveSince(sctx, consAddr, sctx.BlockHeight())
	}
	return nil
}
func (h Hooks) BeforeDelegationCreated(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}
func (h Hooks) BeforeDelegationSharesModified(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}
func (h Hooks) BeforeDelegationRemoved(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}
func (h Hooks) AfterDelegationModified(context.Context, sdk.AccAddress, sdk.ValAddress) error {
	return nil
}
func (h Hooks) BeforeValidatorSlashed(ctx context.Context, valAddr sdk.ValAddress, _ sdkmath.LegacyDec) error {
	if k, ok := h.keeper(); ok {
		k.recordSlash(sdk.UnwrapSDKContext(ctx), valAddr)
	}
	return nil
}
func (h Hooks) AfterUnbondingInitiated(context.Context, uint64) error { return nil }

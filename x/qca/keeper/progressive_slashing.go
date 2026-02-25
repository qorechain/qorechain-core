//go:build proprietary

package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/qca/types"
	"github.com/qorechain/qorechain-core/x/rlconsensus/mathutil"
)

var ln2 = math.LegacyMustNewDecFromStr("0.693147180559945309")

// ComputeProgressivePenalty computes the progressive slashing penalty for a validator.
// It considers all past infractions with temporal decay.
//
// Formula:
//
//	penalty = base_rate * escalation_factor^effective_count * severity_factor
//
// Temporal decay:
//
//	effective_count = SUM( 0.5^(blocks_since_i / decay_halflife) ) for each past infraction i
//
// The penalty is capped at MaxPenalty (default 33%) per slash event.
func (k Keeper) ComputeProgressivePenalty(
	ctx sdk.Context,
	validatorAddr string,
	currentHeight int64,
	infractionType string,
	severityFactor math.LegacyDec,
) (math.LegacyDec, error) {
	config := k.GetConfig(ctx)
	slashConfig := config.SlashingConfig

	baseRate, err := math.LegacyNewDecFromStr(slashConfig.BaseRate)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid base_rate: %w", err)
	}
	escalationFactor, err := math.LegacyNewDecFromStr(slashConfig.EscalationFactor)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid escalation_factor: %w", err)
	}
	maxPenalty, err := math.LegacyNewDecFromStr(slashConfig.MaxPenalty)
	if err != nil {
		return math.LegacyZeroDec(), fmt.Errorf("invalid max_penalty: %w", err)
	}
	halflife := int64(slashConfig.DecayHalflife)
	if halflife <= 0 {
		halflife = 100_000
	}

	// Get past infractions for this validator
	records := k.GetSlashingRecords(ctx, validatorAddr)

	// Compute effective count with temporal decay:
	// effective_count = SUM( 0.5^(blocks_since_i / halflife) )
	effectiveCount := math.LegacyZeroDec()
	halflifeDec := math.LegacyNewDec(halflife)

	for _, record := range records {
		blocksSince := currentHeight - record.InfractionHeight
		if blocksSince < 0 {
			continue // future records should not count
		}
		// exponent = blocks_since / halflife
		exponent := math.LegacyNewDec(blocksSince).Quo(halflifeDec)
		// 0.5^exponent = exp(-ln2 * exponent)
		decay := mathutil.ExpApprox(ln2.Neg().Mul(exponent))
		effectiveCount = effectiveCount.Add(decay)
	}

	// escalation_factor^effective_count = exp(effective_count * ln(escalation_factor))
	// ln(escalation_factor) = ln(1 + (escalation_factor - 1)) via TaylorLn1PlusX
	one := math.LegacyOneDec()
	lnEscalation := mathutil.TaylorLn1PlusX(escalationFactor.Sub(one))
	escalationPow := mathutil.ExpApprox(effectiveCount.Mul(lnEscalation))

	// penalty = base_rate * escalation^effective_count * severity_factor
	penalty := baseRate.Mul(escalationPow).Mul(severityFactor)

	// Cap at maxPenalty
	if penalty.GT(maxPenalty) {
		penalty = maxPenalty
	}

	// Store the new slashing record
	record := types.SlashingRecord{
		ValidatorAddr:    validatorAddr,
		InfractionHeight: currentHeight,
		InfractionType:   infractionType,
		SeverityFactor:   severityFactor.String(),
		Penalty:          penalty.String(),
	}
	k.SetSlashingRecord(ctx, record)

	// Update stats
	stats := k.GetStats(ctx)
	stats.SlashingEvents++
	k.SetStats(ctx, stats)

	return penalty, nil
}

// SetSlashingRecord stores a slashing record in the KV store.
func (k Keeper) SetSlashingRecord(ctx sdk.Context, record types.SlashingRecord) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(record)
	store.Set(types.SlashingRecordKey(record.ValidatorAddr, record.InfractionHeight), bz)
}

// GetSlashingRecords retrieves all slashing records for a validator.
func (k Keeper) GetSlashingRecords(ctx sdk.Context, validatorAddr string) []types.SlashingRecord {
	store := ctx.KVStore(k.storeKey)
	prefix := append(types.SlashingRecordPrefix, []byte(validatorAddr)...)
	prefix = append(prefix, '/')

	var records []types.SlashingRecord
	iter := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var record types.SlashingRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		records = append(records, record)
	}
	return records
}

// PruneOldSlashingRecords removes slashing records older than pruneAfter blocks.
func (k Keeper) PruneOldSlashingRecords(ctx sdk.Context, validatorAddr string, pruneAfter int64) {
	store := ctx.KVStore(k.storeKey)
	records := k.GetSlashingRecords(ctx, validatorAddr)
	currentHeight := ctx.BlockHeight()

	for _, record := range records {
		if currentHeight-record.InfractionHeight > pruneAfter {
			store.Delete(types.SlashingRecordKey(record.ValidatorAddr, record.InfractionHeight))
		}
	}
}

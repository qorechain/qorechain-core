package types

import (
	"cosmossdk.io/math"
)

// BurnSource identifies the origin of a burn event.
type BurnSource string

const (
	BurnSourceGasFee          BurnSource = "gas_fee"
	BurnSourceContractCreate  BurnSource = "contract_create"
	BurnSourceAIService       BurnSource = "ai_service"
	BurnSourceBridgeFee       BurnSource = "bridge_fee"
	BurnSourceTreasuryBuyback BurnSource = "treasury_buyback"
	BurnSourceFailedTx        BurnSource = "failed_tx"
	BurnSourceXQOREPenalty    BurnSource = "xqore_penalty"
	BurnSourceAutoBuyback     BurnSource = "auto_buyback"
	BurnSourceTGE             BurnSource = "tge"
	BurnSourceRollupCreate    BurnSource = "rollup_create"
	BurnSourceMilestone       BurnSource = "milestone"
	BurnSourceAMM             BurnSource = "amm"
)

// ValidBurnSources returns all valid burn sources.
func ValidBurnSources() []BurnSource {
	return []BurnSource{
		BurnSourceGasFee, BurnSourceContractCreate, BurnSourceAIService,
		BurnSourceBridgeFee, BurnSourceTreasuryBuyback, BurnSourceFailedTx,
		BurnSourceXQOREPenalty, BurnSourceAutoBuyback, BurnSourceTGE,
		BurnSourceRollupCreate, BurnSourceMilestone, BurnSourceAMM,
	}
}

// IsValidBurnSource checks if a burn source is valid.
func IsValidBurnSource(s BurnSource) bool {
	for _, v := range ValidBurnSources() {
		if v == s {
			return true
		}
	}
	return false
}

// BurnRecord tracks a single burn event.
type BurnRecord struct {
	Source BurnSource `json:"source"`
	Amount math.Int   `json:"amount"`
	Height int64      `json:"height"`
	TxHash string     `json:"tx_hash,omitempty"`
}

// BurnStats tracks aggregate burn statistics.
type BurnStats struct {
	TotalBurned    math.Int                `json:"total_burned"`
	BurnsBySource  map[BurnSource]math.Int `json:"burns_by_source"`
	LastBurnHeight int64                   `json:"last_burn_height"`
}

// MilestoneBurnTier defines a cumulative TX threshold that triggers a bonus burn.
type MilestoneBurnTier struct {
	TxThreshold uint64   `json:"tx_threshold"` // cumulative TX count to trigger burn
	BurnAmount  math.Int `json:"burn_amount"`  // uqor to burn when threshold is crossed
}

// MilestoneState tracks milestone burn progress.
type MilestoneState struct {
	CumulativeTxCount  uint64 `json:"cumulative_tx_count"`
	LastTriggeredIndex int    `json:"last_triggered_index"` // index of last triggered tier (-1 = none)
}

// DefaultMilestoneState returns a zero-valued milestone state.
func DefaultMilestoneState() MilestoneState {
	return MilestoneState{
		CumulativeTxCount:  0,
		LastTriggeredIndex: -1,
	}
}

// DefaultBurnStats returns zero-valued burn stats.
func DefaultBurnStats() BurnStats {
	bySource := make(map[BurnSource]math.Int)
	for _, s := range ValidBurnSources() {
		bySource[s] = math.ZeroInt()
	}
	return BurnStats{
		TotalBurned:   math.ZeroInt(),
		BurnsBySource: bySource,
	}
}

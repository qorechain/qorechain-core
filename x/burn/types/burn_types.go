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
)

// ValidBurnSources returns all valid burn sources.
func ValidBurnSources() []BurnSource {
	return []BurnSource{
		BurnSourceGasFee, BurnSourceContractCreate, BurnSourceAIService,
		BurnSourceBridgeFee, BurnSourceTreasuryBuyback, BurnSourceFailedTx,
		BurnSourceXQOREPenalty, BurnSourceAutoBuyback, BurnSourceTGE,
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
	Amount math.Int  `json:"amount"`
	Height int64     `json:"height"`
	TxHash string    `json:"tx_hash,omitempty"`
}

// BurnStats tracks aggregate burn statistics.
type BurnStats struct {
	TotalBurned    math.Int                `json:"total_burned"`
	BurnsBySource  map[BurnSource]math.Int `json:"burns_by_source"`
	LastBurnHeight int64                   `json:"last_burn_height"`
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

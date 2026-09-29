package app

import (
	"time"

	"github.com/spf13/cast"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// app.toml keys for the node-local mempool and proposal limits ([qorechain]
// section). A missing key, as in an app.toml written by an older release,
// means the default.
const (
	FlagCosmosMempoolMaxTxs     = "qorechain.cosmos-mempool-max-txs"
	FlagPrepareProposalBudget   = "qorechain.prepare-proposal-budget"
	FlagPrepareProposalMaxSkips = "qorechain.prepare-proposal-max-skips"
)

// MempoolConfig holds the node-local limits on the app-side mempool and on
// PrepareProposal. None of them affects consensus: they only decide what this
// node keeps in memory and what it puts in the blocks it proposes.
type MempoolConfig struct {
	// CosmosMempoolMaxTxs caps the Cosmos side of the unified mempool.
	CosmosMempoolMaxTxs int
	// PrepareProposalBudget is the time PrepareProposal may spend selecting
	// transactions. Zero disables the budget.
	PrepareProposalBudget time.Duration
	// PrepareProposalMaxSkips is how many non-fitting candidates in a row end
	// selection. Zero disables the cap.
	PrepareProposalMaxSkips int
}

// DefaultMempoolConfig returns the limits used when app.toml says nothing.
func DefaultMempoolConfig() MempoolConfig {
	return MempoolConfig{
		CosmosMempoolMaxTxs:     DefaultCosmosMempoolMaxTxs,
		PrepareProposalBudget:   DefaultPrepareProposalBudget,
		PrepareProposalMaxSkips: DefaultPrepareProposalMaxSkips,
	}
}

// ReadMempoolConfig reads the [qorechain] mempool keys. The Cosmos pool can
// not be made unbounded from config: a non-positive size falls back to the
// default, because an unbounded pool is exactly what QSR-2026-0286 is about.
// Unparseable or negative budget and skip values also fall back to their
// defaults; an explicit 0 disables that limit.
func ReadMempoolConfig(appOpts servertypes.AppOptions) MempoolConfig {
	cfg := DefaultMempoolConfig()
	if appOpts == nil {
		return cfg
	}
	if v := appOpts.Get(FlagCosmosMempoolMaxTxs); v != nil {
		if n, err := cast.ToIntE(v); err == nil && n > 0 {
			cfg.CosmosMempoolMaxTxs = n
		}
	}
	if v := appOpts.Get(FlagPrepareProposalBudget); v != nil {
		if d, err := cast.ToDurationE(v); err == nil && d >= 0 {
			cfg.PrepareProposalBudget = d
		}
	}
	if v := appOpts.Get(FlagPrepareProposalMaxSkips); v != nil {
		if n, err := cast.ToIntE(v); err == nil && n >= 0 {
			cfg.PrepareProposalMaxSkips = n
		}
	}
	return cfg
}

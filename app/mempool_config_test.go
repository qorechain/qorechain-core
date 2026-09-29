package app

import (
	"testing"
	"time"
)

type mapOpts map[string]interface{}

func (m mapOpts) Get(k string) interface{} { return m[k] }

func TestReadMempoolConfigDefaults(t *testing.T) {
	want := MempoolConfig{CosmosMempoolMaxTxs: 5000, PrepareProposalBudget: time.Second, PrepareProposalMaxSkips: 64}
	if got := ReadMempoolConfig(nil); got != want {
		t.Fatalf("nil options: got %+v, want %+v", got, want)
	}
	// An app.toml from an older release has no [qorechain] section.
	if got := ReadMempoolConfig(mapOpts{}); got != want {
		t.Fatalf("missing section: got %+v, want %+v", got, want)
	}
}

func TestReadMempoolConfigValues(t *testing.T) {
	got := ReadMempoolConfig(mapOpts{
		FlagCosmosMempoolMaxTxs:     "2000",
		FlagPrepareProposalBudget:   "750ms",
		FlagPrepareProposalMaxSkips: 16,
	})
	want := MempoolConfig{CosmosMempoolMaxTxs: 2000, PrepareProposalBudget: 750 * time.Millisecond, PrepareProposalMaxSkips: 16}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestReadMempoolConfigCannotUnboundThePool(t *testing.T) {
	for _, v := range []interface{}{0, -1, "0", "abc"} {
		if got := ReadMempoolConfig(mapOpts{FlagCosmosMempoolMaxTxs: v}); got.CosmosMempoolMaxTxs != DefaultCosmosMempoolMaxTxs {
			t.Fatalf("max txs %v: got %d, want the default", v, got.CosmosMempoolMaxTxs)
		}
	}
}

func TestReadMempoolConfigZeroDisablesProposalLimits(t *testing.T) {
	got := ReadMempoolConfig(mapOpts{FlagPrepareProposalBudget: "0s", FlagPrepareProposalMaxSkips: 0})
	if got.PrepareProposalBudget != 0 || got.PrepareProposalMaxSkips != 0 {
		t.Fatalf("explicit zero must disable: %+v", got)
	}
	got = ReadMempoolConfig(mapOpts{FlagPrepareProposalBudget: "soon", FlagPrepareProposalMaxSkips: -3})
	if got.PrepareProposalBudget != DefaultPrepareProposalBudget || got.PrepareProposalMaxSkips != DefaultPrepareProposalMaxSkips {
		t.Fatalf("invalid values must fall back to defaults: %+v", got)
	}
}

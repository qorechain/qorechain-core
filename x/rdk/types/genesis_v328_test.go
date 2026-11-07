package types

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// A stored parameter record and a genesis file written before v3.2.8 keep
// their bytes: the new retention field and the escrow ledger are omitted
// while unset, so nothing a replaying node writes at genesis or on a
// parameter update changes.
func TestV328FieldsOmittedWhenUnset(t *testing.T) {
	p, err := json.Marshal(DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"max_rollups":100,"min_stake_for_rollup":10000000000,"rollup_creation_burn_rate":"0.01",` +
		`"default_challenge_window":604800,"max_da_blob_size":2097152,"blob_retention_blocks":432000,"max_batches_per_block":10}`
	if string(p) != want {
		t.Fatalf("default params bytes changed:\n got %s\nwant %s", p, want)
	}
	g, err := json.Marshal(DefaultGenesisState())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(g, []byte("escrow_ledger")) {
		t.Fatalf("default genesis carries escrow_ledger: %s", g)
	}
	// A message built before v3.2.8 carries no retention.
	if r := (&MsgUpdateParams{MaxRollups: 1}).ToParams().FinalizedBatchRetentionBlocks; r != 0 {
		t.Fatalf("ToParams retention %d, want 0", r)
	}
}

func TestFinalizedBatchRetention(t *testing.T) {
	p := DefaultParams()
	if got := p.FinalizedBatchRetention(); got != DefaultFinalizedBatchRetentionBlocks {
		t.Fatalf("zero retention resolves to %d, want the default", got)
	}
	p.FinalizedBatchRetentionBlocks = 7_000
	if got := p.FinalizedBatchRetention(); got != 7_000 {
		t.Fatalf("retention %d, want 7000", got)
	}
	if err := p.ValidateUpdate(); err != nil {
		t.Fatal(err)
	}
	p.FinalizedBatchRetentionBlocks = MinFinalizedBatchRetentionBlocks - 1
	if err := p.ValidateUpdate(); err == nil {
		t.Fatal("a retention below the minimum must be refused")
	}
	p.FinalizedBatchRetentionBlocks = 0
	if err := p.ValidateUpdate(); err != nil {
		t.Fatalf("zero (default) retention refused: %v", err)
	}
}

// The escrow ledger in genesis: amounts as strings, entries only for rollups
// in the same genesis, one per rollup.
func TestGenesisEscrowLedger(t *testing.T) {
	gs := DefaultGenesisState()
	r := RollupConfig{
		RollupID: "test-rollup", Creator: "qor1abc", Profile: ProfileDeFi, SettlementMode: SettlementOptimistic,
		SequencerConfig: SequencerConfig{Mode: SequencerDedicated}, DABackend: DANative, BlockTimeMs: 1000,
		MaxTxPerBlock: 10000, GasConfig: DefaultRollupGasConfig(), VMType: "evm",
		ProofConfig: ProofConfig{System: ProofSystemFraud, ChallengeWindowSec: 604800}, StakeAmount: 10000000000,
	}
	gs.Rollups = []RollupConfig{r}
	gs.EscrowLedger = []RollupEscrowEntry{{RollupID: r.RollupID, Amount: 18446744073709551615}}
	if err := gs.Validate(); err != nil {
		t.Fatal(err)
	}
	bz, err := json.Marshal(gs)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bz), `"amount":"18446744073709551615"`) {
		t.Fatalf("ledger amount not encoded as a string: %s", bz)
	}
	var back GenesisState
	if err := json.Unmarshal(bz, &back); err != nil || back.EscrowLedger[0] != gs.EscrowLedger[0] {
		t.Fatalf("round trip: %+v, %v", back.EscrowLedger, err)
	}

	unknown := *gs
	unknown.EscrowLedger = []RollupEscrowEntry{{RollupID: "nope", Amount: 1}}
	if err := unknown.Validate(); err == nil {
		t.Fatal("a ledger entry for a rollup not in genesis must be refused")
	}
	dup := *gs
	dup.EscrowLedger = []RollupEscrowEntry{{RollupID: r.RollupID, Amount: 1}, {RollupID: r.RollupID, Amount: 0}}
	if err := dup.Validate(); err == nil {
		t.Fatal("a duplicate ledger entry must be refused")
	}
}

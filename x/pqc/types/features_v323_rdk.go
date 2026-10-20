package types

// Released in v3.2.3, RDK area.
const (
	// FeatureRDKEscrowLedger makes the per-rollup escrow ledger the single
	// account of what each rollup holds in the pooled rdk module balance
	// (QSR-2026-0513). From the switch on, CreateRollup credits the ledger with
	// the escrow it actually keeps (the stake less the creation burn), a
	// withdrawal and a fraud-proof reward debit it, StopRollup refunds only
	// what is left in it and zeroes it, and a stopped rollup pays no further
	// withdrawals. Below the switch the v3.2.2 behaviour is kept: the ledger is
	// credited with 0, every withdrawal fails closed, and StopRollup refunds
	// the recorded escrow figure.
	FeatureRDKEscrowLedger = "rdk-escrow-ledger"
)

// V323RdkFeatures is the RDK area's share of the v3.2.3 release switches.
var V323RdkFeatures = []string{
	FeatureRDKEscrowLedger,
}

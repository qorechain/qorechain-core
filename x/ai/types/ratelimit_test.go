package types

import "testing"

// The shipped default is a wallet's ceiling. The release raises it in the
// upgrade handler, because an IBC relayer sends one transaction per packet and
// would stall at ten per 30-block window, and x/ai has no message that can
// change the value on a running chain.
func TestDefaultRateLimitIsTheWalletCeiling(t *testing.T) {
	if got := DefaultAIConfig().MaxTxPerMinute; got != 10 {
		t.Fatalf("default max_tx_per_minute changed to %d; the upgrade handler assumes 10", got)
	}
}

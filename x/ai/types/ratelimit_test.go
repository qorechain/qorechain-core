package types

import "testing"

// The shipped default is the relayer-friendly ceiling the v3.2.0 release
// handler set on the running networks (600 per 30-block window). The handler
// only raises a lower stored value, so on a chain started with this default it
// changes nothing. A chain started with the old default of 10 would stall an
// IBC relayer (one transaction per packet) and, once the v3.2.2 limit counts
// every transaction rather than one per block, exchange hot wallets too.
func TestDefaultRateLimitIsRelayerFriendly(t *testing.T) {
	if got := DefaultAIConfig().MaxTxPerMinute; got != 600 {
		t.Fatalf("default max_tx_per_minute changed to %d; must stay relayer-friendly (600)", got)
	}
	if err := DefaultAIConfig().Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

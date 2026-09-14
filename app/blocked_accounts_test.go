package app

import "testing"

// Every module account must be blocked from receiving an ordinary bank send,
// except the one that is designed to be pre-funded. A six-entry hand list left
// eighteen custom module accounts receivable; a computed list cannot forget one.
func TestEveryModuleAccountIsBlockedExceptSVM(t *testing.T) {
	blocked := BlockedAddresses()

	for _, p := range moduleAccPerms {
		if p.Account == "svm" {
			if blocked["svm"] {
				t.Fatal("svm is blocked; the testnet faucet pre-funds it with a plain send " +
					"and it never mints, so it must stay receivable")
			}
			continue
		}
		if !blocked[p.Account] {
			t.Fatalf("module account %q can receive an ordinary bank send: coins sent "+
				"there are unrecoverable and unsolicited deposits break its invariants",
				p.Account)
		}
	}
	// The six that were always blocked must still be.
	for _, name := range []string{"fee_collector", "distribution", "mint", "bonded_tokens_pool", "not_bonded_tokens_pool", "nft"} {
		if !blocked[name] {
			t.Fatalf("%q was blocked before and is not now", name)
		}
	}
}

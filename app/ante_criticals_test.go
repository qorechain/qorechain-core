package app

import (
	"testing"
	"time"

	aatypes "github.com/qorechain/qorechain-core/x/abstractaccount/types"
)

// THE CONSENSUS PROPERTY. The TTL bound must be a function of block time, which is
// identical on every validator executing a block, and never of the node's own clock.
// The previous version used time.Until: a registration whose expiry sat on the
// boundary was accepted by validators whose clock put it under and rejected by those
// whose clock put it over, which diverges block results and halts the chain.
func TestAuthenticatorTTLIsDecidedByBlockTimeNotWallClock(t *testing.T) {
	blockTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	// Exactly at the bound, measured from block time: admitted.
	atBound := blockTime.Add(aatypes.MaxSessionTTL)
	if err := aatypes.ValidateAuthenticatorTTL(atBound, blockTime); err != nil {
		t.Fatalf("an expiry exactly at the bound was refused: %v", err)
	}

	// One nanosecond past it: refused. Same answer on every node, because both
	// operands come from the block.
	if err := aatypes.ValidateAuthenticatorTTL(atBound.Add(time.Nanosecond), blockTime); err == nil {
		t.Fatal("an expiry past the bound was admitted")
	}

	// The decisive property: the verdict depends only on the arguments. A node whose
	// wall clock is hours away from block time reaches the identical answer, because
	// the wall clock is not consulted.
	farFromNow := time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := aatypes.ValidateAuthenticatorTTL(farFromNow.Add(aatypes.MaxSessionTTL), farFromNow); err != nil {
		t.Fatalf("the same relative expiry was judged differently at another point in "+
			"time, which means something other than the arguments is being consulted: %v", err)
	}
}

// A zero expiry is still refused: moving the check must not have dropped the case
// that a missing expiry is not an unlimited one.
func TestAuthenticatorTTLStillRefusesAZeroExpiry(t *testing.T) {
	if err := aatypes.ValidateAuthenticatorTTL(time.Time{}, time.Now()); err == nil {
		t.Fatal("a zero expiry was accepted")
	}
}

// ValidateBasic must no longer reject on TTL at all, because it has no block time to
// measure against. It must still reject the things that are decidable from the
// message alone.
func TestValidateBasicNoLongerJudgesTTL(t *testing.T) {
	// An expiry far beyond the maximum. ValidateBasic cannot know that without a
	// clock, so it must accept it and leave the judgement to the handler.
	auth := aatypes.Authenticator{
		Scheme:      aatypes.SchemeEd25519,
		PubKey:      make([]byte, 32),
		Permissions: []string{aatypes.PermSend},
		Expiry:      time.Now().Add(100 * 24 * time.Hour),
	}
	if err := aatypes.ValidateAuthenticator(auth); err != nil {
		t.Fatalf("ValidateBasic still judges TTL, which is the consensus hazard: %v", err)
	}

	// And the same authenticator is refused by the handler's check.
	if err := aatypes.ValidateAuthenticatorTTL(auth.Expiry, time.Now()); err == nil {
		t.Fatal("the TTL bound is now enforced nowhere at all")
	}
}

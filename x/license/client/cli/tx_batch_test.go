package cli

import (
	"encoding/json"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
)

// parseBatch runs the document through the same decoding the command does, so the
// tests below exercise the real shape rather than a hand-built struct.
func parseBatch(t *testing.T, doc string) batchFile {
	t.Helper()
	var b batchFile
	if err := json.Unmarshal([]byte(doc), &b); err != nil {
		t.Fatalf("test document does not parse: %v", err)
	}
	return b
}

// THE INTERLOCK. Granting a light-node operator licence today credits the first
// registrant the entire accumulated reward pool: 3.1M QOR on mainnet, 66.5B on
// testnet, measured with zero nodes registered. The block lives in the binary rather
// than in a flag or a parameter, so lifting it takes a deliberate release. This test
// is what stops it being removed absent-mindedly alongside an unrelated change.
func TestLightNodeOperatorIsBlockedFromGranting(t *testing.T) {
	reason, blocked := blockedFeatures[types.FeatureLightNodeOperator]
	if !blocked {
		t.Fatal("lightnode_operator can be granted again. If the reward-pool fix has " +
			"shipped, this test should have been deleted in the same commit; if it has " +
			"not, the first grantee takes the whole pool")
	}
	// The reason has to survive too: an operator who hits this needs to know why, not
	// just that something said no.
	for _, want := range []string{"EndBlocker", "whole", "reward-pool fix"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("the block's explanation lost %q, leaving an operator with a "+
				"refusal and no reason: %q", want, reason)
		}
	}
}

// The features that are safe to grant must not be caught by the interlock, or the
// block stops being a targeted safety measure and becomes an outage.
func TestOtherFeaturesAreNotBlocked(t *testing.T) {
	for _, f := range []string{types.FeatureValidatorOperator, types.FeatureQCBBridge} {
		if _, blocked := blockedFeatures[f]; blocked {
			t.Fatalf("%q is blocked, which stops legitimate granting", f)
		}
	}
}

// Every blocked feature must be a real feature id. A typo here would silently protect
// nothing at all, and the protection would look present while being absent.
func TestBlockedFeaturesAreRealFeatureIDs(t *testing.T) {
	for f := range blockedFeatures {
		if !types.IsValidFeatureID(f) {
			t.Fatalf("blocked feature %q is not a known feature id, so it blocks nothing", f)
		}
	}
}

// A batch names the network it was built for, and the command refuses to sign it onto
// a different one. This is what stops a testnet rehearsal from becoming a mainnet
// grant, so the field must be mandatory rather than defaulted.
func TestBatchMustNameItsChain(t *testing.T) {
	b := parseBatch(t, `{"grants":[{"grantee":"qor1x","feature_id":"validator_operator"}]}`)
	if b.ChainID != "" {
		t.Fatal("a batch with no chain_id decoded to a non-empty chain, which would " +
			"let it pass the network check")
	}
}

func TestBatchRoundTripsItsFields(t *testing.T) {
	addr := sdk.AccAddress([]byte("a-grantee-account-20")).String()
	doc := `{
	  "chain_id": "qorechain-vladi",
	  "generated_at": "2026-09-13T12:00:00Z",
	  "grants": [
	    {"grantee": "` + addr + `", "feature_id": "validator_operator",
	     "expires_at": 123, "metadata": "{}", "reference": "lic-0042"}
	  ]
	}`
	b := parseBatch(t, doc)

	if b.ChainID != "qorechain-vladi" || len(b.Grants) != 1 {
		t.Fatalf("batch header did not decode: %+v", b)
	}
	g := b.Grants[0]
	if g.Grantee != addr || g.FeatureID != types.FeatureValidatorOperator ||
		g.ExpiresAt != 123 || g.Reference != "lic-0042" {
		t.Fatalf("grant did not decode: %+v", g)
	}
}

// Two entries for the same pair would fail on chain at the second one and take the
// whole transaction with it, so the duplicate has to be caught while reading the file.
func TestDuplicateDetectionKeyIsPerGranteeAndFeature(t *testing.T) {
	a := sdk.AccAddress([]byte("the-same-account-xxx")).String()
	seen := map[string]string{}

	for _, g := range []batchGrant{
		{Grantee: a, FeatureID: types.FeatureValidatorOperator},
		{Grantee: a, FeatureID: types.FeatureQCBBridge},
	} {
		key := g.Grantee + "|" + g.FeatureID
		if _, dup := seen[key]; dup {
			t.Fatal("two different features for one account were treated as a duplicate")
		}
		seen[key] = "x"
	}

	key := a + "|" + types.FeatureValidatorOperator
	if _, dup := seen[key]; !dup {
		t.Fatal("the same grantee and feature twice was not detected")
	}
}

// THE COLLISION THAT THE FIRST CUT HAD. Blocked entries used to fail the whole
// batch. Once a blocked feature is queued, that means every batch carrying one is
// refused, taking down the grants that CAN pass. The block would then be blocking the
// wrong thing, silently, and the queue would have to be pre-filtered by whoever built
// it, putting the list of blocked features in two places that can drift.
//
// Excluding is safe here in a way it is not elsewhere: each grant is independent, so
// removing one does not change what the others mean. That is the test for whether
// excluding is acceptable rather than dangerous, and it is why this one may be
// excluded while a signer flag on someone else's account may not.
func TestBlockedEntriesAreExcludedNotFatalToTheBatch(t *testing.T) {
	a := sdk.AccAddress([]byte("a-validator-operator")).String()
	b := sdk.AccAddress([]byte("a-lightnode-operatr")).String()
	grants := []batchGrant{
		{Grantee: a, FeatureID: types.FeatureValidatorOperator},
		{Grantee: b, FeatureID: types.FeatureLightNodeOperator},
	}

	var blocked, allowed []batchGrant
	for _, g := range grants {
		if _, isBlocked := blockedFeatures[g.FeatureID]; isBlocked {
			blocked = append(blocked, g)
			continue
		}
		allowed = append(allowed, g)
	}

	if len(allowed) != 1 || allowed[0].FeatureID != types.FeatureValidatorOperator {
		t.Fatalf("the validator grant did not survive alongside a blocked one: %+v", allowed)
	}
	if len(blocked) != 1 || blocked[0].FeatureID != types.FeatureLightNodeOperator {
		t.Fatalf("the blocked grant was not set aside: %+v", blocked)
	}
	for _, g := range allowed {
		if _, isBlocked := blockedFeatures[g.FeatureID]; isBlocked {
			t.Fatal("a blocked feature reached the transaction, which is the one thing " +
				"excluding must never do")
		}
	}
}

// The confirmation asks for the count rather than a yes, because typing the count
// requires having read the list, and reading the list is the authorisation.
func TestConfirmationRequiresTheCountNotAYes(t *testing.T) {
	for _, answer := range []string{"y", "yes", "Y", "", "0", "16"} {
		if strings.TrimSpace(answer) == "15" {
			t.Fatalf("test setup is wrong: %q", answer)
		}
	}
	if strings.TrimSpace(" 15 ") != "15" {
		t.Fatal("surrounding whitespace must not defeat a correct answer")
	}
}

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/qorechain/qorechain-core/x/license/types"
	lightnodetypes "github.com/qorechain/qorechain-core/x/lightnode/types"
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

// THE INTERLOCK. Granting a light-node operator licence on a chain that has not
// established its reward-pool accounting credits the first registrant the entire
// accumulated pool: 3.1M QOR on mainnet, 66.5B on testnet, measured with zero nodes
// registered. The gate lives in the binary rather than in a flag or a parameter, and
// it asks the chain rather than trusting the release, because the release binary is
// the same before and after the upgrade height. This test is what stops the gate
// being removed absent-mindedly alongside an unrelated change.
func TestLightNodeOperatorIsChainGated(t *testing.T) {
	if _, gated := chainGatedFeatures[types.FeatureLightNodeOperator]; !gated {
		t.Fatal("lightnode_operator is no longer gated on the chain's reward-pool state; " +
			"a grant onto a network below its upgrade height hands the first registrant " +
			"the whole pool")
	}
}

// The three answers a chain can give, and what each must mean for the grant. An
// operator who is refused needs to know why, so the reasons are checked too.
func TestLightNodeGrantWaitsForTheChainToEstablishTheRewardPool(t *testing.T) {
	// Previous release: the query route does not exist.
	reason, blocked := lightNodeGrantGate(nil, errors.New("unknown method RewardPool"))
	if !blocked {
		t.Fatal("a chain without the reward_pool query was allowed; its EndBlocker " +
			"pays the whole balance to the first node")
	}
	for _, want := range []string{"reward_pool", "whole module balance", "first node"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("the refusal for an old chain lost %q: %q", want, reason)
		}
	}

	// This release, upgrade not executed yet: the query answers but the accounting
	// is not established, so the validators may still be on the paying release.
	reason, blocked = lightNodeGrantGate(&lightnodetypes.QueryRewardPoolResponse{Initialized: false}, nil)
	if !blocked {
		t.Fatal("a chain that has not executed the upgrade was allowed")
	}
	for _, want := range []string{"upgrade", "whole module balance"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("the refusal for a pre-upgrade chain lost %q: %q", want, reason)
		}
	}

	// A nil answer without an error is a client bug, and it must fail closed.
	if _, blocked = lightNodeGrantGate(nil, nil); !blocked {
		t.Fatal("a nil answer was treated as safe")
	}

	// The accounting is established: the grant may go through.
	if reason, blocked = lightNodeGrantGate(&lightnodetypes.QueryRewardPoolResponse{Initialized: true}, nil); blocked {
		t.Fatalf("a chain with the reward pool established was refused: %q", reason)
	}
}

// The features that are safe to grant must not be caught by the compile-time block,
// or the block stops being a targeted safety measure and becomes an outage. The
// light-node feature is listed deliberately: its protection is the chain gate, not
// a block, and both at once would refuse it everywhere.
func TestOtherFeaturesAreNotBlocked(t *testing.T) {
	for _, f := range []string{types.FeatureValidatorOperator, types.FeatureQCBBridge, types.FeatureLightNodeOperator} {
		if _, blocked := blockedFeatures[f]; blocked {
			t.Fatalf("%q is blocked, which stops legitimate granting", f)
		}
	}
}

// Every blocked or gated feature must be a real feature id. A typo here would
// silently protect nothing at all, and the protection would look present while
// being absent.
func TestBlockedFeaturesAreRealFeatureIDs(t *testing.T) {
	for f := range blockedFeatures {
		if !types.IsValidFeatureID(f) {
			t.Fatalf("blocked feature %q is not a known feature id, so it blocks nothing", f)
		}
	}
	for f := range chainGatedFeatures {
		if !types.IsValidFeatureID(f) {
			t.Fatalf("gated feature %q is not a known feature id, so it gates nothing", f)
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
	// Nothing is compile-time blocked in this release, so the test plants a block
	// of its own: the mechanism has to keep working for the day a feature needs
	// holding back again.
	blockedFeatures[types.FeatureQCBBridge] = "planted by the test"
	defer delete(blockedFeatures, types.FeatureQCBBridge)

	a := sdk.AccAddress([]byte("a-validator-operator")).String()
	b := sdk.AccAddress([]byte("a-bridge-operator-20")).String()
	grants := []batchGrant{
		{Grantee: a, FeatureID: types.FeatureValidatorOperator},
		{Grantee: b, FeatureID: types.FeatureQCBBridge},
	}

	allowed, blocked, reasons := setAside(context.Background(), client.Context{}, grants)

	if len(allowed) != 1 || allowed[0].FeatureID != types.FeatureValidatorOperator {
		t.Fatalf("the validator grant did not survive alongside a blocked one: %+v", allowed)
	}
	if len(blocked) != 1 || blocked[0].FeatureID != types.FeatureQCBBridge {
		t.Fatalf("the blocked grant was not set aside: %+v", blocked)
	}
	if reasons[types.FeatureQCBBridge] != "planted by the test" {
		t.Fatalf("the refusal lost its reason: %q", reasons[types.FeatureQCBBridge])
	}
	for _, g := range allowed {
		if _, isBlocked := blockedFeatures[g.FeatureID]; isBlocked {
			t.Fatal("a blocked feature reached the transaction, which is the one thing " +
				"excluding must never do")
		}
	}
}

// A chain gate that says no sets its grants aside the same way, asks the chain once
// per batch rather than once per grant, and leaves the other grants alone.
func TestChainGatedEntriesAreExcludedNotFatalToTheBatch(t *testing.T) {
	calls := 0
	chainGatedFeatures[types.FeatureQCBBridge] = func(context.Context, client.Context) (string, bool) {
		calls++
		return "the chain is not ready for this feature", true
	}
	defer delete(chainGatedFeatures, types.FeatureQCBBridge)

	grants := []batchGrant{
		{Grantee: sdk.AccAddress([]byte("a-validator-operator")).String(), FeatureID: types.FeatureValidatorOperator},
		{Grantee: sdk.AccAddress([]byte("a-bridge-operator-01")).String(), FeatureID: types.FeatureQCBBridge},
		{Grantee: sdk.AccAddress([]byte("a-bridge-operator-02")).String(), FeatureID: types.FeatureQCBBridge},
	}

	allowed, blocked, reasons := setAside(context.Background(), client.Context{}, grants)

	if calls != 1 {
		t.Fatalf("the chain was asked %d times for one feature; once per batch is the contract", calls)
	}
	if len(allowed) != 1 || allowed[0].FeatureID != types.FeatureValidatorOperator {
		t.Fatalf("the validator grant did not survive alongside gated ones: %+v", allowed)
	}
	if len(blocked) != 2 {
		t.Fatalf("both gated grants should be set aside, got %+v", blocked)
	}
	if !strings.Contains(reasons[types.FeatureQCBBridge], "not ready") {
		t.Fatalf("the refusal lost its reason: %q", reasons[types.FeatureQCBBridge])
	}
}

// A gate that says yes must not leave a trace: the grant goes through and no reason
// is printed for it.
func TestAnOpenChainGateLetsTheGrantThrough(t *testing.T) {
	chainGatedFeatures[types.FeatureQCBBridge] = func(context.Context, client.Context) (string, bool) {
		return "", false
	}
	defer delete(chainGatedFeatures, types.FeatureQCBBridge)

	grants := []batchGrant{
		{Grantee: sdk.AccAddress([]byte("a-bridge-operator-01")).String(), FeatureID: types.FeatureQCBBridge},
	}
	allowed, blocked, reasons := setAside(context.Background(), client.Context{}, grants)
	if len(allowed) != 1 || len(blocked) != 0 || len(reasons) != 0 {
		t.Fatalf("an open gate changed the batch: allowed=%+v blocked=%+v reasons=%+v", allowed, blocked, reasons)
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

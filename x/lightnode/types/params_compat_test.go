package types

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
)

// The shape of the record on both networks today: no max_reward_per_block_per_node.
const legacyParamsJSON = `{"registration_fee":"1000000","heartbeat_interval":1000,` +
	`"min_delegated_stake":"1000000000","reward_share":"0.030000000000000000",` +
	`"min_uptime_for_rewards":"0.800000000000000000","max_light_nodes":10000,` +
	`"heartbeat_grace_period":100}`

// The three questions from the release runbook, answered by bytes: a record
// without the new field must validate, must read the default bound, and must
// marshal back to exactly the bytes it came from. Anything else is a consensus
// change at height 0 or a node that refuses the real genesis file.
func TestParamsWithoutTheBoundRoundTripByteForByte(t *testing.T) {
	var p Params
	if err := json.Unmarshal([]byte(legacyParamsJSON), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(); err != nil {
		t.Fatalf("the pre-field record was refused: %v", err)
	}
	if got := p.MaxRewardPerBlock(); !got.Equal(math.NewInt(1713)) {
		t.Fatalf("default bound: got %s", got)
	}
	out, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != legacyParamsJSON {
		t.Fatalf("bytes changed on a round trip:\n got %s\nwant %s", out, legacyParamsJSON)
	}

	// Set, the value survives and a negative one is refused.
	v := math.NewInt(5)
	p.MaxRewardPerBlockPerNode = &v
	out, _ = json.Marshal(p)
	var back Params
	if err := json.Unmarshal(out, &back); err != nil || !back.MaxRewardPerBlock().Equal(v) {
		t.Fatalf("set value did not survive: %s err %v", out, err)
	}
	neg := math.NewInt(-1)
	p.MaxRewardPerBlockPerNode = &neg
	if err := p.Validate(); err == nil {
		t.Fatal("negative bound accepted")
	}
}

// The genuine genesis section of a network that upgrades into this binary:
// params in the old shape and no reward_pool at all. It must validate, and it
// must decode with no pool, so InitGenesis writes none at height 0.
func TestGenesisWithoutRewardPoolIsTheUpgradeCase(t *testing.T) {
	doc := `{"params":` + legacyParamsJSON + `,"light_nodes":[],"stats":{"total_registered":0,"total_active":0,"total_rewards":"0"}}`
	var gs GenesisState
	if err := json.Unmarshal([]byte(doc), &gs); err != nil {
		t.Fatal(err)
	}
	if err := gs.Validate(); err != nil {
		t.Fatalf("the upgrade-case genesis was refused: %v", err)
	}
	if gs.RewardPool != nil {
		t.Fatal("a genesis without reward_pool decoded with one; InitGenesis would write a record the history never had")
	}
	if DefaultGenesisState().RewardPool == nil || !DefaultGenesisState().RewardPool.Initialized {
		t.Fatal("a chain born on this binary must start with the pool established")
	}
}

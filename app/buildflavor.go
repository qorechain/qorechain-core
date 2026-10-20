package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cast"

	"github.com/cosmos/cosmos-sdk/client/flags"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
)

// FlagCommunityUnenforcedPQC lets a community build start on a public network
// anyway. The operator takes on the consequences described at CheckBuildFlavor.
const FlagCommunityUnenforcedPQC = "community-unenforced-pqc"

// BuildFlavor names the flavour for logs.
func BuildFlavor() string {
	if FullBuild {
		return "full"
	}
	return "community"
}

// NodeChainID returns the Cosmos chain-id this node runs: the --chain-id flag
// when given, else the chain_id of the node home's genesis.
func NodeChainID(appOpts servertypes.AppOptions) string {
	if id := cast.ToString(appOpts.Get(flags.FlagChainID)); id != "" {
		return id
	}
	return genesisChainID(cast.ToString(appOpts.Get(flags.FlagHome)))
}

// publicGenesisTimes pins each public network to the genesis_time of its real
// genesis. A chain-id alone is not enough: local devnets (docker compose,
// init-testnet.sh) reuse "qorechain-diana" with a freshly generated genesis.
var publicGenesisTimes = map[string]string{
	"qorechain-vladi": "2026-06-29T17:03:09.395855717Z",
	"qorechain-diana": "2026-07-01T20:28:30.435519983Z",
}

// NodeGenesisTime returns the genesis_time of the node home's genesis, or ""
// when it cannot be read.
func NodeGenesisTime(appOpts servertypes.AppOptions) string {
	home := cast.ToString(appOpts.Get(flags.FlagHome))
	if home == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, "config", "genesis.json"))
	if err != nil {
		return ""
	}
	var g struct {
		GenesisTime string `json:"genesis_time"`
	}
	if json.Unmarshal(b, &g) != nil {
		return ""
	}
	return g.GenesisTime
}

// IsPublicNetwork reports whether the node runs one of QoreChain's public
// networks (mainnet qorechain-vladi, testnet qorechain-diana): its chain-id is
// a public one and, when the genesis is readable, its genesis_time is that
// network's. An unreadable genesis falls back to the chain-id alone.
func IsPublicNetwork(chainID, genesisTime string) bool {
	want, ok := publicGenesisTimes[chainID]
	if !ok {
		return false
	}
	if genesisTime == "" {
		return true
	}
	a, err1 := time.Parse(time.RFC3339Nano, genesisTime)
	b, err2 := time.Parse(time.RFC3339Nano, want)
	if err1 != nil || err2 != nil {
		return genesisTime == want
	}
	return a.Equal(b)
}

// CheckBuildFlavor refuses to start a community build on a public network
// unless the operator passed --community-unenforced-pqc (QSR-2026-0044,
// QSR-2026-0074).
//
// The community build carries the PQC ante stub, which passes every
// transaction through unverified, and none of the full overlay's keepers or
// dependency patches. On a public network such a node admits transactions to
// its mempool and answers simulate as if no PQC signature were required, and
// it cannot replay the network's blocks with the logic the validators ran, so
// its state can diverge without warning. Public networks are served by the
// release binary (download.qore.host/<net>/latest.json). The check is
// node-local: it only decides whether this process starts.
func CheckBuildFlavor(chainID, genesisTime string, override bool) error {
	if FullBuild || !IsPublicNetwork(chainID, genesisTime) || override {
		return nil
	}
	return fmt.Errorf("refusing to start: this is a community build (no PQC verification, "+
		"no full overlay) and chain %s is a public QoreChain network that it cannot follow "+
		"faithfully; run the release binary from download.qore.host, or pass --%s to start "+
		"anyway at your own risk", chainID, FlagCommunityUnenforcedPQC)
}

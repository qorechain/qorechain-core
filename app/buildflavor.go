package app

import (
	"fmt"

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

// IsPublicNetwork reports whether chainID is one of QoreChain's public
// networks (mainnet qorechain-vladi, testnet qorechain-diana).
func IsPublicNetwork(chainID string) bool {
	_, ok := knownEVMChainIDs[chainID]
	return ok
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
func CheckBuildFlavor(chainID string, override bool) error {
	if FullBuild || !IsPublicNetwork(chainID) || override {
		return nil
	}
	return fmt.Errorf("refusing to start: this is a community build (no PQC verification, "+
		"no full overlay) and chain %s is a public QoreChain network that it cannot follow "+
		"faithfully; run the release binary from download.qore.host, or pass --%s to start "+
		"anyway at your own risk", chainID, FlagCommunityUnenforcedPQC)
}
